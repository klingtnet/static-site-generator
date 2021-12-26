package model

import "github.com/klingtnet/static-site-generator/frontmatter"

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

type Page struct {
	content  []byte
	fm       FrontMatter
	fullPath string
	name     string
}

func (p *Page) Children() []Tree {
	return nil
}

func (p *Page) Path() string {
	return p.fullPath
}

func (p *Page) Name() string {
	return p.fm.Title
}

func (p *Page) Walk(fn func(tree Tree) error) error {
	return fn(p)
}

func (p *Page) Frontmatter() *FrontMatter {
	return &p.fm
}

func (p *Page) Content() []byte {
	return p.content
}
