// Package ir is deck-engine's intermediate representation: a typed,
// backend-neutral model of a deck. Front ends (the Go builder API in the
// root package, the HTML extractor in cmd/deckgen) produce an ir.Deck;
// emitters (pptx, gslides) consume it.
//
// Coordinates are in 1600x900 design-grid units (the root package's
// CSS-pixel-like grid); colors are 6-digit hex without '#'; sizes are
// points. Emitters own all unit conversion. All defaults are resolved by
// the producer: emitters render exactly what the IR says (the only
// remaining fallbacks are Run.Size/Color/Font inheriting Text/Table
// defaults, which is part of the text model, not a palette default).
//
// The package depends only on the standard library.
package ir

// Deck is a complete presentation: an ordered slide list plus the theme.
type Deck struct {
	Theme  Theme
	Slides []*Slide
}

// Slide is an ordered list of shapes; order is z-order (first = bottom).
// Notes is the slide's speaker notes (plain text, blank line = paragraph
// break), or "".
type Slide struct {
	Shapes []Shape
	Notes  string
}

// Shape is the sealed sum of drawable node types: Rect, Oval, AutoShape,
// Text, Line, Elbow, Table.
type Shape interface{ isShape() }

// Frame is a shape's bounding box in grid units.
type Frame struct {
	X, Y, W, H float64
}

// Rect is a (possibly rounded) rectangle. Radius is the corner radius as a
// fraction (0..0.5) of the smaller side. Fill/Line are hex or "" for none.
// LineWpt is the outline width in points; 0 means the historical default
// of 1pt. Dashed draws the outline dashed (group boundaries in diagrams).
type Rect struct {
	Frame
	Fill, Line string
	LineWpt    float64
	Radius     float64
	Dashed     bool
	FillAlpha  float64 // 0 or 1 = opaque; (0,1) = partially transparent fill
}

// Oval is an ellipse. LineWpt is the outline width in points.
type Oval struct {
	Frame
	Fill, Line string
	LineWpt    float64
	Dashed     bool
	FillAlpha  float64
}

// AutoShape is a preset geometry shape for diagrams. Preset uses PPTX
// prstGeom names; emitters map them: "diamond", "can" (cylinder/database),
// "hexagon", "parallelogram", "cloud", "trapezoid", "chevron", "homePlate"
// (process/pentagon arrow), "triangle", "wedgeRectCallout" (callout).
type AutoShape struct {
	Frame
	Preset     string
	Fill, Line string
	LineWpt    float64
	Dashed     bool
	FillAlpha  float64
}

// Text is a multi-paragraph text box. DefSize/DefColor/DefFont apply to
// runs that leave the corresponding field zero. Rot rotates the box
// clockwise about its center, in degrees.
type Text struct {
	Frame
	Paras    []Para
	DefSize  float64
	DefColor string
	DefFont  string
	Rot      float64
}

// Line is a straight connector from (X1,Y1) to (X2,Y2). Arrow puts a
// triangle head at the (X2,Y2) end, ArrowStart at the (X1,Y1) end.
type Line struct {
	X1, Y1, X2, Y2 float64
	Color          string
	Wpt            float64
	Arrow, Dashed  bool
	ArrowStart     bool
}

// Table is a simple grid. ColW are column widths, RowH the uniform row
// height (a minimum in backends that grow rows), BorderColor the hairline
// color between rows. Cell fills and run colors/fonts are pre-resolved by
// the producer.
type Table struct {
	X, Y        float64
	ColW        []float64
	RowH        float64
	Rows        [][]Cell
	DefSize     float64
	BorderColor string
}

// Elbow is an orthogonal L-connector from (X1,Y1) to (X2,Y2): it runs
// vertically from the start, turns once, and runs horizontally into the
// end. (For horizontal-first routing, swap the endpoints and the arrow
// flags.) Arrow puts a head at the (X2,Y2) end, ArrowStart at (X1,Y1).
type Elbow struct {
	X1, Y1, X2, Y2 float64
	Color          string
	Wpt            float64
	Arrow, Dashed  bool
	ArrowStart     bool
}

func (Rect) isShape()      {}
func (Oval) isShape()      {}
func (AutoShape) isShape() {}
func (Text) isShape()      {}
func (Line) isShape()      {}
func (Elbow) isShape()     {}
func (Table) isShape()     {}

// Run is a span of uniformly-styled text. Zero Size/Color/Font inherit the
// containing Text/Table defaults. Link makes the run a hyperlink.
type Run struct {
	Text      string
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Color     string  // hex, "" = inherit
	Size      float64 // pt, 0 = inherit
	Font      string  // "" = inherit
	Link      string  // URL, "" = none
}

// Para is one paragraph of runs. Align is "l", "ctr" or "r" ("" = left);
// Spacing is a line-spacing multiple (0 = renderer default). Bullet turns
// the paragraph into a list item: "disc" or "number"; Level is the list
// nesting depth (0 = top level, meaningful only with Bullet).
type Para struct {
	Runs    []Run
	Align   string
	Spacing float64
	Bullet  string
	Level   int
}

// Cell is one table cell.
type Cell struct {
	Paras []Para
	Fill  string
}

// Theme mirrors the root package's Theme: color scheme + fonts written
// into backend-level theming (the OOXML theme part; advisory for Slides).
type Theme struct {
	Name                         string
	Font                         string
	Dark1, Light1, Dark2, Light2 string
	Accents                      [6]string
	Hyperlink, FollowedLink      string
}
