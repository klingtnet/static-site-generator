package internal

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Utils below are a code smell and should not exist.

// EnglishTitleCaser implements unicode aware title casing for english strings.
// This function should be used instead of strings.Title which is deprecated and not unicode aware.
//
// Note that this caser is just a workaround to make the linter happy.  I should
// find a better solution for this, especially one that allows to select different
// title casing based on the language.  For now I only plan to publish english content, but
// this might change.
var EnglishTitleCaser = cases.Title(language.English)
