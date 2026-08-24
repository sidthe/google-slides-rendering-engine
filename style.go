package deck

// The house-style kit: opinionated layout components built on the raw shape
// emitters. All coordinates are on the 1600x900 grid. Everything here is a
// convenience — decks can ignore this file entirely and place shapes
// directly.

import (
	"fmt"
	"math"
	"strings"
)

// Kicker draws a small uppercase eyebrow line.
func Kicker(s *Slide, t string, y float64) {
	s.Text(80, y, 1300, 26, []Para{P(BC(strings.ToUpper(t), BLUE7))}, 12, BLUE7, FONT)
}

// Title draws the slide title at the standard position.
func Title(s *Slide, t string) {
	s.Text(80, 82, 1440, 66, []Para{P(RC(t, INK))}, 31, INK, FONT)
}

// Foot draws the footer line and slide number.
func Foot(s *Slide, left string, n int) {
	s.Text(80, 862, 1150, 22, []Para{P(R(left))}, 9.5, INK3, FONT)
	s.Text(1380, 862, 140, 22, []Para{{Runs: []Run{R(fmt.Sprint(n))}, Align: "r"}}, 9.5, INK3, FONT)
}

// Card draws a surface card with an optional accent strip and title row.
func Card(s *Slide, x, y, w, h float64, accent, titleTxt, titleColor string) {
	s.Rect(x, y, w, h, SURF2, LINEC, 0.06)
	if accent != "" {
		s.Rect(x+2, y, w-4, 7, accent, "", 0.5)
	}
	if titleTxt != "" {
		s.Text(x+20, y+18, w-40, 26, []Para{P(BC(titleTxt, titleColor))}, 14, titleColor, FONT)
	}
}

// Tile draws a stat tile: big number + label on a tinted background.
func Tile(s *Slide, x, y, w, h float64, n, label, bg, ncolor string, nsize float64) {
	s.Rect(x, y, w, h, bg, "", 0.08)
	s.Text(x+22, y+16, w-44, nsize+14, []Para{P(RC(n, ncolor))}, nsize, ncolor, FONT)
	s.Text(x+22, y+nsize+34, w-44, h-nsize-46, []Para{{Runs: []Run{R(label)}, Spacing: 1.1}}, 10.5, INK2, FONT)
}

// CodeBlock draws a dark terminal-style block of monospaced lines.
func CodeBlock(s *Slide, x, y, w, h float64, lines []string, size float64) {
	s.Rect(x, y, w, h, DARK, "", 0.05)
	paras := make([]Para, len(lines))
	for i, l := range lines {
		paras[i] = Para{Runs: []Run{R(l)}, Spacing: 1.2}
	}
	s.Text(x+22, y+18, w-44, h-36, paras, size, CODET, MONO)
}

// Pill draws a small rounded status chip with centered bold text.
func Pill(s *Slide, x, y, w float64, txt, bg, color string) {
	s.Rect(x, y, w, 26, bg, "", 0.5)
	s.Text(x, y+4, w, 20, []Para{{Runs: []Run{BC(txt, color)}, Align: "ctr"}}, 9, color, FONT)
}

// Dots draws the four-color accent dots.
func Dots(s *Slide, x, y float64) {
	for i, c := range []string{BLUE, RED, YELL, GREEN} {
		s.Oval(x+float64(i)*22, y, 14, 14, c, "", 0)
	}
}

// BarSeries is one series of a grouped column chart.
type BarSeries struct {
	Name   string
	Color  string
	Values []float64
}

// BarChart draws a shape-based grouped column chart with gridlines and
// direct value labels. maxV is the top of the scale; ticks are the gridline
// values; unit is appended to tick and value labels; catLines reserves room
// for multi-line category labels.
func BarChart(s *Slide, x, y, w, h float64, cats []string, series []BarSeries,
	maxV float64, ticks []float64, unit string, catLines int) {
	plotL, plotT := x+45.0, y
	plotW, plotH := w-55.0, h-30.0-float64(catLines-1)*16
	for _, tv := range ticks {
		ty := plotT + plotH - tv/maxV*plotH
		s.Line(plotL, ty, x+w, ty, LINEC, 0.6, false, false)
		s.Text(x, ty-9, 38, 18, []Para{{Runs: []Run{R(fmt.Sprintf("%g%s", tv, unit))}, Align: "r"}}, 9.5, INK3, FONT)
	}
	nCat, nSer := len(cats), len(series)
	gw := plotW / float64(nCat)
	bw := math.Min(46, gw/float64(nSer)*0.55)
	for ci, cat := range cats {
		cx := plotL + gw*float64(ci) + gw/2
		grpW := bw*float64(nSer) + 3*float64(nSer-1)
		for si, ser := range series {
			v := ser.Values[ci]
			bh := v / maxV * plotH
			bx := cx - grpW/2 + float64(si)*(bw+3)
			s.Rect(bx, plotT+plotH-bh, bw, bh, ser.Color, "", 0.10)
			s.Text(bx-30, plotT+plotH-bh-22, bw+60, 18,
				[]Para{{Runs: []Run{BC(trimFloat(v)+unit, INK)}, Align: "ctr"}}, 10, INK, FONT)
		}
		s.Text(cx-gw/2, plotT+plotH+8, gw, 20+float64(catLines-1)*16,
			[]Para{{Runs: []Run{R(cat)}, Align: "ctr", Spacing: 1.0}}, 10, INK2, FONT)
	}
}

// Legend draws swatch+label pairs in a row.
func Legend(s *Slide, x, y float64, items [][2]string) {
	cx := x
	for _, it := range items {
		s.Rect(cx, y+3, 12, 12, it[1], "", 0.25)
		s.Text(cx+18, y, 340, 20, []Para{P(R(it[0]))}, 10, INK2, FONT)
		cx += 46 + float64(len(it[0]))*7.6
	}
}

func trimFloat(v float64) string {
	if v == math.Trunc(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2g", v)
}
