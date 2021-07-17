// Package main implements the CLI for ssg, a static site generator.
package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/klingtnet/static-site-generator/generator"
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
		r.templateFS, err = fs.Sub(generator.DefaultTemplateFS, "templates")
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
	templates := generator.NewTemplates(config.Author, config.BaseURL, slugifier, resources.templateFS)

	storage := generator.NewFileStorage(c.String("output"))
	renderer := generator.NewRenderer(goldmark.New(goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.Footnote)), templates)
	err = generator.New(resources.sourceFS, resources.staticFS, storage, slugifier, renderer).Run(c.Context)
	if err != nil {
		return cli.Exit(fmt.Sprintf("generator failed: %s", err.Error()), InternalError)
	}

	return nil
}

func main() {
	app := cli.App{
		Name:        "ssg",
		Description: "A opinionated static site generator. Flags overwrite config file settings.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "content",
				Usage: "path to source folder containing markdown articles and related files of any type",
				Value: "content",
			},
			&cli.StringFlag{
				Name:  "static",
				Usage: "path to folder containing static files (js, css, ...)",
			},
			&cli.StringFlag{
				Name:  "output",
				Usage: "path to output folder",
				Value: "output",
			},
			&cli.StringFlag{
				Name:     "config",
				Usage:    "config file to use",
				Required: true,
			},
		},
		Action: run,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
