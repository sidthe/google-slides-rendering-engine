package gslides

import "strings"

// Substitutions maps font names unavailable in Google Slides (or CSS generic
// families from the HTML front end) to fonts that
// exist in Slides. Lookup is case-insensitive; unknown names pass through
// unchanged. Callers may add or override entries before emitting.
var Substitutions = map[string]string{
	"product sans":   "Roboto",
	"sans-serif":     "Roboto",
	"serif":          "Georgia",
	"monospace":      "Roboto Mono",
	"ui-monospace":   "Roboto Mono",
	"system-ui":      "Roboto",
	"-apple-system":  "Roboto",
	"helvetica":      "Arial",
	"helvetica neue": "Arial",
}

// ResolveFont resolves a font-family value (possibly a CSS stack like
// `"Roboto", Arial, sans-serif`) to a single Slides-usable family: the
// first entry, substituted if denied/generic. Exported so front ends (the
// HTML extractor) can resolve fonts the same way before building IR.
func ResolveFont(family string) string {
	first := family
	if i := strings.IndexByte(family, ','); i >= 0 {
		first = family[:i]
	}
	first = strings.Trim(strings.TrimSpace(first), `"'`)
	if sub, ok := Substitutions[strings.ToLower(first)]; ok {
		return sub
	}
	return first
}
