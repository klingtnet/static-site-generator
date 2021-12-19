package testutils

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed testdata
var testdata embed.FS

func NewTestContentFS(t *testing.T) fs.FS {
	contentFS, err := fs.Sub(testdata, "testdata/content")
	require.NoError(t, err)

	return contentFS
}
