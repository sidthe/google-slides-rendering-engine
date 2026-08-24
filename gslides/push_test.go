package gslides_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	deck "github.com/sidthe/google-slides-rendering-engine"
	"github.com/sidthe/google-slides-rendering-engine/gslides"
)

func TestPushEndpointsAndChunking(t *testing.T) {
	// 300 rects = 600 requests, plus per-slide bootstrap and the
	// default-slide delete — forces multiple batches at the 400 cap.
	d := deck.New()
	s := d.Slide()
	for i := 0; i < 300; i++ {
		s.Rect(float64(i), 10, 20, 20, deck.BLUE50, "", 0.1)
	}

	var paths []string
	var batches [][]map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1/presentations":
			json.NewEncoder(w).Encode(map[string]any{
				"presentationId": "pres1",
				"slides":         []map[string]string{{"objectId": "defaultSlide"}},
			})
		case "/v1/presentations/pres1:batchUpdate":
			var body struct {
				Requests []map[string]json.RawMessage `json:"requests"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("bad batch body: %v", err)
			}
			batches = append(batches, body.Requests)
			w.Write([]byte("{}"))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	res, err := gslides.PushTo(context.Background(), srv.Client(), srv.URL, d.IR(), "t")
	if err != nil {
		t.Fatal(err)
	}
	if res.PresentationID != "pres1" || res.URL != "https://docs.google.com/presentation/d/pres1/edit" {
		t.Errorf("bad result: %+v", res)
	}
	if len(paths) < 3 || paths[0] != "/v1/presentations" {
		t.Fatalf("expected create then batches, got %v", paths)
	}

	var all []map[string]json.RawMessage
	for _, b := range batches {
		if len(b) > 400 {
			t.Errorf("batch exceeds cap: %d", len(b))
		}
		all = append(all, b...)
	}
	if len(batches) < 2 {
		t.Errorf("expected multiple batches, got %d", len(batches))
	}
	// First request overall must delete the default slide before anything
	// references the new ones.
	if _, ok := all[0]["deleteObject"]; !ok {
		t.Errorf("first request should be deleteObject, got %v", keys(all[0]))
	}
	// Group atomicity: every createShape must be immediately followed by
	// its updateShapeProperties in the same batch.
	for _, b := range batches {
		for i, r := range b {
			if _, ok := r["createShape"]; !ok {
				continue
			}
			if i+1 >= len(b) {
				t.Fatalf("createShape is last request of its batch — group split across chunks")
			}
			if _, ok := b[i+1]["updateShapeProperties"]; !ok {
				t.Fatalf("createShape not followed by updateShapeProperties in same batch")
			}
		}
	}
}

func keys(m map[string]json.RawMessage) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
