package deck

import (
	"github.com/sidthe/google-slides-rendering-engine/ir"
	"github.com/sidthe/google-slides-rendering-engine/pptx"
)

// Theme controls the OOXML theme part (color scheme + font scheme) written
// into the package. Slide content sets its own explicit colors and fonts, so
// the theme mostly matters for what users see when they edit the deck.
// Structurally identical to ir.Theme; kept as a distinct named type so its
// resolution helpers live here.
type Theme struct {
	Name string
	// Major/minor latin typeface.
	Font string
	// Color scheme, hex without '#'.
	Dark1, Light1, Dark2, Light2 string
	Accents                      [6]string
	Hyperlink, FollowedLink      string
}

// DefaultTheme returns the built-in Material-flavored theme that matches the
// default palette in palette.go. Substitute your own brand values freely.
func DefaultTheme() Theme {
	return Theme{
		Name:  "deck-engine",
		Font:  FONT,
		Dark1: INK, Light1: WHITE, Dark2: INK2, Light2: SURF2,
		Accents:   [6]string{BLUE, RED, YELL, GREEN, BLUE7, GREEND},
		Hyperlink: BLUE7, FollowedLink: "681DA8",
	}
}

// theme resolves the deck's theme, falling back to the default for a zero
// value so `var d deck.Deck` keeps working.
func (d *Deck) theme() Theme {
	if d.Theme.Font == "" {
		return DefaultTheme()
	}
	return d.Theme
}

func (t Theme) xml() string {
	return pptx.ThemeXML(ir.Theme(t))
}
