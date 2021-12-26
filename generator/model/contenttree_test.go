package model

import (
	"context"
	"testing"

	"github.com/klingtnet/static-site-generator/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestContentTree(t *testing.T) {
	contentFS := testutils.NewTestContentFS(t)

	content, err := NewContentTree(context.Background(), contentFS, ".")
	require.NoError(t, err)

	var files []string
	var pages []string
	var dirs []string
	err = content.Walk(func(tree Tree) error {
		switch el := tree.(type) {
		case *ContentTree:
			dirs = append(dirs, el.Path())
		case *File:
			files = append(files, el.Path())
		case *Page:
			pages = append(pages, el.Path())
		}

		return nil
	})
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"files/random.txt"}, files)
	require.ElementsMatch(t, []string{
		"about.md",
		"index.md",
		"blog/first.md",
		"blog/second.md",
	}, pages)
	require.ElementsMatch(t, []string{
		".",
		"blog",
		"files",
	}, dirs)
}
