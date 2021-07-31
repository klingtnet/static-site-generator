package generator

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/klingtnet/static-site-generator/frontmatter"
	"github.com/klingtnet/static-site-generator/slug"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
)

// DiscardStorage implements Storage.
type DiscardStorage struct {
	b *testing.B
}

// Store implements Storage.
func (ds *DiscardStorage) Store(ctx context.Context, name string, content io.Reader) (err error) {
	_, err = io.Copy(io.Discard, content)
	return
}

func initSourceDir(b *testing.B, pages, directories int, tempDir string) {
	content, err := os.ReadFile("../README.md")
	if err != nil {
		b.Fatal(err.Error())
	}

	writePage := func(filename string) {
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			b.Fatal(err.Error())
		}
		defer f.Close()

		var sb strings.Builder
		_, _ = sb.Write([]byte("```json\n"))
		fm := FrontMatter{Author: "John Doe", Title: b.Name(), Description: "A random page used for benchmarking the generator.", CreatedAt: frontmatter.NewSimpleDate(2021, 07, 17), Tags: []string{"generator", "benchmark", "Go"}, Hidden: false}
		err = json.NewEncoder(&sb).Encode(fm)
		if err != nil {
			b.Fatal(err.Error())
		}
		_, _ = sb.Write([]byte("```\n"))
		_, _ = sb.Write(content)

		_, err = f.Write([]byte(sb.String()))
		if err != nil {
			b.Fatal(err)
		}
	}

	writePage(filepath.Join(tempDir, "index.md"))
	d := 0
	for i := 1; i < pages; i++ {
		var dir string
		if d == 0 {
			// write into content root
			dir = tempDir
		} else {
			dir = filepath.Join(tempDir, "dir"+strconv.Itoa(d))
		}
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			b.Fatal(err.Error())
		}
		writePage(filepath.Join(dir, "page"+strconv.Itoa(i)+".md"))
		if i%(pages/directories) == 0 {
			d++
			// new directory
		}
	}
}

func BenchmarkGenerator(b *testing.B) {
	tempDir := b.TempDir()
	initSourceDir(b, 1000, 10, tempDir)
	sourceFS := os.DirFS(tempDir)
	sl := slug.NewSlugifier('-')
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.Footnote))
	templates := NewTemplates(b.Name(), "https://does.not.matter", sl, DefaultTemplateFS())
	generator := New(sourceFS, nil, &DiscardStorage{b}, sl, NewRenderer(md, templates))

	for n := 0; n < b.N; n++ {
		err := generator.Run(context.Background())
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}
