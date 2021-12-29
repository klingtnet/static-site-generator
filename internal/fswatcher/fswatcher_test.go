package fswatcher

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFSWatcher(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	indexMD := &fstest.MapFile{
		ModTime: time.Now(),
		Mode:    0600,
		Data:    []byte("Hello, World!"),
	}
	sourceFS := fstest.MapFS{
		"index.md": indexMD,
	}
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
		ModTime: time.Now(),
		Data:    []byte("we should consider a content hash instead of diffing the file size."),
	}
	sourceFS["new.md"] = newMD
	result = <-resultCh
	require.NoError(t, result.Err)
	require.True(t, result.HasChanged)

	// Check if a change in modification time is detected.
	indexMD.ModTime = time.Now()
	result = <-resultCh
	require.NoError(t, result.Err)
	require.True(t, result.HasChanged)

	// Check if a size change is detected.
	indexMD.Data = []byte("This has a different size than 'Hello, World!'.")
	result = <-resultCh
	require.NoError(t, result.Err)
	require.True(t, result.HasChanged)

	// We do not check file contents yet, hance this change in casing will not be detected.
	newMD.Data = []byte("We should consider a content hash instead of diffing the file size.")
	result = <-resultCh
	require.NoError(t, result.Err)
	require.False(t, result.HasChanged)
}
