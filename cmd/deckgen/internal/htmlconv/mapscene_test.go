package htmlconv

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/sidthe/google-slides-rendering-engine/ir"
)

var update = flag.Bool("update", false, "rewrite golden files")

func loadFixture(t *testing.T) *Result {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "demo-scene.json"))
	if err != nil {
		t.Fatal(err)
	}
	var res Result
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatal(err)
	}
	return &res
}

// TestMapScenesGolden pins the scene -> IR conversion against a committed
// extraction of examples/demo.html, so the mapper is fully tested without
// Chrome. Regenerate the fixture with:
//
//	deckgen check examples/demo.html -scene cmd/deckgen/internal/htmlconv/testdata/demo-scene.json
func TestMapScenesGolden(t *testing.T) {
	d, warns := MapScenes(loadFixture(t))
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	got, err := json.MarshalIndent(d, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", "demo-ir.json")
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run `go test ./internal/htmlconv -update`)", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("IR differs from %s", path)
	}
}

func TestMapScenesShapes(t *testing.T) {
	d, _ := MapScenes(loadFixture(t))
	if len(d.Slides) != 3 {
		t.Fatalf("slides: got %d want 3", len(d.Slides))
	}
	counts := map[string]int{}
	for _, s := range d.Slides {
		for _, sh := range s.Shapes {
			switch sh.(type) {
			case ir.Rect:
				counts["rect"]++
			case ir.Oval:
				counts["oval"]++
			case ir.Text:
				counts["text"]++
			case ir.Line:
				counts["line"]++
			case ir.Table:
				counts["table"]++
			}
		}
	}
	// The demo exercises every element kind.
	if counts["oval"] != 4 {
		t.Errorf("ovals (dots): got %d want 4", counts["oval"])
	}
	if counts["table"] != 1 {
		t.Errorf("tables: got %d want 1", counts["table"])
	}
	if counts["line"] != 3 {
		t.Errorf("lines (hairlines): got %d want 3", counts["line"])
	}
	if counts["rect"] == 0 || counts["text"] == 0 {
		t.Errorf("missing rects/texts: %v", counts)
	}
	// Table cells carry italic/bold runs from slide 3.
	var tbl ir.Table
	for _, sh := range d.Slides[2].Shapes {
		if v, ok := sh.(ir.Table); ok {
			tbl = v
		}
	}
	var sawBold, sawItalic bool
	for _, row := range tbl.Rows {
		for _, c := range row {
			for _, p := range c.Paras {
				for _, r := range p.Runs {
					sawBold = sawBold || r.Bold
					sawItalic = sawItalic || r.Italic
				}
			}
		}
	}
	if !sawBold || !sawItalic {
		t.Errorf("table runs should include bold and italic spans (bold=%v italic=%v)", sawBold, sawItalic)
	}
}

func TestMapTextOnlyAddsWrapSlackToOneVisualLine(t *testing.T) {
	oneLine := mapText(Element{Kind: "text", W: 220, H: 28, FontSize: 24})
	if !oneLine.WrapSlack {
		t.Fatal("single visual line should retain native-renderer wrap slack")
	}
	twoLines := mapText(Element{Kind: "text", W: 800, H: 108, FontSize: 48})
	if twoLines.WrapSlack {
		t.Fatal("multi-line visual text must not gain width and unwrap after import")
	}
}

// TestExtractLive runs the real Chrome extraction; skipped when Chrome is
// unavailable or DECKGEN_SKIP_CHROME is set.
func TestExtractLive(t *testing.T) {
	if os.Getenv("DECKGEN_SKIP_CHROME") != "" {
		t.Skip("DECKGEN_SKIP_CHROME set")
	}
	res, err := Extract(context.Background(), filepath.Join("..", "..", "..", "..", "examples", "demo.html"))
	if err != nil {
		t.Skipf("chrome unavailable: %v", err)
	}
	if len(res.Scenes) != 3 {
		t.Fatalf("scenes: got %d want 3", len(res.Scenes))
	}
	d, warns := MapScenes(res)
	if len(warns) != 0 {
		t.Errorf("warnings: %v", warns)
	}
	if len(d.Slides) != 3 {
		t.Errorf("slides: got %d want 3", len(d.Slides))
	}
}
