// Package htmlconv turns an HTML deck (one 1600x900 <section> per slide,
// laid out by headless Chrome) into deck-engine IR. The extraction is split
// in two so the interesting half stays testable without a browser:
// extract.js runs in the page and returns a scene graph (computed geometry
// + styles, CSS px); MapScenes (mapscene.go) is a pure Go conversion from
// scenes to ir.Deck.
package htmlconv

// Result is extract.js's return value.
type Result struct {
	Scenes   []Scene  `json:"scenes"`
	Warnings []string `json:"warnings"`
}

// Scene is one <section>: its size, background and paint-ordered elements.
type Scene struct {
	W          float64   `json:"w"`
	H          float64   `json:"h"`
	Background string    `json:"background"`
	Notes      string    `json:"notes"`
	Elements   []Element `json:"elements"`
	Warnings   []string  `json:"warnings"`
}

// Element is the union of extracted node kinds ("box", "line", "text",
// "table"); which fields are meaningful depends on Kind. All geometry is in
// CSS px relative to the section origin.
type Element struct {
	Kind string `json:"kind"`

	// box, text, table
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`

	// box (Border doubles as the table hairline color for Kind "table";
	// Dashed doubles as the line dash flag for Kind "line")
	Shape     string  `json:"shape"` // data-shape preset name, "" = plain box
	Fill      string  `json:"fill"`
	FillAlpha float64 `json:"fillAlpha"` // 0 = opaque, (0,1) = transparency
	Border    string  `json:"border"`
	BorderW   float64 `json:"borderW"`
	Radius    float64 `json:"radius"` // fraction of the smaller side, 0..0.5
	Oval      bool    `json:"oval"`

	// line + elbow (Kind "elbow": vertical from (x1,y1), bend, horizontal
	// into (x2,y2))
	X1         float64 `json:"x1"`
	Y1         float64 `json:"y1"`
	X2         float64 `json:"x2"`
	Y2         float64 `json:"y2"`
	Color      string  `json:"color"`
	WidthPx    float64 `json:"widthPx"`
	Dashed     bool    `json:"dashed"`
	Arrow      bool    `json:"arrow"`
	ArrowStart bool    `json:"arrowStart"`

	// text
	Align     string  `json:"align"`
	Spacing   float64 `json:"spacing"`
	FontSize  float64 `json:"fontSize"` // px
	FontColor string  `json:"fontColor"`
	FontFam   string  `json:"fontFam"`
	Rot       float64 `json:"rot"` // clockwise degrees about the box center
	Paras     []Para  `json:"paras"`

	// table
	ColW    []float64 `json:"colW"`
	RowH    float64   `json:"rowH"`
	DefSize float64   `json:"defSize"`
	Rows    [][]Cell  `json:"rows"`
}

// Para is one paragraph of runs; Bullet/Level make it a list item.
type Para struct {
	Bullet string `json:"bullet"` // "", "disc", "number"
	Level  int    `json:"level"`
	Runs   []Run  `json:"runs"`
}

// Run is a styled span.
type Run struct {
	Text      string  `json:"text"`
	Bold      bool    `json:"bold"`
	Italic    bool    `json:"italic"`
	Underline bool    `json:"underline"`
	Strike    bool    `json:"strike"`
	Color     string  `json:"color"`
	Size      float64 `json:"size"` // px
	Font      string  `json:"font"`
	Link      string  `json:"link"`
}

// Cell is one table cell.
type Cell struct {
	Fill  string `json:"fill"`
	Align string `json:"align"`
	Paras []Para `json:"paras"`
}
