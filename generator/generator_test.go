package generator

import (
	"context"
	"io"
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/klingtnet/static-site-generator/generator/renderer"
	"github.com/klingtnet/static-site-generator/internal/testutils"
	"github.com/klingtnet/static-site-generator/slug"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
)

type memoryStorage struct {
	t     *testing.T
	lock  sync.Mutex
	memFS fstest.MapFS
}

func (ms *memoryStorage) Store(_ context.Context, name string, content io.Reader) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return err
	}

	ms.lock.Lock()
	ms.memFS[name] = &fstest.MapFile{
		Data:    data,
		ModTime: time.Now(),
	}
	ms.lock.Unlock()

	return nil
}

func TestGenerator(t *testing.T) {
	// Idea:
	// If this setup is used multiple times inside tests then
	// the generator setup could be moved into a testutil function
	// with the following signature:
	// NewTestGenerator(*testing.T, sourceFS, staticFS fs.FS, storage Storage)
	config := &Config{
		Author:  "Andreas Linz",
		BaseURL: "https://klingt.net",
	}
	contentFS := testutils.NewTestContentFS(t)
	memStor := &memoryStorage{t: t, memFS: make(fstest.MapFS)}
	slugifier := slug.NewSlugifier('-')
	templates := renderer.NewTemplates(
		config.Author,
		config.BaseURL,
		slugifier,
		DefaultTemplateFS(),
	)
	renderer := renderer.NewMarkdown(
		goldmark.New(goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.Footnote)),
		templates,
	)

	generator := New(config, contentFS, nil, memStor, slugifier, renderer)
	err := generator.Run(context.Background())
	require.NoError(t, err)

	files, folders := []string{}, []string{}
	err = fs.WalkDir(memStor.memFS, ".", func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)

		if d.IsDir() {
			folders = append(folders, path)
		} else {
			files = append(files, path)
		}

		return nil
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		".",
		"blog",
		"files",
		"static",
	}, folders)
	require.ElementsMatch(t, []string{
		"about.html",
		"index.html",
		"files/random.txt",
		"static/base.css",
		"blog/feed.rss",
		"blog/index.html",
		"blog/first-article.html",
		"blog/second-article.html",
	}, files)

	// TODO:
	// Find an efficient and reliable way to verify the generated files' contents.
	// Comparing against a set of golden files or doing a hash comparison of the contents.
	// Golden files had the advantage of being easy to edit and review but come with the disadvantage of being tedious to maintain.
	// Hashes are opaque but easy to generate and maintain.
}
