package gslides_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	deck "github.com/sidthe/google-slides-rendering-engine"
	"github.com/sidthe/google-slides-rendering-engine/gslides"
	"github.com/sidthe/google-slides-rendering-engine/internal/demo"
	"github.com/sidthe/google-slides-rendering-engine/ir"
)

var update = flag.Bool("update", false, "rewrite golden files")

func checkGolden(t *testing.T, name string, plan *gslides.Plan) {
	t.Helper()
	got, err := json.MarshalIndent(plan.Requests(), "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", name+".json")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run `go test ./gslides -update` to create goldens)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("request stream differs from %s;\ngot:\n%s", path, got)
	}
}

// oneShape builds a single-slide deck via the public builder so all
// builder-level default resolution (table colors/fonts) is exercised.
func oneShape(build func(s *deck.Slide)) *ir.Deck {
	d := deck.New()
	build(d.Slide())
	return d.IR()
}

func TestGoldenRects(t *testing.T) {
	checkGolden(t, "rects", gslides.Build(oneShape(func(s *deck.Slide) {
		s.Rect(80, 180, 340, 150, deck.GREEN50, "", 0.08) // rounded, no outline
		s.Rect(500, 180, 200, 100, "", deck.LINEC, 0)     // square outline, no fill
	})))
}

func TestGoldenOval(t *testing.T) {
	checkGolden(t, "oval", gslides.Build(oneShape(func(s *deck.Slide) {
		s.Oval(100, 100, 14, 14, deck.BLUE, "", 0)
		s.Oval(200, 100, 60, 40, deck.YELL50, deck.YELLD, 2)
	})))
}

func TestGoldenText(t *testing.T) {
	checkGolden(t, "text", gslides.Build(oneShape(func(s *deck.Slide) {
		s.Text(80, 300, 1200, 200, []deck.Para{
			deck.P(deck.B("bold "), deck.R("plain "), deck.BC("blue", deck.BLUE7)),
			{}, // blank spacer paragraph
			{Runs: []deck.Run{deck.R("centered 🚀 emoji"), deck.RC(" tail", deck.RED)}, Align: "ctr", Spacing: 1.15},
			{Runs: []deck.Run{{Text: "mono", Font: deck.MONO, Size: 10}}, Align: "r"},
		}, 14, deck.INK, deck.FONT)
	})))
}

func TestGoldenLines(t *testing.T) {
	checkGolden(t, "lines", gslides.Build(oneShape(func(s *deck.Slide) {
		s.Line(80, 400, 600, 400, deck.INK3, 1, true, false)   // horizontal arrow
		s.Line(600, 500, 80, 450, deck.BLUE, 0.6, false, true) // right-to-left dashed (flipH+flipV)
	})))
}

func TestGoldenTable(t *testing.T) {
	checkGolden(t, "table", gslides.Build(oneShape(func(s *deck.Slide) {
		hdr := func(t string) deck.Cell {
			return deck.Cell{Paras: []deck.Para{deck.P(deck.BC(t, deck.INK))}, Fill: deck.SURF2}
		}
		cell := func(t string) deck.Cell { return deck.Cell{Paras: []deck.Para{deck.P(deck.R(t))}} }
		s.Table(80, 180, []float64{300, 560, 300}, 44, [][]deck.Cell{
			{hdr("A"), hdr("B"), hdr("C")},
			{cell("a1"), cell("b1"), {}}, // one empty cell
		}, 11)
	})))
}

func TestGoldenDemoDeck(t *testing.T) {
	checkGolden(t, "demo", gslides.Build(demo.Build().IR()))
}

// TestUTF16Ranges pins the emoji run-range math: '🚀' is two UTF-16 units,
// so styling ranges after it must shift by two, not one.
func TestUTF16Ranges(t *testing.T) {
	plan := gslides.Build(oneShape(func(s *deck.Slide) {
		s.Text(0, 0, 100, 20, []deck.Para{
			deck.P(deck.R("a🚀b"), deck.BC("x", deck.RED)),
		}, 10, deck.INK, deck.FONT)
	}))
	var styles []*gslides.UpdateTextStyleRequest
	for _, r := range plan.Requests() {
		if r.UpdateTextStyle != nil {
			styles = append(styles, r.UpdateTextStyle)
		}
	}
	if len(styles) != 2 {
		t.Fatalf("want 2 run styles, got %d", len(styles))
	}
	// "a🚀b" = 1+2+1 = 4 UTF-16 units.
	if got := *styles[0].TextRange.EndIndex; got != 4 {
		t.Errorf("first run end: got %d want 4", got)
	}
	if got := *styles[1].TextRange.StartIndex; got != 4 {
		t.Errorf("second run start: got %d want 4", got)
	}
	if got := *styles[1].TextRange.EndIndex; got != 5 {
		t.Errorf("second run end: got %d want 5", got)
	}
}
