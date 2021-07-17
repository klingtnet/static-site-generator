package generator

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

type Config struct {
	Author       string `json:"author"`
	BaseURL      string `json:"base_url"`
	ContentDir   string `json:"content_dir"`
	StaticDir    string `json:"static_dir"`
	TemplatesDir string `json:"templates_dir"`
}

var (
	ErrAuthorUnset     = fmt.Errorf("author is unset")
	ErrContentDirUnset = fmt.Errorf("content dir is unset")
)

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

	return nil
}

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
