package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/klingtnet/static-site-generator/frontmatter"
	"github.com/klingtnet/static-site-generator/generator/model"
	"github.com/klingtnet/static-site-generator/generator/renderer"
	"github.com/klingtnet/static-site-generator/slug"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
)

// DiscardStorage implements Storage.
type DiscardStorage struct {
	b    *testing.B
	lock sync.RWMutex
	N    int
}

// Store implements Storage.
func (ds *DiscardStorage) Store(ctx context.Context, name string, content io.Reader) (err error) {
	ds.inc()
	_, err = io.Copy(io.Discard, content)
	return
}

func (ds *DiscardStorage) reset() {
	ds.lock.Lock()
	ds.N = 0
	ds.lock.Unlock()
}

func (ds *DiscardStorage) inc() {
	ds.lock.Lock()
	ds.N++
	ds.lock.Unlock()
}

func (ds *DiscardStorage) calls() (N int) {
	ds.lock.RLock()
	N = ds.N
	ds.lock.RUnlock()
	return
}

func frontmatterToJson(b *testing.B, fm model.FrontMatter) []byte {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")

	err := enc.Encode(fm)
	if err != nil {
		b.Fatal(err)
	}

	return buf.Bytes()
}

func buildPage(b *testing.B, content string, fm model.FrontMatter) []byte {
	buf := bytes.NewBufferString("```json\n")
	mustWrite := func(s string) {
		_, err := buf.WriteString(s)
		if err != nil {
			b.Fatal(err)
		}
	}

	mustWrite(string(frontmatterToJson(b, fm)))
	mustWrite("\n```\n")
	mustWrite("\n\n")
	mustWrite(content)

	return buf.Bytes()
}

func newBenchContentFS(b *testing.B, depth, pages int) fs.FS {
	contentFS := fstest.MapFS{}
	fm := model.FrontMatter{Author: "John Doe", Title: b.Name(), Description: "A random page used for benchmarking the generator.", CreatedAt: frontmatter.NewSimpleDate(2021, 07, 17), Tags: []string{"generator", "benchmark", "Go"}, Hidden: false}
	pageContent, err := os.ReadFile("../README.md")
	if err != nil {
		b.Fatal(err)
	}

	prefix := ""
	for i := 0; i < pages; i++ {
		if i%(pages/depth) == 0 {
			prefix += "sub/"
		}

		filename := fmt.Sprintf("%spage%05d.md", prefix, i)
		contentFS[filename] = &fstest.MapFile{
			Data: buildPage(b, string(pageContent), fm),
		}
	}

	return contentFS
}

func BenchmarkGenerator(b *testing.B) {
	ds := &DiscardStorage{b, sync.RWMutex{}, 0}
	sl := slug.NewSlugifier('-')
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.Footnote))
	templates := renderer.NewTemplates(b.Name(), "https://does.not.matter", sl, DefaultTemplateFS())
	generator := New(newBenchContentFS(b, 10, 1000), nil, ds, sl, renderer.NewMarkdown(md, templates))

	for _, concurrency := range []int{1, runtime.NumCPU() / 2, runtime.NumCPU(), runtime.NumCPU() * 2} {
		b.Run(fmt.Sprintf("concurrency-%d", concurrency), func(b *testing.B) {
			generator.concurrency = concurrency
			for n := 0; n < b.N; n++ {
				ds.reset()
				err := generator.Run(context.Background())
				if err != nil {
					b.Fatal(err.Error())
				}
				if ds.calls() != 1066 {
					b.Fatalf("not enough pages rendered, expected %d but was %d", 1066, ds.calls())
				}
			}
		})
	}
}

func BenchmarkCopyStaticFiles(b *testing.B) {
	testFS := make(fstest.MapFS)

	rr := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 1000; i++ {
		f := &fstest.MapFile{Data: make([]byte, 1024*1024)}
		_, err := rr.Read(f.Data)
		if err != nil {
			b.Fatal(err.Error())
		}
		testFS[strconv.Itoa(i)+".bin"] = f
	}

	ds := &DiscardStorage{b, sync.RWMutex{}, 0}
	generator := New(nil, testFS, ds, nil, nil)
	for _, concurrency := range []int{1, runtime.NumCPU(), runtime.NumCPU() * 2} {
		b.Run(fmt.Sprintf("concurrency-%d", concurrency), func(b *testing.B) {
			generator.concurrency = concurrency
			for n := 0; n < b.N; n++ {
				ds.reset()
				err := generator.copyStaticFiles(context.Background())
				if err != nil {
					b.Fatal(err.Error())
				}
			}
		})
	}
}
