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
	"sort"
	"strings"
	"time"

	"github.com/klingtnet/static-site-generator/frontmatter"
	"github.com/klingtnet/static-site-generator/slug"
)

type Library struct {
	Pages  map[string]Page
	Assets []string
	Dirs   []string
	Menu   []MenuEntry
}

func (l *Library) PagesIn(dir string) []Page {
	var pages []Page
	for path, page := range l.Pages {
		if filepath.Base(filepath.Dir(filepath.Clean(path))) == dir {
			pages = append(pages, page)
		}
	}
	// Sort pages by Date descending
	sort.SliceStable(pages, func(i, j int) bool {
		return time.Time(pages[i].FM.CreatedAt).After(time.Time(pages[j].FM.CreatedAt))
	})

	return pages
}

type Page struct {
	Path     string
	FM       FrontMatter
	Markdown []byte
}

type FrontMatter struct {
	Author      string                 `json:"author"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	CreatedAt   frontmatter.SimpleDate `json:"created_at"`
	Tags        []string               `json:"tags"`
}

type MenuEntry struct {
	Name string
	Path string
}

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
		if page.Path == "index.md" {
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
var DefaultTemplateFS embed.FS

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

func (g *Generator) renderPage(library *Library, page Page) error {
	var dest string
	if strings.HasSuffix(page.Path, "index.md") {
		dest = filepath.Join(filepath.Dir(page.Path), "index.html")
	} else {
		dest = filepath.Join(filepath.Dir(page.Path), g.slugifier.Slugify(page.FM.Title)) + ".html"
	}

	buf := bytes.NewBuffer(make([]byte, 0, 8192))
	err := g.renderer.Page(context.TODO(), buf, library, page)
	if err != nil {
		return err
	}

	return g.stor.Store(context.TODO(), dest, buf)
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

	for _, page := range library.Pages {
		err := g.renderPage(library, page)
		if err != nil {
			return fmt.Errorf("rendering page %q failed: %w", page.Path, err)
		}
	}

	return nil
}

func (g *Generator) Run(ctx context.Context) error {
	library, err := initLibrary(ctx, g.sourceFS)
	if err != nil {
		return fmt.Errorf("library initialization failed: %w", err)
	}

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

func New(sourceFS, staticFS fs.FS, stor Storage, slugifier *slug.Slugifier, renderer *Renderer) (*Generator, error) {
	if staticFS == nil {
		staticFS = defaultStaticFS
	}

	g := &Generator{
		sourceFS:  sourceFS,
		staticFS:  staticFS,
		stor:      stor,
		slugifier: slugifier,
		renderer:  renderer,
	}

	return g, nil
}
