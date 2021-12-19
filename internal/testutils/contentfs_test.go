package testutils

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTestContentFS(t *testing.T) {
	contentFS := NewTestContentFS(t)
	require.NotNil(t, contentFS)

	expected := []string{
		"about.md",
		"index.md",
		"blog/first.md",
		"blog/second.md",
		"files/random.txt",
	}

	var actual []string
	fs.WalkDir(contentFS, ".", func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)

		if d.Type().IsRegular() {
			actual = append(actual, path)
		}

		return nil
	})

	require.ElementsMatch(t, expected, actual)
}
