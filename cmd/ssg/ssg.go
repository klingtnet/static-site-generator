// Package main implements the CLI for ssg, a static site generator.
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/klingtnet/static-site-generator/generator"
	"github.com/klingtnet/static-site-generator/generator/renderer"
	"github.com/klingtnet/static-site-generator/internal/fswatcher"
	"github.com/klingtnet/static-site-generator/slug"
	"github.com/urfave/cli/v2"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

const (
	InternalError = iota + 1
	BadArgument
)

func flagOverride(config *generator.Config, c *cli.Context) {
	if c.String("content") != "" {
		config.ContentDir = c.String("content")
	}
	if c.String("static") != "" {
		config.StaticDir = c.String("static")
	}
	if c.String("output") != "" {
		config.OutputDir = c.String("output")
	}
}

type resources struct {
	sourceFS, staticFS, templateFS fs.FS
}

func initResources(config *generator.Config) (r *resources, err error) {
	r = &resources{sourceFS: os.DirFS(config.ContentDir)}

	if config.StaticDir != "" {
		r.staticFS = os.DirFS(config.StaticDir)
	}

	if config.TemplatesDir != "" {
		r.templateFS = os.DirFS(config.TemplatesDir)
	} else {
		r.templateFS = generator.DefaultTemplateFS()
	}

	return
}

func setup(c *cli.Context) (
	gen *generator.Generator,
	config *generator.Config,
	resources *resources,
	err error,
) {
	config, err = generator.ParseConfigFile(c.String("config"))
	if err != nil {
		err = cli.Exit(fmt.Sprintf("parsing config %q failed: %s", c.String("config"), err.Error()), BadArgument)
		return
	}
	flagOverride(config, c)

	err = config.Validate()
	if err != nil {
		err = cli.Exit(fmt.Sprintf("bad config: %s", err.Error()), BadArgument)
		return
	}

	resources, err = initResources(config)
	if err != nil {
		err = cli.Exit(fmt.Sprintf("bad resources: %s", err.Error()), BadArgument)
		return
	}

	slugifier := slug.NewSlugifier('-')
	templates := renderer.NewTemplates(config.Author, config.BaseURL, slugifier, resources.templateFS)
	storage := generator.NewFileStorage(config.OutputDir)

	markdownOptions := []goldmark.Option{goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.Footnote)}
	if config.EnableUnsafeHTML {
		markdownOptions = append(markdownOptions, goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()))
	}
	renderer := renderer.NewMarkdown(goldmark.New(markdownOptions...), templates)

	gen = generator.New(config, resources.sourceFS, resources.staticFS, storage, slugifier, renderer)

	return
}

func run(c *cli.Context) error {
	generator, _, _, err := setup(c)
	if err != nil {
		return err
	}

	err = generator.Run(c.Context)
	if err != nil {
		return cli.Exit(fmt.Sprintf("generator failed: %s", err.Error()), InternalError)
	}

	return nil
}

func fileHandler(httpDir http.Dir, notFoundPage []byte) http.HandlerFunc {
	fileServer := http.FileServer(httpDir)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := httpDir.Open(r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)

			_, err = io.Copy(w, bytes.NewBuffer(notFoundPage))
			if err != nil {
				log.Println("sending 404 response page failed:", err.Error())
			}
			return
		}
		f.Close()

		fileServer.ServeHTTP(w, r)
	})
}

func runServer(host string, port int, outputDir string) error {
	notFoundPage, err := os.ReadFile(filepath.Join(outputDir, "404.html"))
	if err != nil {
		notFoundPage = []byte(http.StatusText(http.StatusNotFound))
	}

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: fileHandler(http.Dir(outputDir), notFoundPage),
	}
	log.Printf("listening on http://%s", server.Addr)

	return server.ListenAndServe()
}

func liveReload(c *cli.Context) error {
	generator, config, resources, err := setup(c)
	if err != nil {
		return err
	}

	checkInterval := c.Duration("check-interval")
	watchers := []<-chan fswatcher.Result{
		fswatcher.New(resources.sourceFS, time.NewTicker(checkInterval)).Watch(c.Context),
		fswatcher.New(resources.staticFS, time.NewTicker(checkInterval)).Watch(c.Context),
		fswatcher.New(resources.templateFS, time.NewTicker(checkInterval)).Watch(c.Context),
	}

	go func() {
		for {
			err = runServer(c.String("host"), c.Int("port"), config.OutputDir)
			if err != nil {
				log.Printf("server crashed: %s", err.Error())
				panic("exiting")
			}
		}
	}()

	for {
		hasChanged := false

		for _, watcher := range watchers {
			result := <-watcher
			if result.Err != nil {
				return err
			}

			hasChanged = hasChanged || result.HasChanged
		}

		if hasChanged {
			log.Println("something has changed, rebuilding...")

			err = generator.Run(c.Context)
			if err != nil {
				log.Printf("generator failed: %s", err.Error())
			}
		}
	}
}

func main() {
	app := cli.App{
		Name:        "ssg",
		Description: "A opinionated static site generator. Flags overwrite config file settings.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "content",
				Usage: "path to source folder containing markdown articles and related files of any type",
			},
			&cli.StringFlag{
				Name:  "static",
				Usage: "path to folder containing static files (js, css, ...)",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "path to output folder",
			},
			&cli.StringFlag{
				Name:     "config",
				Usage:    "config file to use",
				Required: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "livereload",
				Usage: "start a webserver and rebuild website on every change",
				Flags: []cli.Flag{
					&cli.DurationFlag{
						Name:  "check-interval",
						Usage: "how long to wait between checking for changed files",
						Value: 1 * time.Second,
					},
					&cli.StringFlag{
						Name:  "host",
						Usage: "hostname the server should listen to",
						Value: "localhost",
					},
					&cli.IntFlag{
						Name:  "port",
						Usage: "port the server should listen to",
						Value: 10000,
					},
				},
				Action: liveReload,
			},
		},
		Action: run,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
