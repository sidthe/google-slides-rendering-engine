// Package deck is a minimal, dependency-free deck builder.
//
// The builder API in this package produces a typed intermediate
// representation (see the ir package); emitters render it as a native
// .pptx package (pptx package) or as Google Slides API requests (gslides
// package). All content is native and editable in PowerPoint and Google
// Slides.
//
// Coordinates use a 1600x900 design grid mapped onto a 16:9 slide, so layout
// code reads like CSS-pixel positioning.
package deck

import (
	"github.com/deck-engine/deck-engine/ir"
	"github.com/deck-engine/deck-engine/pptx"
)

// ---------- runs & paragraphs (shared with the IR) ----------

type (
	Run  = ir.Run
	Para = ir.Para
	Cell = ir.Cell
)

func P(runs ...Run) Para { return Para{Runs: runs} }
func R(t string) Run     { return Run{Text: t} }
func B(t string) Run     { return Run{Text: t, Bold: true} }
func BC(t, c string) Run { return Run{Text: t, Bold: true, Color: c} }
func RC(t, c string) Run { return Run{Text: t, Color: c} }

// ---------- slide ----------

// Slide is a free-form shape canvas; emit order is z-order.
type Slide struct {
	node *ir.Slide
}

// Deck is an ordered collection of slides plus the theme written into the
// package. A zero Deck is usable and saves with DefaultTheme.
type Deck struct {
	Theme  Theme
	slides []*Slide
}

// New returns a Deck with the default theme.
func New() *Deck { return &Deck{Theme: DefaultTheme()} }

func (d *Deck) Slide() *Slide {
	s := &Slide{node: &ir.Slide{}}
	d.slides = append(d.slides, s)
	return s
}

// Len returns the number of slides added so far.
func (d *Deck) Len() int { return len(d.slides) }

// IR returns the deck's intermediate representation, the input to the
// pptx and gslides emitters. The returned value shares slide data with
// the builder.
func (d *Deck) IR() *ir.Deck {
	out := &ir.Deck{Theme: ir.Theme(d.theme())}
	for _, s := range d.slides {
		out.Slides = append(out.Slides, s.node)
	}
	return out
}

// ---------- shape builders ----------

// Rect: rounded rectangle. radius: corner radius as fraction (0..0.5) of the
// smaller side, matching CSS-like feel. fill/line are hex or "" for none.
func (s *Slide) Rect(x, y, w, h float64, fill, line string, radius float64) {
	s.node.Shapes = append(s.node.Shapes, ir.Rect{
		Frame: ir.Frame{X: x, Y: y, W: w, H: h},
		Fill:  fill, Line: line, Radius: radius,
	})
}

func (s *Slide) Oval(x, y, w, h float64, fill, line string, lineWpt float64) {
	s.node.Shapes = append(s.node.Shapes, ir.Oval{
		Frame: ir.Frame{X: x, Y: y, W: w, H: h},
		Fill:  fill, Line: line, LineWpt: lineWpt,
	})
}

// Text: multi-paragraph text box. defSize pt, defColor hex, defFont.
func (s *Slide) Text(x, y, w, h float64, paras []Para, defSize float64, defColor, defFont string) {
	s.node.Shapes = append(s.node.Shapes, ir.Text{
		Frame: ir.Frame{X: x, Y: y, W: w, H: h},
		Paras: copyParas(paras), DefSize: defSize, DefColor: defColor, DefFont: defFont,
	})
}

// Line: straight connector; arrow=true adds triangle tail; dashed optional.
func (s *Slide) Line(x1, y1, x2, y2 float64, color string, wpt float64, arrow, dashed bool) {
	s.node.Shapes = append(s.node.Shapes, ir.Line{
		X1: x1, Y1: y1, X2: x2, Y2: y2,
		Color: color, Wpt: wpt, Arrow: arrow, Dashed: dashed,
	})
}

// Table: simple grid with header row styling handled by caller via bold
// flags. Palette defaults (secondary-ink text, white fills, hairline row
// borders) are resolved here so every emitter renders identical colors.
func (s *Slide) Table(x, y float64, colW []float64, rowH float64, rows [][]Cell, defSize float64) {
	resolved := make([][]Cell, len(rows))
	for ri, row := range rows {
		resolved[ri] = make([]Cell, len(row))
		for ci, c := range row {
			c.Paras = copyParas(c.Paras)
			for pi := range c.Paras {
				for qi := range c.Paras[pi].Runs {
					r := &c.Paras[pi].Runs[qi]
					if r.Color == "" {
						r.Color = INK2
					}
					if r.Font == "" {
						r.Font = FONT
					}
				}
			}
			if c.Fill == "" {
				c.Fill = WHITE
			}
			resolved[ri][ci] = c
		}
	}
	s.node.Shapes = append(s.node.Shapes, ir.Table{
		X: x, Y: y, ColW: append([]float64(nil), colW...), RowH: rowH,
		Rows: resolved, DefSize: defSize, BorderColor: LINEC,
	})
}

// copyParas snapshots paragraph/run slices so callers can reuse or mutate
// their buffers after the builder call (the pre-IR implementation rendered
// immediately and had the same property).
func copyParas(paras []Para) []Para {
	out := make([]Para, len(paras))
	for i, p := range paras {
		p.Runs = append([]Run(nil), p.Runs...)
		out[i] = p
	}
	return out
}

// ---------- output ----------

// Save writes the deck as a .pptx file at path.
func (d *Deck) Save(path string) error {
	return pptx.Write(path, d.IR())
}
