// Package generator implements the static site generator.
package generator

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/klingtnet/static-site-generator/frontmatter"
	"github.com/klingtnet/static-site-generator/slug"
	"golang.org/x/sync/errgroup"
)

func collectArtifacts(ctx context.Context, library *Library, sourceFS fs.FS) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			library.Dirs = append(library.Dirs, path)
		}

		if !d.Type().IsRegular() {
			return nil
		}

		if filepath.Ext(path) != ".md" {
			library.Assets = append(library.Assets, path)
			return nil
		}

		f, err := sourceFS.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		var fm FrontMatter
		err = frontmatter.Read(ctx, f, &fm)
		if err != nil {
			return fmt.Errorf("%q: frontmatter parsing failed: %w", path, err)
		}
		markdown, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("failed to read content: %w", err)
		}

		library.Pages[path] = Page{
			Path:     path,
			FM:       fm,
			Markdown: markdown,
		}

		return nil
	}
}

func buildMenu(ctx context.Context, library *Library) error {
	_, ok := library.Pages["index.md"]
	if !ok {
		return fmt.Errorf("no index.md in source root")
	}

	// The root index.md page is assumed to be the homepage.
	menu := []MenuEntry{{"Home", "index.md"}}

	// Add folders before root pages.
	for _, dir := range library.Dirs {
		if dir == "." {
			continue
		}
		menu = append(menu, MenuEntry{strings.Title(dir), dir})
	}

	rootPages := library.PagesIn(".")
	// Sort root pages alphabetically.
	sort.SliceStable(rootPages, func(i, j int) bool {
		return rootPages[i].FM.Title < rootPages[j].FM.Title
	})
	for _, page := range rootPages {
		if page.FM.Hidden || page.Path == "index.md" {
			continue
		}
		menu = append(menu, MenuEntry{page.FM.Title, page.Path})
	}

	library.Menu = menu

	return nil
}

func initLibrary(ctx context.Context, sourceFS fs.FS) (*Library, error) {
	library := Library{Pages: make(map[string]Page)}

	err := fs.WalkDir(sourceFS, ".", collectArtifacts(ctx, &library, sourceFS))
	if err != nil {
		return nil, err
	}
	sort.Strings(library.Dirs)

	err = buildMenu(ctx, &library)
	if err != nil {
		return nil, fmt.Errorf("building menu failed: %w", err)
	}

	return &library, nil
}

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
	sourceFS, staticFS fs.FS
	stor               Storage
	slugifier          *slug.Slugifier
	renderer           *Renderer
}

func (g *Generator) copyAsset(file string) error {
	src, err := g.sourceFS.Open(file)
	if err != nil {
		return err
	}
	defer src.Close()

	return g.stor.Store(context.TODO(), file, src)
}

func (g *Generator) copyAssets(files []string) error {
	for _, file := range files {
		err := g.copyAsset(file)
		if err != nil {
			return fmt.Errorf("copying asset %q failed: %w", file, err)
		}
	}

	return nil
}

func (g *Generator) copyStaticFiles() error {
	return fs.WalkDir(g.staticFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		src, err := g.staticFS.Open(path)
		if err != nil {
			return err
		}
		return g.stor.Store(context.TODO(), path, src)
	})
}

// TemplateData contains data used to render page templates.
type TemplateData struct {
	Title, Description string
	Content            interface{}
	Menu               []MenuEntry
}

func (g *Generator) renderListPage(library *Library, dir string) error {
	buf := bytes.NewBuffer(make([]byte, 0, 8192))
	err := g.renderer.List(context.TODO(), buf, library, dir)
	if err != nil {
		return err
	}

	return g.stor.Store(context.TODO(), filepath.Join(dir, "index.html"), buf)
}

func (g *Generator) renderPage(ctx context.Context, library *Library, pageCh <-chan Page) error {
	for page := range pageCh {
		var dest string
		if strings.HasSuffix(page.Path, "index.md") {
			dest = filepath.Join(filepath.Dir(page.Path), "index.html")
		} else {
			dest = filepath.Join(filepath.Dir(page.Path), g.slugifier.Slugify(page.FM.Title)) + ".html"
		}

		buf := bytes.NewBuffer(make([]byte, 0, 8192))
		err := g.renderer.Page(ctx, buf, library, page)
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

func (g *Generator) copyStatic(ctx context.Context, library *Library) error {
	err := g.copyAssets(library.Assets)
	if err != nil {
		return fmt.Errorf("copying assets failed: %w", err)
	}

	if g.staticFS != nil {
		err = g.copyStaticFiles()
		if err != nil {
			return fmt.Errorf("copying static files failed: %w", err)
		}
	}

	return nil
}

func (g *Generator) render(ctx context.Context, library *Library) error {
	// Render list pages sequentially since there are commonly not that many of them.
	for _, dir := range library.Dirs {
		// List pages are only rendered if the folder does not contain a index.md.
		_, err := fs.Stat(g.sourceFS, filepath.Clean(filepath.Join(dir, "index.md")))
		if os.IsNotExist(err) {
			err = g.renderListPage(library, dir)
			if err != nil {
				return fmt.Errorf("rendering list page for %q failed: %w", dir, err)
			}
		}
	}

	N := runtime.NumCPU()
	pageCh := make(chan Page, N)
	go func() {
		defer close(pageCh)
		for _, page := range library.Pages {
			pageCh <- page
		}
	}()

	eg, ctx := errgroup.WithContext(ctx)
	for i := 0; i < N; i++ {
		eg.Go(func() error { return g.renderPage(ctx, library, pageCh) })
	}

	return eg.Wait()
}

// Run generates the website.
func (g *Generator) Run(ctx context.Context) error {
	library, err := initLibrary(ctx, g.sourceFS)
	if err != nil {
		return fmt.Errorf("library initialization failed: %w", err)
	}

	// TODO: parallelize copying
	err = g.copyStatic(ctx, library)
	if err != nil {
		return fmt.Errorf("copying static content failed: %w", err)
	}

	err = g.render(ctx, library)
	if err != nil {
		return fmt.Errorf("rendering failed: %w", err)
	}
	return nil
}

// New returns a new Generator instance.
func New(sourceFS, staticFS fs.FS, stor Storage, slugifier *slug.Slugifier, renderer *Renderer) *Generator {
	if staticFS == nil {
		staticFS = defaultStaticFS
	}

	return &Generator{
		sourceFS:  sourceFS,
		staticFS:  staticFS,
		stor:      stor,
		slugifier: slugifier,
		renderer:  renderer,
	}
}
