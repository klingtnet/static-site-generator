package generator

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/klingtnet/static-site-generator/slug"
	"github.com/yuin/goldmark"
)

type Renderer struct {
	md        goldmark.Markdown
	templates *Templates
}

func NewRenderer(md goldmark.Markdown, templates *Templates) *Renderer {
	return &Renderer{
		md:        md,
		templates: templates,
	}
}

// Templates are used by the Renderer to render HTML pages.
type Templates struct {
	// Page is a template for simple website pages.
	Page *template.Template
	// List is a template for list pages, e.g. a list of all blog articles.
	List *template.Template
}

// NewTemplates parses templates from the given fs.FS and provides a set of default template functions.
// The template folder is expected to contain three files, base.gohtml, page.gohtml and list.gohtml, where
// base.gohtml will be shared by both, the page and list template.
//
// TODO: export template functions and document them.
func NewTemplates(author, baseURL string, slugifier *slug.Slugifier, templateFS fs.FS) *Templates {
	fns := defaultFuncMap(author, baseURL, slugifier)
	return &Templates{
		Page: template.Must(template.New("").Funcs(fns).ParseFS(templateFS, "base.gohtml", "page.gohtml")),
		List: template.Must(template.New("").Funcs(fns).ParseFS(templateFS, "base.gohtml", "list.gohtml")),
	}
}

func pageLink(baseURL string, slugifier *slug.Slugifier, page Page) string {
	return baseURL + filepath.Join("/", filepath.Dir(page.Path), slugifier.Slugify(page.FM.Title)+".html")
}

func absLink(baseURL string, path string) string {
	return baseURL + filepath.Clean("/"+path)
}

func replaceExtension(path, ext string) string {
	actual := filepath.Ext(path)
	if actual != "" {
		return path[:len(path)-len(actual)] + ext
	}
	return path
}

func defaultFuncMap(author, baseURL string, slugifier *slug.Slugifier) template.FuncMap {
	return template.FuncMap{
		"pageLink": func(page Page) string {
			return pageLink(baseURL, slugifier, page)
		},
		"absLink":          func(path string) string { return absLink(baseURL, path) },
		"replaceExtension": replaceExtension,
	}
}

func (r *Renderer) Page(ctx context.Context, w io.Writer, library *Library, page Page) error {
	buf := bytes.NewBuffer(nil)
	err := r.md.Convert(page.Markdown, buf)
	if err != nil {
		return err
	}

	data := TemplateData{
		page.FM.Title, page.FM.Description,
		template.HTML(buf.String()),
		library.Menu,
	}
	return r.templates.Page.ExecuteTemplate(w, "base.gohtml", data)
}

func (r *Renderer) List(ctx context.Context, w io.Writer, library *Library, dir string) error {
	data := TemplateData{
		strings.Title(dir), "List of " + dir,
		struct {
			Pages []Page
			Dir   string
		}{
			library.PagesIn(dir),
			dir,
		},
		library.Menu,
	}
	return r.templates.List.ExecuteTemplate(w, "base.gohtml", data)
}
