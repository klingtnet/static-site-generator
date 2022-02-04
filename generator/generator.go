// Package generator implements the static site generator.
package generator

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/feeds"
	"github.com/klingtnet/static-site-generator/generator/model"
	"github.com/klingtnet/static-site-generator/generator/renderer"
	"github.com/klingtnet/static-site-generator/internal/distribute"
	"github.com/klingtnet/static-site-generator/slug"
	"golang.org/x/sync/errgroup"
)

//go:embed templates/*.gohtml
var defaultTemplateFS embed.FS

func DefaultTemplateFS() fs.FS {
	templateFS, err := fs.Sub(defaultTemplateFS, "templates")
	if err != nil {
		panic(err)
	}
	return templateFS
}

//go:embed static
var defaultStaticFS embed.FS

type Generator struct {
	concurrency        int
	sourceFS, staticFS fs.FS
	stor               Storage
	slugifier          *slug.Slugifier
	renderer           renderer.Renderer
	config             *Config
	bufPool            *sync.Pool
}

func (g *Generator) copyStaticFiles(ctx context.Context) error {
	cp := func(ctx context.Context, path string) error {
		src, err := g.staticFS.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		return g.stor.Store(ctx, path, src)
	}

	return distribute.OneToN(
		ctx,
		func(ctx context.Context, dataCh chan<- interface{}) error {
			return fs.WalkDir(g.staticFS, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					return nil
				}

				dataCh <- path
				return nil
			})
		},
		func(ctx context.Context, data interface{}) error {
			path := data.(string)
			return cp(ctx, path)
		},
		g.concurrency,
	)
}

func (g *Generator) renderListPage(
	ctx context.Context,
	content model.Tree,
	siteMenu []model.MenuEntry,
) error {
	buf := g.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer g.bufPool.Put(buf)

	err := g.renderer.List(ctx, buf, content, siteMenu)
	if err != nil {
		return err
	}

	return g.stor.Store(ctx, filepath.Join(content.Path(), "index.html"), buf)
}

func (g *Generator) renderFeed(ctx context.Context, content model.Tree) error {
	if content.Path() == "." {
		// Ignore root dir.
		return nil
	}

	feed, err := g.buildFeed(ctx, content)
	if err != nil {
		return err
	}

	pr, pw := io.Pipe()
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		defer pw.Close()
		return feed.WriteRss(pw)
	})
	eg.Go(func() error {
		defer pr.Close()
		return g.stor.Store(ctx, filepath.Join(content.Path(), "feed.rss"), pr)
	})

	return eg.Wait()
}

func (g *Generator) buildFeed(ctx context.Context, content model.Tree) (*feeds.Feed, error) {
	feed := &feeds.Feed{
		Title:   content.Name(),
		Link:    &feeds.Link{Href: renderer.AbsLink(g.config.BaseURL, content.Path())},
		Author:  &feeds.Author{Name: g.config.Author},
		Created: time.Now(),
	}
	feedLock := sync.RWMutex{}

	concurrency := len(content.Children())
	if g.concurrency < concurrency {
		concurrency = g.concurrency
	}
	err := distribute.OneToN(
		ctx,
		func(ctx context.Context, dataCh chan<- interface{}) error {
			for _, child := range content.Children() {
				page, ok := child.(*model.Page)
				if ok && !page.Frontmatter().Hidden {
					dataCh <- page
				}
			}

			return nil
		},
		func(ctx context.Context, data interface{}) error {
			page := data.(*model.Page)
			item, err := g.renderFeedPage(ctx, page)
			if err != nil {
				return err
			}
			feedLock.Lock()
			feed.Items = append(feed.Items, item)
			feedLock.Unlock()

			return nil
		},
		concurrency,
	)

	return feed, err
}

func (g *Generator) renderFeedPage(ctx context.Context, page *model.Page) (*feeds.Item, error) {
	buf := g.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer g.bufPool.Put(buf)

	templatePage := renderer.NewTemplatePage(page)
	err := g.renderer.FeedPage(ctx, buf, templatePage)
	if err != nil {
		return nil, err
	}

	return &feeds.Item{
		Title:       page.Frontmatter().Title,
		Description: page.Frontmatter().Description,
		Author:      &feeds.Author{Name: page.Frontmatter().Author},
		Link: &feeds.Link{
			Href: renderer.PageLink(g.config.BaseURL, g.slugifier, templatePage),
		},
		Created: time.Now(),
		Content: buf.String(),
	}, nil
}

func (g *Generator) renderPage(
	ctx context.Context,
	content model.Tree,
	siteMenu []model.MenuEntry,
	page *model.Page,
) error {
	var dest string
	if strings.HasSuffix(page.Path(), "index.md") {
		dest = filepath.Join(filepath.Dir(page.Path()), "index.html")
	} else {
		dest = filepath.Join(filepath.Dir(page.Path()), g.slugifier.Slugify(page.Frontmatter().Title)) + ".html"
	}

	buf := g.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer g.bufPool.Put(buf)

	err := g.renderer.Page(ctx, buf, renderer.NewTemplatePage(page), siteMenu)
	if err != nil {
		return err
	}

	err = g.stor.Store(ctx, dest, buf)
	if err != nil {
		return err
	}
	return nil
}

func (g *Generator) copyStatic(ctx context.Context, content *model.ContentTree) error {
	err := content.Walk(func(tree model.Tree) error {
		file, ok := tree.(*model.File)
		if !ok {
			return nil
		}
		f, err := g.sourceFS.Open(file.Path())
		if err != nil {
			return err
		}
		defer f.Close()

		return g.stor.Store(ctx, file.Path(), f)
	})
	if err != nil {
		return fmt.Errorf("copying asset files failed: %w", err)
	}

	if g.staticFS != nil {
		err = g.copyStaticFiles(ctx)
		if err != nil {
			return fmt.Errorf("copying static files failed: %w", err)
		}
	}

	return nil
}

func (g *Generator) collectContentDirs(
	ctx context.Context,
	tree model.Tree,
	resultCh chan<- interface{},
) error {
	content, ok := tree.(*model.ContentTree)
	if !ok {
		return nil
	}

	var containsIndexMD, containsPages bool
	for _, child := range tree.Children() {
		switch el := child.(type) {
		case *model.ContentTree:
			err := g.collectContentDirs(ctx, el, resultCh)
			if err != nil {
				return err
			}
		case *model.Page:
			ext := filepath.Ext(el.Path())
			if ext != ".md" {
				continue
			}

			if el.Name() == "index.md" {
				containsIndexMD = true
			} else {
				containsPages = true
			}
		}
	}
	if containsPages && !containsIndexMD {
		resultCh <- content
	}

	return nil
}

func (g *Generator) render(ctx context.Context, content *model.ContentTree) error {
	rootMenu := model.Menu(content)

	err := distribute.OneToN(
		ctx,
		func(ctx context.Context, dataCh chan<- interface{}) error {
			err := content.Walk(func(tree model.Tree) error {
				return g.collectContentDirs(ctx, tree, dataCh)
			})
			if err != nil {
				return fmt.Errorf("rendering list pages failed: %w", err)
			}

			return nil
		},
		func(ctx context.Context, data interface{}) error {
			content := data.(*model.ContentTree)
			eg, ctx := errgroup.WithContext(ctx)
			eg.Go(func() error {
				err := g.renderFeed(ctx, content)
				if err != nil {
					return fmt.Errorf("feed rendering failed: %w", err)
				}

				return nil
			})
			eg.Go(func() error {
				return g.renderListPage(ctx, content, rootMenu)
			})

			return eg.Wait()
		},
		g.concurrency,
	)
	if err != nil {
		return err
	}

	return distribute.OneToN(
		ctx,
		func(ctx context.Context, dataCh chan<- interface{}) error {
			return content.Walk(func(tree model.Tree) error {
				page, ok := tree.(*model.Page)
				if ok {
					dataCh <- page
				}

				return nil
			})
		},
		func(ctx context.Context, data interface{}) error {
			page := data.(*model.Page)
			return g.renderPage(ctx, content, rootMenu, page)
		},
		g.concurrency,
	)
}

// Run generates the website.
func (g *Generator) Run(ctx context.Context) error {
	content, err := model.NewContentTree(ctx, g.sourceFS, ".")
	if err != nil {
		return fmt.Errorf("library initialization failed: %w", err)
	}

	err = g.copyStatic(ctx, content)
	if err != nil {
		return fmt.Errorf("copying static content failed: %w", err)
	}

	err = g.render(ctx, content)
	if err != nil {
		return fmt.Errorf("rendering failed: %w", err)
	}
	return nil
}

// New returns a new Generator instance.
func New(
	config *Config,
	sourceFS, staticFS fs.FS,
	stor Storage,
	slugifier *slug.Slugifier,
	renderer renderer.Renderer,
) *Generator {
	if staticFS == nil {
		staticFS = defaultStaticFS
	}

	return &Generator{
		config:      config,
		concurrency: runtime.NumCPU(),
		sourceFS:    sourceFS,
		staticFS:    staticFS,
		stor:        stor,
		slugifier:   slugifier,
		renderer:    renderer,
		bufPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 2<<14))
			},
		},
	}
}
