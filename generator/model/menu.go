package model

import (
	"sort"

	"github.com/klingtnet/static-site-generator/internal"
)

// MenuEntry is an entry in the navigation menu.
type MenuEntry struct {
	// Title of the menu entry.
	Title string
	// Path of the page file.
	Path string
	// IsDir is true if path is pointing to a directory.
	IsDir bool
}

// Menu builds a slice of menu entries for the given content tree.
//
// Note that the function will not recurse into the tree, instead
// the menu will only contain entries for the root level.
func Menu(tree Tree) []MenuEntry {
	menu := []MenuEntry{}
	containsPages := func(tree Tree) bool {
		for _, child := range tree.Children() {
			_, ok := child.(*Page)
			if ok {
				return true
			}
		}

		return false
	}

	for _, child := range tree.Children() {
		switch el := child.(type) {
		case *ContentTree:
			if containsPages(el) {
				menu = append(menu, MenuEntry{Title: internal.TitleCase(el.Name()), Path: el.Path(), IsDir: true})
			}
		case *Page:
			if el.fm.Hidden {
				continue
			}

			if el.Path() == "index.md" {
				menu = append(menu, MenuEntry{Title: "Home", Path: el.Path()})
			} else {
				menu = append(menu, MenuEntry{Title: el.Name(), Path: el.Path()})
			}
		}
	}

	// Sort directories before pages.
	sort.Slice(menu, func(i, j int) bool {
		a, b := menu[i], menu[j]

		if a.Path == "index.md" {
			return true
		}

		if a.IsDir == b.IsDir {
			return a.Title < b.Title
		}

		return a.IsDir
	})

	return menu
}
