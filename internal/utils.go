package internal

import (
	"sync"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Utils below are a code smell and should not exist.

var caserPool = sync.Pool{
	New: func() any {
		caser := cases.Title(language.English)
		return &caser
	},
}

// TitleCase returns string s in title-case.  For now only english strings are supported.
//
// Note that this is just a workaround to make the linter happy.  I should
// find a better solution for this, especially one that allows to select different
// title casing based on the language.  For now I only plan to publish english content, but
// this might change.
func TitleCase(s string) string {
	caser := caserPool.Get().(*cases.Caser)
	defer caserPool.Put(caser)

	return caser.String(s)
}
