package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/klingtnet/static-site-generator/internal/fswatcher"
	"github.com/urfave/cli/v2"
)

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
