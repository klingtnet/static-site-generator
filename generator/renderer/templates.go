package renderer

import (
	"html/template"
	"io/fs"
	"path/filepath"

	"github.com/klingtnet/static-site-generator/generator/model"
	"github.com/klingtnet/static-site-generator/slug"
)

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
func NewTemplates(author, baseURL string, slugifier *slug.Slugifier, templateFS fs.FS) *Templates {
	fns := defaultFuncMap(author, baseURL, slugifier)

	return &Templates{
		Page: template.Must(template.New("").Funcs(fns).ParseFS(templateFS, "base.gohtml", "page.gohtml")),
		List: template.Must(template.New("").Funcs(fns).ParseFS(templateFS, "base.gohtml", "list.gohtml")),
	}
}

// PageLink returns a link for the given page using its slugified title as filename.
func PageLink(baseURL string, slugifier *slug.Slugifier, page TemplatePage) string {
	return baseURL + filepath.Join("/", filepath.Dir(page.Path), slugifier.Slugify(page.FM.Title)+".html")
}

// AbsLink returns an absolute representation of the given path.
func AbsLink(baseURL string, path string) string {
	return baseURL + filepath.Clean("/"+path)
}

// ReplaceExtension replaces the extension of path with ext.
// The given path remains unchanged if it does not end with a file extension.
// Note that ext is expected to start with a dot.
func ReplaceExtension(path, ext string) string {
	actual := filepath.Ext(path)
	if actual != "" {
		return path[:len(path)-len(actual)] + ext
	}

	return path
}

func defaultFuncMap(author, baseURL string, slugifier *slug.Slugifier) template.FuncMap {
	return template.FuncMap{
		"pageLink": func(page TemplatePage) string {
			return PageLink(baseURL, slugifier, page)
		},
		"absLink":          func(path string) string { return AbsLink(baseURL, path) },
		"replaceExtension": ReplaceExtension,
	}
}

// TemplateData contains data used to render page templates.
type TemplateData struct {
	Title, Description string
	Content            interface{}
	Menu               []model.MenuEntry
}
