package generator

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExampleConfig(t *testing.T) {
	config, err := ParseConfigFile("../config.example.json")
	require.NoError(t, err, "parsing example config failed")
	require.Equal(t, config, &Config{
		Author:       "John Doe",
		BaseURL:      "https://john.doe",
		ContentDir:   "./content",
		StaticDir:    "/optional/static",
		TemplatesDir: "/optional/templates",
	})
}

func TestConfigValidate(t *testing.T) {
	tCases := []struct {
		name   string
		config *Config
		err    error
	}{
		{"minimal config", &Config{Author: "John Doe", ContentDir: "."}, nil},
		{"no author", &Config{ContentDir: "./content"}, ErrAuthorUnset},
		{"no content dir", &Config{Author: "John Doe"}, ErrContentDirUnset},
		{"bad content dir", &Config{Author: "John Doe", ContentDir: "./should/not/exist"}, fs.ErrNotExist},
	}
	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			err := tCase.config.Validate()
			if tCase.err != nil {
				require.ErrorIs(t, err, tCase.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
