package generator

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// Config contains generator configuration values.
type Config struct {
	// Author is the websites' author (required).
	Author string `json:"author"`
	// BaseURL is the base URL that is used when generating absolute/canoncical URLs.
	BaseURL string `json:"base_url"`
	// ContentDir is the path of a directory that contains the websites content (required).
	ContentDir string `json:"content_dir"`
	// StaticDir is the path of a directory that contains static files to include in the generated website.
	StaticDir string `json:"static_dir"`
	// OutputDir is the path of a directory where the generated website will be stored into.
	OutputDir string `json:"output_dir"`
	// TemplatesDir is the path of a directory that contains a set of custom templates used to render the website.
	TemplatesDir string `json:"templates_dir"`
	// EnableUnsafeHTML allow embedding raw HTML snippets into markdown.
	EnableUnsafeHTML bool `json:"unsafe_html"`
}

var (
	ErrAuthorUnset     = fmt.Errorf("author is unset")
	ErrContentDirUnset = fmt.Errorf("content dir is unset")
	ErrOutputDirUnset  = fmt.Errorf("output dir is unset")
)

// Validate returns an error if the configuration is incomplete or invalid.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Author) == "" {
		return ErrAuthorUnset
	}

	if strings.TrimSpace(c.ContentDir) == "" {
		return ErrContentDirUnset
	}
	_, err := fs.Stat(os.DirFS(c.ContentDir), ".")
	if err != nil {
		return fmt.Errorf("bad source dir %q: %w", c.ContentDir, err)
	}

	if strings.TrimSpace(c.OutputDir) == "" {
		return ErrOutputDirUnset
	}
	_, err = fs.Stat(os.DirFS(c.OutputDir), ".")
	if err != nil {
		return fmt.Errorf("bad output dir %q: %w", c.OutputDir, err)
	}

	return nil
}

// ParseConfigFile instantiates a configuration from the given file.
func ParseConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parseConfig(data)
}

func parseConfig(data []byte) (config *Config, err error) {
	config = new(Config)
	err = json.Unmarshal(data, config)
	return
}
