package generator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage provides methods for persisting files of the generated website.
type Storage interface {
	Store(ctx context.Context, name string, content io.Reader) error
}

// FileStorage persists to a local file system.
type FileStorage struct {
	baseDir string
}

// NewFileStorage returns an initialized FileStorage.
func NewFileStorage(baseDir string) *FileStorage {
	return &FileStorage{baseDir}
}

var ErrEmptyName = fmt.Errorf("name must not be empty")

// Store implements Storage.
func (s *FileStorage) Store(ctx context.Context, name string, content io.Reader) error {
	if strings.TrimSpace(name) == "" {
		return ErrEmptyName
	}

	// Note that making the name absolute is preventing path traversal.
	// This is because filepath.Clean is then removing parent directory references, aka double dots.
	destPath := filepath.Join(s.baseDir, filepath.Clean("/"+name))
	err := os.MkdirAll(filepath.Dir(destPath), 0700)
	if err != nil {
		return err
	}
	dest, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, content)
	return err
}
