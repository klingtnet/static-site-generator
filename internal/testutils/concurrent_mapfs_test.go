package testutils

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConcurrentMapFS(t *testing.T) {
	// Note that concurrent access is indirectly by tested by internal/fswatcher/fswatcher_test.go.

	cmfs := NewConcurrentMapFS(fstest.MapFS{
		"root.txt": &fstest.MapFile{
			Data:    []byte("root"),
			Mode:    0o400,
			ModTime: time.Now(),
		},
		"dir/file.bin": &fstest.MapFile{
			Data:    []byte{1, 2, 3, 4},
			Mode:    0o755,
			ModTime: time.Now(),
		},
	})
	require.NoError(t, fstest.TestFS(cmfs, "root.txt", "dir/file.bin"))
}
