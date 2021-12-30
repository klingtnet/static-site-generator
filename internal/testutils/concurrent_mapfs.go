package testutils

import (
	"io/fs"
	"sync"
	"testing/fstest"
)

// ConcurrentMapFS implements fs.FS and provides synchronized modification methods.
// This is intended as a replacement for fstest.MapFS that is safe to use concurrently.
type ConcurrentMapFS struct {
	lock  sync.RWMutex
	mapFS fstest.MapFS
}

func NewConcurrentMapFS(mapFS fstest.MapFS) *ConcurrentMapFS {
	return &ConcurrentMapFS{mapFS: mapFS}
}

// Open implements fs.FS.
func (cmfs *ConcurrentMapFS) Open(name string) (fs.File, error) {
	cmfs.lock.RLock()
	defer cmfs.lock.RUnlock()

	return cmfs.mapFS.Open(name)
}

func (cmfs *ConcurrentMapFS) Store(path string, file *fstest.MapFile) {
	cmfs.lock.Lock()
	cmfs.mapFS[path] = file
	cmfs.lock.Unlock()
}

func (cmfs *ConcurrentMapFS) Delete(path string) {
	cmfs.lock.Lock()
	delete(cmfs.mapFS, path)
	cmfs.lock.Unlock()
}

// Ensure that fs.FS is implemented.
var _ fs.FS = &ConcurrentMapFS{}
