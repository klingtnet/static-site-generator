package generator

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"testing"

	"github.com/klingtnet/static-site-generator/frontmatter"
	"github.com/klingtnet/static-site-generator/slug"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark/parser"
	goldmarkRenderer "github.com/yuin/goldmark/renderer"
)

type Nopdown struct {
	t *testing.T
}

func (*Nopdown) Convert(source []byte, writer io.Writer, opts ...parser.ParseOption) error {
	_, err := writer.Write(source)
	return err
}

func (*Nopdown) Parser() parser.Parser {
	return nil
}

func (*Nopdown) SetParser(parser.Parser) {

}

func (*Nopdown) Renderer() goldmarkRenderer.Renderer {
	return nil
}

func (*Nopdown) SetRenderer(goldmarkRenderer.Renderer) {

}

func TestRenderer(t *testing.T) {
	ts := &Templates{
		Page: template.Must(template.New("base.gohtml").Parse("{{ .Title }}\n{{ .Content }}")),
		List: template.Must(template.New("base.gohtml").Parse("{{ .Title }}\n{{ range $_, $page := .Content.Pages }}{{$page.FM.Title}}{{ end }}")),
	}
	renderer := NewRenderer(&Nopdown{t}, ts)
	require.NotNil(t, renderer)

	buf := bytes.NewBuffer(nil)
	err := renderer.Page(context.Background(), buf, &Library{}, Page{Markdown: []byte(`Some content.`), FM: Frontmatter{Title: "Test Page"}})
	require.NoError(t, err)
	require.Equal(t, "Test Page\nSome content.", buf.String())

	buf.Reset()
	err = renderer.List(context.Background(), buf, &Library{Pages: map[string]Page{"articles/1.md": {FM: Frontmatter{CreatedAt: frontmatter.NewSimpleDate(2021, 07, 11), Title: "A"}}, "articles/2.md": {FM: Frontmatter{CreatedAt: frontmatter.NewSimpleDate(2021, 07, 12), Title: "B"}}}}, "articles")
	require.NoError(t, err)
	// List of articles should be sorted by date of creation descending.
	require.Equal(t, "Articles\nBA", buf.String())
}

func TestAbsLink(t *testing.T) {
	tCases := []struct {
		name     string
		path     string
		expected string
	}{
		{"dir", "articles", "https://john.doe/articles"},
		{"rooted", "articles/some-article.html", "https://john.doe/articles/some-article.html"},
		{"relative", "./articles/some-article.html", "https://john.doe/articles/some-article.html"},
		{"path-traversal", "../../etc/passwd", "https://john.doe/etc/passwd"},
	}

	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			require.Equal(t, tCase.expected, AbsLink("https://john.doe", tCase.path))
		})
	}
}

func TestReplaceExtension(t *testing.T) {
	tCases := []struct {
		name            string
		path, extension string
		expected        string
	}{
		{"unchanged", "https://john.doe/no/extension", ".doesnotmatter", "https://john.doe/no/extension"},
		{"index.md", "index.md", ".html", "index.html"},
		{"multiple extensions", "file.having.multiple.extensions", ".exts", "file.having.multiple.exts"},
	}
	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			require.Equal(t, tCase.expected, ReplaceExtension(tCase.path, tCase.extension))
		})
	}
}

func TestPageLink(t *testing.T) {
	tCases := []struct {
		name     string
		page     Page
		expected string
	}{
		{"root-folder", Page{Path: "/about-me.md", FM: Frontmatter{Title: "About Me"}}, "https://john.doe/about-me.html"},
		{"sub-folder", Page{Path: "/articles/my-first-article.md", FM: Frontmatter{Title: "My first article"}}, "https://john.doe/articles/my-first-article.html"},
		{"sub-sub-folder", Page{Path: "/articles/2021/my-first-article.md", FM: Frontmatter{Title: "My first article"}}, "https://john.doe/articles/2021/my-first-article.html"},
	}

	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			require.Equal(t, tCase.expected, PageLink("https://john.doe", slug.NewSlugifier('-'), tCase.page))
		})
	}
}
