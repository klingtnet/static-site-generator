package model

import (
	"context"
	"testing"

	"github.com/klingtnet/static-site-generator/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestMenu(t *testing.T) {
	content, err := NewContentTree(context.Background(), testutils.NewTestContentFS(t), ".")
	require.NoError(t, err)

	rootMenu := []MenuEntry{
		{Title: "Home", Path: "index.md"},
		{Title: "About", Path: "about.md"},
		{Title: "Blog", Path: "blog", IsDir: true},
	}
	require.ElementsMatch(
		t,
		rootMenu,
		Menu(content),
	)

	var blog *ContentTree
	for _, child := range content.Children() {
		ct, ok := child.(*ContentTree)
		if !ok {
			continue
		}

		if ct.Path() == "blog" {
			blog = ct
		}
	}
	require.ElementsMatch(
		t,
		[]MenuEntry{
			{Title: "First Article", Path: "blog/first.md"},
			{Title: "Second Article", Path: "blog/second.md"},
		},
		Menu(blog),
	)
}
