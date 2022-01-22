package renderer

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/klingtnet/static-site-generator/generator/model"
	"github.com/yuin/goldmark"
)

type Renderer interface {
	Page(context.Context, io.Writer, TemplatePage, []model.MenuEntry) error
	FeedPage(context.Context, io.Writer, TemplatePage) error
	List(context.Context, io.Writer, model.Tree, []model.MenuEntry) error
}

// Markdown renders markdown pages to HTML websites.
type Markdown struct {
	md        goldmark.Markdown
	templates *Templates
}

// NewMarkdown returns an instantiated markdown renderer.
func NewMarkdown(md goldmark.Markdown, templates *Templates) *Markdown {
	return &Markdown{
		md:        md,
		templates: templates,
	}
}

// Page renders a single page.
func (m *Markdown) Page(
	ctx context.Context,
	w io.Writer,
	page TemplatePage,
	siteMenu []model.MenuEntry,
) error {
	buf := bytes.NewBuffer(nil)
	err := m.md.Convert(page.Markdown, buf)
	if err != nil {
		return err
	}

	data := TemplateData{
		page.FM.Title, page.FM.Description,
		template.HTML(buf.String()),
		siteMenu,
	}

	return m.templates.Page.ExecuteTemplate(w, "base.gohtml", data)
}

// FeedPage renders a page for use in a feed.
func (m *Markdown) FeedPage(ctx context.Context, w io.Writer, page TemplatePage) error {
	buf := bytes.NewBuffer(nil)
	err := m.md.Convert(page.Markdown, buf)
	if err != nil {
		return err
	}

	data := TemplateData{
		page.FM.Title, page.FM.Description,
		template.HTML(buf.String()),
		nil,
	}

	return m.templates.FeedPage.ExecuteTemplate(w, "feed.gohtml", data)
}

type TemplatePage struct {
	Path     string
	FM       model.FrontMatter
	Markdown []byte
}

func NewTemplatePage(page *model.Page) TemplatePage {
	return TemplatePage{
		Path:     page.Path(),
		FM:       *page.Frontmatter(),
		Markdown: page.Content(),
	}
}

// List renders a list, or directory overview, page.
func (m *Markdown) List(
	ctx context.Context,
	w io.Writer,
	content model.Tree,
	siteMenu []model.MenuEntry,
) error {
	var pages []TemplatePage
	for _, child := range content.Children() {
		page, ok := child.(*model.Page)
		if ok && !page.Frontmatter().Hidden {
			pages = append(pages, NewTemplatePage(page))
		}
	}

	// Sort pages by Date descending
	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].FM.CreatedAt == nil || pages[j].FM.CreatedAt == nil {
			return false
		}

		return time.Time(*pages[i].FM.CreatedAt).After(time.Time(*pages[j].FM.CreatedAt))
	})

	data := TemplateData{
		strings.Title(content.Name()), "List of " + content.Name(),
		struct {
			Pages []TemplatePage
			Dir   string
		}{
			pages,
			content.Path(),
		},
		siteMenu,
	}

	return m.templates.List.ExecuteTemplate(w, "base.gohtml", data)
}
