package gslides_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sidthe/google-slides-rendering-engine/gslides"
	"github.com/sidthe/google-slides-rendering-engine/ir"
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
