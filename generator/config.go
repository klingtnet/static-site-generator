package generator

import (
	"encoding/json"
	"os"
)

type Config struct {
	Author       string `json:"author"`
	BaseURL      string `json:"base_url"`
	ContentDir   string `json:"content_dir"`
	StaticDir    string `json:"static_dir"`
	TemplatesDir string `json:"templates_dir"`
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
