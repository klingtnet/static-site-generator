package frontmatter

import (
	"bytes"
	"context"
	"embed"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type testFrontMatter struct {
	Author    string      `json:"author"`
	CreatedAt *SimpleDate `json:"created_at"`
}

//go:embed testdata/*.md
var testdata embed.FS

func readTestContent(t *testing.T, name string) string {
	data, err := testdata.ReadFile(filepath.Join("testdata", name+".md"))
	require.NoError(t, err)
	return string(data)
}

func TestRead(t *testing.T) {
	tCases := []struct {
		name     string
		document string
		content  string
		data     testFrontMatter
		err      error
	}{
		{
			"valid",
			readTestContent(t, "valid-page"),
			"\n# A test page",
			testFrontMatter{Author: "Andreas Linz", CreatedAt: NewSimpleDate(2021, 05, 26)},
			nil,
		},
		{
			"no-content",
			readTestContent(t, "no-content"),
			"",
			testFrontMatter{Author: "Andreas Linz", CreatedAt: NewSimpleDate(2021, 05, 26)},
			nil,
		},
		{
			"empty",
			"",
			"",
			testFrontMatter{},
			io.EOF,
		},
		{
			"yaml",
			readTestContent(t, "yaml"),
			"",
			testFrontMatter{},
			ErrUnsupportedFormat,
		},
		{
			"no-front-matter",
			readTestContent(t, "no-front-matter"),
			"",
			testFrontMatter{},
			ErrNoFrontMatter,
		},
		{
			"unfinished-front-matter",
			readTestContent(t, "unfinished-front-matter"),
			"",
			testFrontMatter{},
			io.EOF,
		},
	}
	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			b := bytes.NewBufferString(tCase.document)
			var d testFrontMatter
			err := Read(context.Background(), b, &d)
			if tCase.err != nil {
				require.ErrorIs(t, err, tCase.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tCase.data, d)
			content, err := io.ReadAll(b)
			require.NoError(t, err)
			require.Equal(t, tCase.content, string(content))
		})
	}
}

func TestSimpleDate(t *testing.T) {
	d := NewSimpleDate(2020, 07, 17)
	jsonEncoded := `"2020-07-17"`

	actual, err := d.MarshalJSON()
	require.NoError(t, err, "marshaling failed")
	require.Equal(t, jsonEncoded, string(actual))
	d = &SimpleDate{}
	require.NoError(t, d.UnmarshalJSON([]byte(jsonEncoded)), "unmarshaling failed")
	require.Equal(t, "2020-07-17", d.String())
}
