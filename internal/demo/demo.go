// Package demo builds the canonical three-slide demo deck: title slide,
// stat tiles + bar chart, and a table + code block. It is shared by
// example/main.go and the golden tests so both exercise the same content.
package demo

import (
	deck "github.com/deck-engine/deck-engine"
)

// Build returns the demo deck.
func Build() *deck.Deck {
	d := deck.New()

	// 1 — title
	s := d.Slide()
	deck.Dots(s, 80, 96)
	s.Text(80, 300, 1440, 90, []deck.Para{deck.P(deck.RC("deck-engine", deck.INK))}, 54, deck.INK, deck.FONT)
	s.Text(80, 400, 1200, 40, []deck.Para{deck.P(deck.RC("Native, editable PPTX decks from Go, zero dependencies", deck.INK2))}, 18, deck.INK2, deck.FONT)
	deck.Pill(s, 80, 470, 130, "DEMO DECK", deck.BLUE50, deck.BLUE7)
	deck.Foot(s, "deck-engine example", 1)

	// 2 — tiles + chart
	s = d.Slide()
	deck.Kicker(s, "components", 52)
	deck.Title(s, "Stat tiles and a shape-drawn bar chart")
	deck.Tile(s, 80, 180, 340, 150, "98.6%", "of shapes rendered natively (editable in Slides and PowerPoint)", deck.GREEN50, deck.GREEND, 40)
	deck.Tile(s, 440, 180, 340, 150, "0", "external dependencies beyond the Go standard library", deck.BLUE50, deck.BLUE7, 40)
	deck.Tile(s, 800, 180, 340, 150, "1600x900", "design grid: position shapes like CSS pixels", deck.YELL50, deck.YELLD, 30)
	deck.Card(s, 80, 370, 1440, 430, deck.BLUE, "Latency by build (illustrative)", deck.BLUE7)
	deck.BarChart(s, 130, 450, 1300, 300,
		[]string{"v1", "v2", "v3", "v4"},
		[]deck.BarSeries{
			{Name: "p50", Color: deck.BLUE, Values: []float64{42, 31, 22, 18}},
			{Name: "p90", Color: deck.YELL, Values: []float64{80, 61, 40, 27}},
		},
		100, []float64{0, 25, 50, 75, 100}, "ms", 1)
	deck.Legend(s, 130, 760, [][2]string{{"p50", deck.BLUE}, {"p90", deck.YELL}})
	deck.Foot(s, "deck-engine example", 2)

	// 3 — table + code
	s = d.Slide()
	deck.Kicker(s, "components", 52)
	deck.Title(s, "Tables and code blocks")
	hdr := func(t string) deck.Cell {
		return deck.Cell{Paras: []deck.Para{deck.P(deck.BC(t, deck.INK))}, Fill: deck.SURF2}
	}
	cell := func(t string) deck.Cell { return deck.Cell{Paras: []deck.Para{deck.P(deck.R(t))}} }
	s.Table(80, 180, []float64{300, 560, 300}, 44, [][]deck.Cell{
		{hdr("Component"), hdr("What it draws"), hdr("File")},
		{cell("Rect / Oval / Line"), cell("rounded rects, ellipses, connectors with arrows"), cell("deck.go")},
		{cell("Text"), cell("multi-paragraph rich text with per-run styling"), cell("deck.go")},
		{cell("Table"), cell("simple grids with per-cell fills"), cell("deck.go")},
		{cell("Tile / Card / BarChart"), cell("opinionated layout kit"), cell("style.go")},
	}, 11)
	deck.CodeBlock(s, 80, 480, 1440, 220, []string{
		"d := deck.New()",
		"s := d.Slide()",
		"deck.Title(s, \"Hello\")",
		"deck.Tile(s, 80, 180, 340, 150, \"42\", \"the answer\", deck.BLUE50, deck.BLUE7, 40)",
		"d.Save(\"out.pptx\")",
	}, 12)
	deck.Foot(s, "deck-engine example", 3)

	return d
}
