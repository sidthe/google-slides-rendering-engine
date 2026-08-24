package htmlconv

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/sidthe/google-slides-rendering-engine/ir"
)

// TestMapScenesDiagrams pins the diagram-critical conversions from a
// committed extraction of examples/diagrams.html: arrow direction,
// bidirectional arrows, dashed lifelines and group boundaries, and rotated
// divs becoming true diagonal lines. Regenerate with:
//
//	deckgen check examples/diagrams.html -scene cmd/deckgen/internal/htmlconv/testdata/diagrams-scene.json
func TestMapScenesDiagrams(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "diagrams-scene.json"))
	if err != nil {
		t.Fatal(err)
	}
	var res Result
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatal(err)
	}
	d, warns := MapScenes(&res)
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
	if len(d.Slides) != 2 {
		t.Fatalf("slides: got %d want 2", len(d.Slides))
	}

	var startArrows, endArrows, both, dashedLines, diagonals, dashedBoxes int
	for _, s := range d.Slides {
		for _, sh := range s.Shapes {
			switch v := sh.(type) {
			case ir.Line:
				if v.ArrowStart && v.Arrow {
					both++
				} else if v.ArrowStart {
					startArrows++
				} else if v.Arrow {
					endArrows++
				}
				if v.Dashed {
					dashedLines++
				}
				if math.Abs(v.X2-v.X1) > 1 && math.Abs(v.Y2-v.Y1) > 1 {
					diagonals++
				}
			case ir.Rect:
				if v.Dashed && v.Line != "" && v.Fill == "" {
					dashedBoxes++
				}
			}
		}
	}
	if startArrows != 3 { // three dashed return messages
		t.Errorf("start-arrow lines: got %d want 3", startArrows)
	}
	if both != 1 { // pod A <-> cache
		t.Errorf("bidirectional lines: got %d want 1", both)
	}
	if endArrows < 8 {
		t.Errorf("end-arrow lines: got %d want >= 8", endArrows)
	}
	if dashedLines != 7 { // 4 lifelines + 3 returns
		t.Errorf("dashed lines: got %d want 7", dashedLines)
	}
	if diagonals != 4 { // LB->pods, pods->db
		t.Errorf("diagonal lines: got %d want 4", diagonals)
	}
	if dashedBoxes != 1 { // VPC group boundary
		t.Errorf("dashed boundary rects: got %d want 1", dashedBoxes)
	}

	// Diagonal endpoint sanity: the LB->pod-A edge starts at LB's right
	// edge (~652,495) and ends near pod A's left edge (~762,345).
	var found bool
	for _, sh := range d.Slides[1].Shapes {
		if v, ok := sh.(ir.Line); ok {
			if math.Abs(v.X1-652) < 3 && math.Abs(v.Y1-495) < 3 &&
				math.Abs(v.X2-762) < 3 && math.Abs(v.Y2-345) < 3 {
				found = true
			}
		}
	}
	if !found {
		t.Error("LB->pod-A diagonal endpoints not found where the CSS rotation puts them")
	}
}
