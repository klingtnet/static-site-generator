package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Slugifyer provides a method to clean strings for the use in URLs.
type Slugifier struct {
	replacement        rune
	repetitionRE       *regexp.Regexp
	unicodeTransformer transform.Transformer
}

// NewSlugifyer returns a initialized Slugifyer instance.
func NewSlugifier(replacement rune) *Slugifier {
	mappingFn := func() func(r rune) rune {
		isPunctuation := runes.In(unicode.Punct).Contains
		isSpace := runes.In(unicode.Space).Contains
		return func(r rune) rune {
			if isPunctuation(r) || isSpace(r) {
				return replacement
			}
			return r
		}
	}()

	return &Slugifier{
		replacement:  replacement,
		repetitionRE: regexp.MustCompile(`(` + string(replacement) + `{2,})`),
		// NKFD is the decomposed compatibility equivalsle form of unicode, e.g. ê will become e^.
		// For more details see this blog post https://blog.golang.org/normalization.
		unicodeTransformer: transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mark)), runes.Map(mappingFn)),
	}
}

var transliterations = map[rune]string{
	'ä': "ae",
	'ö': "oe",
	'ü': "ue",
	'ß': "ss",
}

// transliterateGerman will replace Umlauts and Eszett with their transliterations.
func transliterateGerman(s string) string {
	var out string
	for _, c := range s {
		transliteration, ok := transliterations[c]
		if ok {
			out += transliteration
		} else {
			out += string(c)
		}
	}
	return out
}

// Slugify takes a string and returns a SEO friendly normalized version of that string.
// Normalization applies a set of transformations to the string, e.g. lower-casing, trimming whitespace and replacing certain unicode characters/symbols.
// See the test cases for some examples.
func (sl *Slugifier) Slugify(s string) string {
	s = transliterateGerman(strings.TrimSpace(strings.ToLower(s)))

	s, _, err := transform.String(sl.unicodeTransformer, s)
	if err != nil {
		// Note that this should not happen since the test cases are covering a variety of common corner-cases.
		panic(err)
	}

	return sl.trimReplacement(sl.collapseRepetitions(s))
}

// collapseRepetitions replaces repeated occursles of the replacement rune with a single instance.
func (sl *Slugifier) collapseRepetitions(s string) string {
	return sl.repetitionRE.ReplaceAllString(s, string(sl.replacement))
}

// trimReplacement removes leading and trailing occurrsles of the replacement character.
func (sl *Slugifier) trimReplacement(s string) string {
	return strings.Trim(s, string(sl.replacement))
}
