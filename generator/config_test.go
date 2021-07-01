package generator

import (
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
