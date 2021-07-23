package generator

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorage(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStorage(dir)

	tCases := []struct {
		name         string
		expectedName string
		content      string
		err          error
	}{
		{"a/b/c", "/a/b/c", "content", nil},
		{"../../etc/passwd", "/etc/passwd", "something", nil},
		{"", "", "", ErrEmptyName},
	}
	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			err := s.Store(context.Background(), tCase.name, bytes.NewBufferString(tCase.content))
			if tCase.err != nil {
				require.ErrorIs(t, err, tCase.err)
				return
			}
			require.NoError(t, err)
			content, err := os.ReadFile(filepath.Join(dir, tCase.expectedName))
			require.NoError(t, err, "could not read destination file")
			require.Equal(t, string(content), tCase.content)
		})
	}
}
