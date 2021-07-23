package generator

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/klingtnet/static-site-generator/frontmatter"
)

// Library contains all content of the website.
type Library struct {
	// Pages map content path to page.
	Pages map[string]Page
	// Assets is a list of asset paths.
	Assets []string
	// Dirs is a list of all directories storing website content.
	Dirs []string
	// Menu is a list of entries for the navigation menu.
	Menu []MenuEntry
}

// PagesIn returns a list of pages for the given directory in descending order of creation date.
func (l *Library) PagesIn(dir string) []Page {
	var pages []Page
	for path, page := range l.Pages {
		if filepath.Base(filepath.Dir(filepath.Clean(path))) == dir {
			pages = append(pages, page)
		}
	}
	// Sort pages by Date descending
	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].FM.CreatedAt == nil || pages[j].FM.CreatedAt == nil {
			return false
		}
		return time.Time(*pages[i].FM.CreatedAt).After(time.Time(*pages[j].FM.CreatedAt))
	})

	return pages
}

// Page is a website page.
type Page struct {
	// Path of content file.
	Path string
	// FM contains page meta data from front matter.dw
	FM FrontMatter
	// Markdown formatted content.
	Markdown []byte
}

// FrontMatter stores metadata of a page.
type FrontMatter struct {
	// Author of the page.
	Author string `json:"author"`
	// Title of the page.
	Title string `json:"title"`
	// Description is a short abstract of the page.
	Description string `json:"description"`
	// CreatedAt determines when the article was written.
	CreatedAt *frontmatter.SimpleDate `json:"created_at"`
	// Tags are list of words categorizing the page.
	Tags []string `json:"tags"`
	// Hidden excludes page from navigation menu.
	Hidden bool `json:"hidden"`
}

// MenuEntry is an entry in the navigation menu.
type MenuEntry struct {
	// Title of the menu entry.
	Title string
	// Path of the page file.
	Path string
}
