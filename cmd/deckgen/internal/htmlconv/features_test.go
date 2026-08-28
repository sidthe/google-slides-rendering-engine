package htmlconv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/deck-engine/deck-engine/ir"
)

// TestMapScenesFeatures pins the Tier 1+2 conversions from a committed
// extraction of examples/features.html: preset shapes, elbows, bullets
// with nesting, links, underline/strike, alpha fills, rotation, notes.
// Regenerate with:
//
//	deckgen check examples/features.html -scene cmd/deckgen/internal/htmlconv/testdata/features-scene.json
func TestMapScenesFeatures(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "features-scene.json"))
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
	if len(d.Slides) != 3 {
		t.Fatalf("slides: got %d want 3", len(d.Slides))
	}

	for i, s := range d.Slides {
		if s.Notes == "" {
			t.Errorf("slide %d: missing speaker notes", i+1)
		}
	}

	presets := map[string]int{}
	var elbows int
	var bulletParas, numberParas, nested int
	var links, underlines, strikes int
	var alphaFills int
	var rotated int
	for _, s := range d.Slides {
		for _, sh := range s.Shapes {
			switch v := sh.(type) {
			case ir.AutoShape:
				presets[v.Preset]++
			case ir.Elbow:
				elbows++
			case ir.Rect:
				if v.FillAlpha > 0 && v.FillAlpha < 1 {
					alphaFills++
				}
			case ir.Text:
				if v.Rot != 0 {
					rotated++
				}
				for _, p := range v.Paras {
					switch p.Bullet {
					case "disc":
						bulletParas++
					case "number":
						numberParas++
					}
					if p.Level > 0 {
						nested++
					}
					for _, r := range p.Runs {
						if r.Link != "" {
							links++
						}
						if r.Underline {
							underlines++
						}
						if r.Strike {
							strikes++
						}
					}
				}
			}
		}
	}
	want := map[string]int{"diamond": 1, "can": 2, "hexagon": 1, "chevron": 3, "homePlate": 1, "trapezoid": 2, "triangle": 1, "wedgeRectCallout": 1}
	for k, n := range want {
		if presets[k] != n {
			t.Errorf("preset %s: got %d want %d", k, presets[k], n)
		}
	}
	if elbows != 2 {
		t.Errorf("elbows: got %d want 2", elbows)
	}
	if bulletParas != 5 || numberParas != 3 || nested != 2 {
		t.Errorf("bullets: disc=%d (want 5) number=%d (want 3) nested=%d (want 2)", bulletParas, numberParas, nested)
	}
	if links != 1 {
		t.Errorf("links: got %d want 1", links)
	}
	if underlines < 2 || strikes < 1 {
		t.Errorf("emphasis: underline=%d (want >=2) strike=%d (want >=1)", underlines, strikes)
	}
	if alphaFills != 1 {
		t.Errorf("alpha fills: got %d want 1", alphaFills)
	}
	if rotated != 1 {
		t.Errorf("rotated text: got %d want 1", rotated)
	}
}
