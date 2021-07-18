// Package frontmatter provides a parser for page metadata stored in markdown code blocks.
package frontmatter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// ErrUnsupportedFormat indicates that the front-matter uses an unsupported format.
var ErrUnsupportedFormat = fmt.Errorf("unsupported format")

// ErrNoFrontMatter indicates that the content began without a front-matter.
var ErrNoFrontMatter = fmt.Errorf("no front-matter")

// ErrBadFrontMatter indicates that the front-matter is incomplete or not decodeable.
var ErrBadFrontMatter = fmt.Errorf("bad front-matter")

// Fence delimits front-matter blocks.
const Fence = "```"

// SimpleDateLayout is the time.Time layout used for SimpleDate typed dates that omit a timestamp.
const SimpleDateLayout = "2006-01-02"

// SimpleDate is a basic date format omitting a timestamp.
type SimpleDate time.Time

// NewSimpleDate returns a SimpleDate for the given date values.
func NewSimpleDate(year, month, day int) *SimpleDate {
	date := SimpleDate(time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC))
	return &date
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SimpleDate) UnmarshalJSON(b []byte) error {
	t, err := time.Parse(SimpleDateLayout, string(bytes.Trim(b, `"`)))
	if err != nil {
		return err
	}
	*s = SimpleDate(t)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (s *SimpleDate) MarshalJSON() ([]byte, error) {
	formatted := time.Time(*s).Format(SimpleDateLayout)
	return []byte(`"` + formatted + `"`), nil
}

func (s *SimpleDate) String() string {
	return time.Time(*s).Format(SimpleDateLayout)
}

// readLine reads from r until a newline symbol occurs or r.Read returns an error.
// Reading is unbuffered to prevent reading more than the actual line.
// This is the caes for bufio.ReadString and similar functions from this package.
func readLine(r io.Reader) (s string, err error) {
	b := make([]byte, 1)
	var sb strings.Builder
	for {
		_, err = r.Read(b)
		if err != nil {
			return
		}
		if b[0] == '\n' {
			s = sb.String()
			return
		}
		err = sb.WriteByte(b[0])
		if err != nil {
			return
		}
	}
}

// Read parses a front-matter from the given reader and stores the decoded data into dest.
// Parsing is stopped after the closing limiter of the front-matter has been read leaving
// the given reader reusable, e.g. to read the following content.
func Read(ctx context.Context, r io.Reader, dest interface{}) error {
	var data, format string

	for {
		line, err := readLine(r)
		if err != nil {
			return fmt.Errorf("%s: %w", ErrBadFrontMatter, err)
		}

		if strings.TrimSpace(line) == Fence {
			// Found closing fence.
			break
		} else if strings.HasPrefix(line, Fence) {
			// Found starting fence with format attribute.
			format = strings.TrimSpace(strings.TrimPrefix(line, Fence))
		} else if format == "" {
			// Content starts without a front-matter.
			return ErrNoFrontMatter
		} else {
			data += line
		}
	}

	switch format {
	case "json":
		err := json.Unmarshal([]byte(data), &dest)
		if err != nil {
			return fmt.Errorf("%s: %w", ErrBadFrontMatter, err)
		}
		return nil
	default:
		return fmt.Errorf("cannot unmarshal %q front-matter: %w", format, ErrUnsupportedFormat)
	}
}
