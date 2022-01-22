package fswatcher

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/klingtnet/static-site-generator/internal/testutils"
)

func TestFSWatcher(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	sourceFS := testutils.NewConcurrentMapFS(fstest.MapFS{
		"index.md": &fstest.MapFile{
			ModTime: time.Unix(1, 0),
			Mode:    0o600,
			Data:    []byte("Hello, World!"),
		},
	})
	ticker := time.NewTicker(100 * time.Millisecond)

	// The first run will indicate a change because
	// a previous successful run is required to calculate a diff.
	watcher := New(sourceFS, ticker)
	resultCh := watcher.Watch(ctx)
	result := <-resultCh
	require.NoError(t, result.Err)
	require.True(t, result.HasChanged)

	// Nothing has changed.
	result = <-resultCh
	require.NoError(t, result.Err)
	require.False(t, result.HasChanged)

	// Check if a new file is detected.
	newMD := &fstest.MapFile{
		ModTime: time.Unix(1, 0),
		Data:    []byte("we should consider a content hash instead of diffing the file size."),
	}
	sourceFS.Store("new.md", newMD)
	result = <-resultCh
	require.NoError(t, result.Err)
	require.True(t, result.HasChanged)

	// Check if a change in modification time is detected.
	sourceFS.Store("index.md", &fstest.MapFile{
		ModTime: time.Unix(2, 0),
		Mode:    0o600,
		Data:    []byte("Hello, World!"),
	})
	result = <-resultCh
	require.NoError(t, result.Err)
	require.True(t, result.HasChanged)

	// Check if a size change is detected.
	sourceFS.Store("index.md", &fstest.MapFile{
		ModTime: time.Unix(2, 0),
		Mode:    0o600,
		Data:    []byte("This has a different size than 'Hello, World!'."),
	})
	result = <-resultCh
	require.NoError(t, result.Err)
	require.True(t, result.HasChanged)

	// We do not check file contents yet, hance this change in casing will not be detected.
	sourceFS.Store("new.md", &fstest.MapFile{
		ModTime: time.Unix(1, 0),
		Data:    []byte("WE SHOULD CONSIDER A CONTENT HASH INSTEAD OF DIFFING THE FILE SIZE."),
	})
	result = <-resultCh
	require.NoError(t, result.Err)
	require.False(t, result.HasChanged)
}
