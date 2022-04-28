package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/klingtnet/static-site-generator/generator"
	"github.com/klingtnet/static-site-generator/generator/renderer"
	"github.com/klingtnet/static-site-generator/slug"
	"github.com/urfave/cli/v2"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
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
		err = cli.Exit(
			fmt.Sprintf("parsing config %q failed: %s", c.String("config"), err.Error()),
			BadArgument,
		)
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
	templates := renderer.NewTemplates(
		config.Author,
		config.BaseURL,
		slugifier,
		resources.templateFS,
	)
	storage := generator.NewFileStorage(config.OutputDir)

	markdownOptions := []goldmark.Option{
		goldmark.WithExtensions(extension.GFM, emoji.Emoji, extension.Footnote),
	}
	if config.EnableUnsafeHTML {
		markdownOptions = append(
			markdownOptions,
			goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
		)
	}
	renderer := renderer.NewMarkdown(goldmark.New(markdownOptions...), templates)

	gen = generator.New(
		config,
		resources.sourceFS,
		resources.staticFS,
		storage,
		slugifier,
		renderer,
	)

	return
}
