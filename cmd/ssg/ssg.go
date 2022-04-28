// Package main implements the CLI for ssg, a static site generator.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	InternalError = iota + 1
	BadArgument
)

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
