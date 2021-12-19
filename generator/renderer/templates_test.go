package renderer

import (
	"testing"

	"github.com/klingtnet/static-site-generator/generator/model"
	"github.com/klingtnet/static-site-generator/slug"
	"github.com/stretchr/testify/require"
)

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
		page     TemplatePage
		expected string
	}{
		{"root-folder", TemplatePage{Path: "/about-me.md", FM: model.FrontMatter{Title: "About Me"}}, "https://john.doe/about-me.html"},
		{"sub-folder", TemplatePage{Path: "/articles/my-first-article.md", FM: model.FrontMatter{Title: "My first article"}}, "https://john.doe/articles/my-first-article.html"},
		{"sub-sub-folder", TemplatePage{Path: "/articles/2021/my-first-article.md", FM: model.FrontMatter{Title: "My first article"}}, "https://john.doe/articles/2021/my-first-article.html"},
	}

	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			require.Equal(t, tCase.expected, PageLink("https://john.doe", slug.NewSlugifier('-'), tCase.page))
		})
	}
}
