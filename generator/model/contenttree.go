package model

import (
	"context"
	"io"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/klingtnet/static-site-generator/frontmatter"
)

type Tree interface {
	// Children returns a list of subtrees, if any.
	Children() []Tree
	// Path retuns the full path, starting from root of the tree.
	Path() string
	// Name returns the name of the tree node.
	Name() string
	// Walk calls func for every node in the (sub-)tree.
	Walk(func(Tree) error) error
}

type ContentTree struct {
	fullPath string
	name     string
	children []Tree
}

func readPage(ctx context.Context, contentFS fs.FS, name string) (*Page, error) {
	f, err := contentFS.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	page := &Page{
		name: name,
	}
	err = frontmatter.Read(ctx, f, &page.fm)
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	page.content = content

	return page, nil
}

func newWithParent(
	ctx context.Context,
	contentFS fs.FS,
	dir, parentDir string,
) (*ContentTree, error) {
	tree := &ContentTree{
		fullPath: filepath.Join(parentDir, dir),
		name:     dir,
	}

	entries, err := fs.ReadDir(contentFS, ".")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			subFS, err := fs.Sub(contentFS, entry.Name())
			if err != nil {
				return nil, err
			}

			// 🌲 Recurse into subtree.
			subTree, err := newWithParent(ctx, subFS, entry.Name(), tree.fullPath)
			if err != nil {
				return nil, err
			}
			tree.children = append(tree.children, subTree)

			continue
		}
		fullPath := path.Join(tree.fullPath, entry.Name())

		if path.Ext(entry.Name()) == ".md" {
			page, err := readPage(ctx, contentFS, entry.Name())
			if err != nil {
				return nil, err
			}
			page.fullPath = fullPath
			tree.children = append(tree.children, page)

			continue
		}

		tree.children = append(tree.children, &File{
			name:     entry.Name(),
			fullPath: fullPath,
		})
	}

	return tree, nil
}

func NewContentTree(ctx context.Context, contentFS fs.FS, dir string) (*ContentTree, error) {
	return newWithParent(ctx, contentFS, dir, "")
}

func (content *ContentTree) Children() []Tree {
	return content.children
}

func (content *ContentTree) Path() string {
	return content.fullPath
}

func (content *ContentTree) Name() string {
	return content.name
}

func (content *ContentTree) Walk(fn func(tree Tree) error) error {
	err := fn(content)
	if err != nil {
		return err
	}

	for _, child := range content.children {
		err = child.Walk(fn)
		if err != nil {
			return err
		}
	}

	return nil
}
