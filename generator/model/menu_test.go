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

	require.ElementsMatch(t,
		[]MenuEntry{
			{Title: "Home", Path: "index.md"},
			{Title: "About", Path: "about.md"},
			{Title: "Blog", Path: "blog", IsDir: true},
		},
		Menu(content),
	)
}
