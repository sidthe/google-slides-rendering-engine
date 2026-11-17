package gslides_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deck-engine/deck-engine/gslides"
	"github.com/deck-engine/deck-engine/ir"
)

// TestGoldenFeatures covers the Tier 1+2 additions: nested bullets and
// numbered lists (tab-consumption index math), preset shapes, elbow
// connectors, hyperlinks, underline/strikethrough, alpha fills, rotation.
func TestGoldenFeatures(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{Shapes: []ir.Shape{
		ir.AutoShape{Frame: ir.Frame{X: 100, Y: 100, W: 200, H: 120}, Preset: "diamond", Fill: "E8F0FE", Line: "1967D2", LineWpt: 1},
		ir.AutoShape{Frame: ir.Frame{X: 400, Y: 100, W: 160, H: 140}, Preset: "can", Fill: "FCE8E6", Line: "C5221F", LineWpt: 1},
		ir.Rect{Frame: ir.Frame{X: 700, Y: 100, W: 300, H: 200}, Fill: "202124", FillAlpha: 0.35, Radius: 0.05},
		ir.Elbow{X1: 200, Y1: 300, X2: 500, Y2: 500, Color: "5F6368", Wpt: 1, Arrow: true},
		ir.Text{
			Frame: ir.Frame{X: 100, Y: 500, W: 600, H: 300}, DefSize: 12, DefColor: "202124", DefFont: "Roboto",
			Paras: []ir.Para{
				{Runs: []ir.Run{{Text: "Plain intro "}, {Text: "linked", Link: "https://example.com/docs"}}},
				{Bullet: "disc", Level: 0, Runs: []ir.Run{{Text: "first 🎯 bullet"}}},
				{Bullet: "disc", Level: 1, Runs: []ir.Run{{Text: "nested", Underline: true}}},
				{Bullet: "disc", Level: 0, Runs: []ir.Run{{Text: "back to top", Strike: true}}},
				{Bullet: "number", Level: 0, Runs: []ir.Run{{Text: "step one"}}},
				{Bullet: "number", Level: 0, Runs: []ir.Run{{Text: "step two"}}},
				{Runs: []ir.Run{{Text: "outro"}}},
			},
		},
		ir.Text{
			Frame: ir.Frame{X: 60, Y: 400, W: 200, H: 30}, DefSize: 10, DefColor: "5F6368", DefFont: "Roboto",
			Rot:   -90,
			Paras: []ir.Para{{Align: "ctr", Runs: []ir.Run{{Text: "latency (ms)"}}}},
		},
	}}}}
	checkGolden(t, "features", gslides.Build(d))
}

// TestBulletTabIndexing verifies the createParagraphBullets index math:
// leading tabs are consumed by each bullets request, so later requests and
// all style requests shift left accordingly.
func TestBulletTabIndexing(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{Shapes: []ir.Shape{ir.Text{
		Frame: ir.Frame{X: 0, Y: 0, W: 100, H: 100}, DefSize: 10, DefColor: "000000", DefFont: "Roboto",
		Paras: []ir.Para{
			{Bullet: "disc", Level: 0, Runs: []ir.Run{{Text: "aa"}}},  // "aa\n"
			{Bullet: "disc", Level: 1, Runs: []ir.Run{{Text: "bb"}}},  // "\tbb\n"
			{Bullet: "number", Level: 0, Runs: []ir.Run{{Text: "c"}}}, // "c\n"
		},
	}}}}}
	var inserts []*gslides.InsertTextRequest
	var bullets []*gslides.CreateParagraphBulletsRequest
	var styles []*gslides.UpdateTextStyleRequest
	for _, r := range gslides.Build(d).Requests() {
		if r.InsertText != nil {
			inserts = append(inserts, r.InsertText)
		}
		if r.CreateParagraphBullets != nil {
			bullets = append(bullets, r.CreateParagraphBullets)
		}
		if r.UpdateTextStyle != nil {
			styles = append(styles, r.UpdateTextStyle)
		}
	}
	if len(inserts) != 1 || inserts[0].Text != "aa\n\tbb\nc" {
		t.Fatalf("insert text: %q", inserts[0].Text)
	}
	if len(bullets) != 2 {
		t.Fatalf("bullet requests: got %d want 2", len(bullets))
	}
	// disc group covers "aa\n\tbb\n" = [0,7) at insertion time.
	if *bullets[0].TextRange.StartIndex != 0 || *bullets[0].TextRange.EndIndex != 7 {
		t.Errorf("disc range: [%d,%d)", *bullets[0].TextRange.StartIndex, *bullets[0].TextRange.EndIndex)
	}
	// number group starts after the disc group consumed 1 tab: "c\n" was
	// [7,9) pre-consumption -> [6,8).
	if *bullets[1].TextRange.StartIndex != 6 || *bullets[1].TextRange.EndIndex != 8 {
		t.Errorf("number range: [%d,%d)", *bullets[1].TextRange.StartIndex, *bullets[1].TextRange.EndIndex)
	}
	// run styles use fully tab-less indices: "aa\nbb\nc" -> runs at
	// [0,2) [3,5) [6,7).
	wantRanges := [][2]int{{0, 2}, {3, 5}, {6, 7}}
	if len(styles) != 3 {
		t.Fatalf("style requests: got %d want 3", len(styles))
	}
	for i, s := range styles {
		if *s.TextRange.StartIndex != wantRanges[i][0] || *s.TextRange.EndIndex != wantRanges[i][1] {
			t.Errorf("run %d range: [%d,%d) want %v", i, *s.TextRange.StartIndex, *s.TextRange.EndIndex, wantRanges[i])
		}
	}
}

// TestPushNotes verifies the follow-up GET + insertText flow for speaker
// notes.
func TestPushNotes(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{
		{Notes: "Open with the incident story."},
		{}, // no notes
	}}
	var notesInserts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/presentations" && r.Method == "POST":
			json.NewEncoder(w).Encode(map[string]any{
				"presentationId": "p1",
				"slides":         []map[string]string{{"objectId": "def"}},
			})
		case r.URL.Path == "/v1/presentations/p1:batchUpdate":
			var body struct {
				Requests []struct {
					InsertText *gslides.InsertTextRequest `json:"insertText"`
				} `json:"requests"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			for _, q := range body.Requests {
				if q.InsertText != nil && q.InsertText.ObjectID == "notes_1" {
					notesInserts = append(notesInserts, q.InsertText.Text)
				}
			}
			w.Write([]byte("{}"))
		case r.URL.Path == "/v1/presentations/p1" && r.Method == "GET":
			json.NewEncoder(w).Encode(map[string]any{
				"slides": []map[string]any{
					{"slideProperties": map[string]any{"notesPage": map[string]any{"notesProperties": map[string]any{"speakerNotesObjectId": "notes_1"}}}},
					{"slideProperties": map[string]any{"notesPage": map[string]any{"notesProperties": map[string]any{"speakerNotesObjectId": "notes_2"}}}},
				},
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	if _, err := gslides.PushTo(context.Background(), srv.Client(), srv.URL, d, "t"); err != nil {
		t.Fatal(err)
	}
	if len(notesInserts) != 1 || notesInserts[0] != "Open with the incident story." {
		t.Errorf("notes inserts: %q", notesInserts)
	}
}

func TestBuildEmitsBulletIndentsSpaceBelowAndPerRowHeights(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{Shapes: []ir.Shape{
		ir.Text{
			Frame: ir.Frame{X: 80, Y: 82, W: 1440, H: 60}, DefSize: 30, DefColor: "202124", DefFont: "Roboto",
			WrapSlack: true,
			Paras:     []ir.Para{{Runs: []ir.Run{{Text: "Clamped title"}}}},
		},
		ir.Text{
			Frame: ir.Frame{X: 80, Y: 200, W: 420, H: 180}, DefSize: 12, DefColor: "5F6368", DefFont: "Roboto",
			Paras: []ir.Para{
				{Bullet: "disc", Level: 0, BulletIndent: 15.6, Spacing: 1.125, SpaceAfter: 4.8, Runs: []ir.Run{{Text: "first"}}},
				{Bullet: "disc", Level: 1, BulletIndent: 9.6, BulletMargin: 27.6, Spacing: 1.125, Runs: []ir.Run{{Text: "nested"}}},
			},
		},
		ir.Text{
			Frame: ir.Frame{X: 80, Y: 390, W: 200, H: 30}, DefSize: 12, DefColor: "5F6368", DefFont: "Roboto",
			Paras: []ir.Para{{Bullet: "disc", Level: 0}}, // single empty bullet item
		},
		ir.Table{
			X: 80, Y: 420, ColW: []float64{200}, RowH: 60, RowHs: []float64{40, 100, 40}, DefSize: 11,
			BorderColor: "DADCE0",
			Rows: [][]ir.Cell{
				{{Paras: []ir.Para{{Runs: []ir.Run{{Text: "a", Color: "202124", Font: "Roboto"}}}}}},
				{{Paras: []ir.Para{{Runs: []ir.Run{{Text: "b", Color: "202124", Font: "Roboto"}}}}}},
				{{Paras: []ir.Para{{Runs: []ir.Run{{Text: "c", Color: "202124", Font: "Roboto"}}}}}},
			},
		},
	}}}}
	reqs := gslides.Build(d).Requests()

	var paraStyles []*gslides.UpdateParagraphStyleRequest
	var rowProps []*gslides.UpdateTableRowPropertiesRequest
	var titleShape *gslides.CreateShapeRequest
	for _, r := range reqs {
		if r.CreateShape != nil && titleShape == nil {
			titleShape = r.CreateShape
		}
		if r.UpdateParagraphStyle != nil {
			paraStyles = append(paraStyles, r.UpdateParagraphStyle)
		}
		if r.UpdateTableRowProperties != nil {
			rowProps = append(rowProps, r.UpdateTableRowProperties)
		}
	}
	// Title at X=80, W=1440 clamped to X+W=1600 -> width = 1520*5715 + 2*91440 = 8869680 EMU.
	if got := titleShape.ElementProperties.Size.Width.Magnitude; got != 8869680 {
		t.Errorf("clamped title width = %.0f EMU, want 8869680", got)
	}
	if len(paraStyles) != 2 {
		t.Fatalf("para style requests = %d, want 2", len(paraStyles))
	}
	// Level 0: indentStart = 15.6*0.75 = 11.7pt, indentFirstLine = (15.6-9.6)*0.75 = 4.5pt, spaceBelow = 4.8*0.75 = 3.6pt.
	p0 := paraStyles[0].Style
	if p0.IndentStart == nil || p0.IndentStart.Magnitude != 11.7 || p0.IndentFirstLine == nil || p0.IndentFirstLine.Magnitude != 4.5 {
		t.Errorf("level 0 indents = start:%+v first:%+v, want 11.7pt / 4.5pt", p0.IndentStart, p0.IndentFirstLine)
	}
	if p0.SpaceBelow == nil || p0.SpaceBelow.Magnitude != 3.6 {
		t.Errorf("level 0 spaceBelow = %+v, want 3.6pt", p0.SpaceBelow)
	}
	// Level 1 (non-uniform nested): BulletMargin 27.6*0.75 = 20.7pt, step 9.6pt -> indentFirstLine = (27.6-9.6)*0.75 = 13.5pt.
	p1 := paraStyles[1].Style
	if p1.IndentStart == nil || p1.IndentStart.Magnitude != 20.7 || p1.IndentFirstLine == nil || p1.IndentFirstLine.Magnitude != 13.5 {
		t.Errorf("level 1 indents = start:%+v first:%+v, want 20.7pt / 13.5pt", p1.IndentStart, p1.IndentFirstLine)
	}
	if p1.SpaceBelow != nil {
		t.Errorf("final bullet should omit spaceBelow, got %+v", p1.SpaceBelow)
	}
	// RowHs [40, 100, 40] -> grouped into 2 UpdateTableRowProperties requests ([0,2] at 40 and [1] at 100).
	if len(rowProps) != 2 || len(rowProps[0].RowIndices) != 2 || len(rowProps[1].RowIndices) != 1 {
		t.Errorf("row height grouping = %+v, want 2 groups ([0,2] and [1])", rowProps)
	}
}
