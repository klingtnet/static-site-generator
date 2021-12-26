// Package generator implements the static site generator.
package generator

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klingtnet/static-site-generator/generator/model"
	"github.com/klingtnet/static-site-generator/generator/renderer"
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

	pathCh := make(chan string, g.concurrency)
	eg, ctx := errgroup.WithContext(ctx)

	for i := 0; i < g.concurrency; i++ {
		eg.Go(func() error {
			for path := range pathCh {
				err := cp(ctx, path)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	eg.Go(func() error {
		defer close(pathCh)
		return fs.WalkDir(g.staticFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			pathCh <- path
			return nil
		})
	})
	return eg.Wait()
}

func (g *Generator) renderListPage(content model.Tree, siteMenu []model.MenuEntry) error {
	buf := bytes.NewBuffer(make([]byte, 0, 8192))
	err := g.renderer.List(context.TODO(), buf, content, siteMenu)
	if err != nil {
		return err
	}

	return g.stor.Store(context.TODO(), filepath.Join(content.Path(), "index.html"), buf)
}

func (g *Generator) renderPage(ctx context.Context, content model.Tree, siteMenu []model.MenuEntry, pageCh <-chan *model.Page) error {
	for page := range pageCh {
		var dest string
		if strings.HasSuffix(page.Path(), "index.md") {
			dest = filepath.Join(filepath.Dir(page.Path()), "index.html")
		} else {
			dest = filepath.Join(filepath.Dir(page.Path()), g.slugifier.Slugify(page.Frontmatter().Title)) + ".html"
		}

		buf := bytes.NewBuffer(make([]byte, 0, 8192))
		err := g.renderer.Page(ctx, buf, renderer.NewTemplatePage(page), siteMenu)
		if err != nil {
			return err
		}

		err = g.stor.Store(ctx, dest, buf)
		if err != nil {
			return err
		}
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

func (g *Generator) traverseTreeForListPages(tree model.Tree, siteMenu []model.MenuEntry) error {
	content, ok := tree.(*model.ContentTree)
	if !ok {
		return nil
	}

	var containsIndexMD, containsPages bool
	for _, child := range tree.Children() {
		switch el := child.(type) {
		case *model.ContentTree:
			err := g.traverseTreeForListPages(el, siteMenu)
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
		return g.renderListPage(content, siteMenu)
	}

	return nil
}

func (g *Generator) render(ctx context.Context, content *model.ContentTree) error {
	err := content.Walk(func(tree model.Tree) error {
		return g.traverseTreeForListPages(tree, model.Menu(content))
	})
	if err != nil {
		return fmt.Errorf("rendering list pages failed: %w", err)
	}

	pageCh := make(chan *model.Page, g.concurrency)
	go func() {
		defer close(pageCh)
		_ = content.Walk(func(tree model.Tree) error {
			page, ok := tree.(*model.Page)
			if ok {
				pageCh <- page
			}

			return nil
		})
	}()

	eg, ctx := errgroup.WithContext(ctx)
	for i := 0; i < g.concurrency; i++ {
		eg.Go(func() error { return g.renderPage(ctx, content, model.Menu(content), pageCh) })
	}

	return eg.Wait()
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
func New(sourceFS, staticFS fs.FS, stor Storage, slugifier *slug.Slugifier, renderer renderer.Renderer) *Generator {
	if staticFS == nil {
		staticFS = defaultStaticFS
	}

	return &Generator{
		concurrency: runtime.NumCPU(),
		sourceFS:    sourceFS,
		staticFS:    staticFS,
		stor:        stor,
		slugifier:   slugifier,
		renderer:    renderer,
	}
}
