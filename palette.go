package deck

// Default palette and fonts used by the style kit (style.go) and
// DefaultTheme. Hex values carry no '#'. These are Material-flavored
// defaults; the core writer (deck.go) takes explicit colors everywhere, so
// swapping brands means changing call sites or these constants, not the
// engine.
const (
	BLUE    = "4285F4"
	BLUE7   = "1967D2"
	BLUE50  = "E8F0FE"
	RED     = "EA4335"
	RED50   = "FCE8E6"
	REDD    = "C5221F"
	YELL    = "FBBC04"
	YELL50  = "FEF7E0"
	YELLD   = "B06000"
	GREEN   = "34A853"
	GREEN50 = "E6F4EA"
	GREEND  = "188038"
	INK     = "202124" // primary text
	INK2    = "5F6368" // secondary text
	INK3    = "80868B" // muted text
	LINEC   = "DADCE0" // hairlines
	SURF2   = "F8F9FA" // card surface
	WHITE   = "FFFFFF"
	DARK    = "1E1E1E" // code block background
	CODET   = "D4D4D4" // code text
	// FONT is Roboto so text renders consistently in Google Slides and through
	// the public API.
	FONT = "Roboto"
	MONO = "Roboto Mono"
)
