package fswatcher

import (
	"context"
	"io/fs"
	"os"
	"time"
)

type fileInfo struct {
	modTime time.Time
	size    int64
	mode    os.FileMode
}

type FSWatcher struct {
	filesystem fs.FS
	state      map[string]fileInfo
	ticker     *time.Ticker
}

type Result struct {
	HasChanged bool
	Err        error
}

func (fsw *FSWatcher) collectState(state map[string]fileInfo) error {
	return fs.WalkDir(fsw.filesystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		state[path] = fileInfo{
			modTime: info.ModTime(),
			size:    info.Size(),
			mode:    info.Mode(),
		}

		return nil
	})
}

// diff will first walk the watchers filesystem to collect information about all files (ignoring directories)
// and second it will compare this metadata against the last seen state.
func (fsw *FSWatcher) diff() (bool, error) {
	state := make(map[string]fileInfo, len(fsw.state))
	err := fsw.collectState(state)
	if err != nil {
		return false, err
	}
	defer func() {
		fsw.state = state
	}()

	if len(state) != len(fsw.state) {
		return true, nil
	}

	for path, info := range state {
		lastInfo, ok := fsw.state[path]
		if !ok {
			return true, nil
		}

		if !info.modTime.Equal(lastInfo.modTime) ||
			info.size != lastInfo.size ||
			info.mode != lastInfo.mode {
			return true, nil
		}
	}

	return false, nil
}

func (fsw *FSWatcher) Watch(ctx context.Context) <-chan Result {
	resultCh := make(chan Result)
	go func(resultCh chan<- Result) {
		for {
			select {
			case <-ctx.Done():
				resultCh <- Result{Err: ctx.Err()}

				return
			case <-fsw.ticker.C:
				hasChanged, err := fsw.diff()
				if err != nil {
					resultCh <- Result{Err: ctx.Err()}

					return
				}

				resultCh <- Result{HasChanged: hasChanged}
			}
		}
	}(resultCh)

	return resultCh
}

func New(filesystem fs.FS, ticker *time.Ticker) *FSWatcher {
	return &FSWatcher{
		filesystem: filesystem,
		state:      make(map[string]fileInfo),
		ticker:     ticker,
	}
}
