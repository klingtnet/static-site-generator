package slug

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlugify(t *testing.T) {
	tCases := []struct {
		name  string
		title string
		slug  string
	}{
		{"empty", "", ""},
		{"only-whitespace", "  	\n	", ""},
		{"unicode", "raison d'être", "raison-d-etre"},
		{"french book title", `Les métamorphoses d’Aladin ou comment il fut passé au caviar`, "les-metamorphoses-d-aladin-ou-comment-il-fut-passe-au-caviar"},
		{"konnichi wa", `こんにちは`, `こんにちは`},
		{"hyphens", "a-b‐c⸗d﹣e", "a-b-c-d-e"},
		{"collapsed-repetitions", "''a'''b''c''d'", "a-b-c-d"},
		{"spiegel.de", `Erklärtes Ziel ist ein »Präsenzschuljahr«`, "erklaertes-ziel-ist-ein-praesenzschuljahr"},
		{"Straße", "Straße", "strasse"},
	}

	sl := NewSlugifier('-')
	for _, tCase := range tCases {
		t.Run(tCase.name, func(t *testing.T) {
			require.Equal(t, tCase.slug, sl.Slugify(tCase.title))
		})
	}
}
