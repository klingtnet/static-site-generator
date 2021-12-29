// Package main implements the CLI for ssg, a static site generator.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"

	"github.com/klingtnet/static-site-generator/generator"
	"github.com/klingtnet/static-site-generator/generator/renderer"
	"github.com/klingtnet/static-site-generator/slug"
	"github.com/urfave/cli/v2"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
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

func run(c *cli.Context) error {
	config, err := generator.ParseConfigFile(c.String("config"))
	if err != nil {
		return cli.Exit(fmt.Sprintf("parsing config %q failed: %s", c.String("config"), err.Error()), BadArgument)
	}
	flagOverride(config, c)

	err = config.Validate()
	if err != nil {
		return cli.Exit(fmt.Sprintf("bad config: %s", err.Error()), BadArgument)
	}

	resources, err := initResources(config)
	if err != nil {
		return cli.Exit(fmt.Sprintf("bad resources: %s", err.Error()), BadArgument)
	}

	slugifier := slug.NewSlugifier('-')
	templates := renderer.NewTemplates(config.Author, config.BaseURL, slugifier, resources.templateFS)
	storage := generator.NewFileStorage(config.OutputDir)
	renderer := renderer.NewMarkdown(goldmark.New(goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.Footnote)), templates)
	err = generator.New(resources.sourceFS, resources.staticFS, storage, slugifier, renderer).Run(c.Context)
	if err != nil {
		return cli.Exit(fmt.Sprintf("generator failed: %s", err.Error()), InternalError)
	}

	return nil
}

type FSWatcher struct {
	filesystem fs.FS
	state      map[string]fs.FileInfo
}

type WatchResult struct {
	HasChanged bool
	Err        error
}

func (fsw *FSWatcher) collectState(state map[string]fs.FileInfo) error {
	return fs.WalkDir(fsw.filesystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		state[path] = info

		return nil
	})
}

// diff will first walk the watchers filesystem to collect information about all files (ignoring directories)
// and second it will compare this metadata against the last seen state.
func (fsw *FSWatcher) diff() (bool, error) {
	state := make(map[string]fs.FileInfo, len(fsw.state))
	err := fsw.collectState(state)
	if err != nil {
		return false, err
	}

	if len(state) != len(fsw.state) {
		fsw.state = state
		return true, nil
	}

	for path, info := range state {
		lastInfo, ok := fsw.state[path]
		if !ok {
			fsw.state = state
			return true, nil
		}

		if info.ModTime() != lastInfo.ModTime() ||
			info.Size() != lastInfo.Size() ||
			info.Mode() != lastInfo.Mode() {
			fsw.state = state
			return true, nil
		}
	}

	return false, nil
}

func (fsw *FSWatcher) Watch(ctx context.Context) <-chan WatchResult {
	resultCh := make(chan WatchResult)
	go func(resultCh chan<- WatchResult) {
		for {
			select {
			case <-ctx.Done():
				resultCh <- WatchResult{Err: ctx.Err()}

				return
			default:
				hasChanged, err := fsw.diff()
				if err != nil {
					resultCh <- WatchResult{Err: ctx.Err()}

					return
				}

				resultCh <- WatchResult{HasChanged: hasChanged}
				time.Sleep(1 * time.Second)
			}
		}
	}(resultCh)

	return resultCh
}

func NewFSWatcher(filesystem fs.FS) *FSWatcher {
	return &FSWatcher{
		filesystem: filesystem,
		state:      make(map[string]fs.FileInfo),
	}
}

func liveReload(c *cli.Context) error {
	config, err := generator.ParseConfigFile(c.String("config"))
	if err != nil {
		return cli.Exit(fmt.Sprintf("parsing config %q failed: %s", c.String("config"), err.Error()), BadArgument)
	}
	flagOverride(config, c)

	err = config.Validate()
	if err != nil {
		return cli.Exit(fmt.Sprintf("bad config: %s", err.Error()), BadArgument)
	}

	resources, err := initResources(config)
	if err != nil {
		return cli.Exit(fmt.Sprintf("bad resources: %s", err.Error()), BadArgument)
	}

	slugifier := slug.NewSlugifier('-')
	templates := renderer.NewTemplates(config.Author, config.BaseURL, slugifier, resources.templateFS)
	storage := generator.NewFileStorage(config.OutputDir)
	renderer := renderer.NewMarkdown(goldmark.New(goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.Footnote)), templates)
	generator := generator.New(resources.sourceFS, resources.staticFS, storage, slugifier, renderer)

	resourcesWatcherCh := NewFSWatcher(resources.sourceFS).Watch(c.Context)
	staticWatcherCh := NewFSWatcher(resources.staticFS).Watch(c.Context)
	templatesWatcherCh := NewFSWatcher(resources.templateFS).Watch(c.Context)

	for {
		hasChanged := false

		select {
		case result := <-resourcesWatcherCh:
			if result.Err != nil {
				return err
			}

			hasChanged = result.HasChanged
		case result := <-staticWatcherCh:
			if result.Err != nil {
				return err
			}

			hasChanged = result.HasChanged
		case result := <-templatesWatcherCh:
			if result.Err != nil {
				return err
			}

			hasChanged = result.HasChanged
		}

		if hasChanged {
			log.Println("something has changed, rebuilding...")

			err = generator.Run(c.Context)
			if err != nil {
				return cli.Exit(fmt.Sprintf("generator failed: %s", err.Error()), InternalError)
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
				Usage: "rebuild website on every change",
				Flags: []cli.Flag{
					&cli.DurationFlag{
						Name:  "check-interval",
						Usage: "time to wait between looking for changes",
						Value: 1 * time.Second,
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
