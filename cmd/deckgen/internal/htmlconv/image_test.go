package htmlconv

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deck-engine/deck-engine/ir"
)

// 1x1 red PNG as a data URI — data: sources don't taint the canvas, so this
// exercises the embed path without depending on file:// access flags.
const tinyPNGDataURI = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg=="

func TestExtractEmbeddedImage(t *testing.T) {
	html := `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>` +
		`<section style="position:relative;width:1600px;height:900px">` +
		`<img data-embed src="` + tinyPNGDataURI + `" style="position:absolute;left:100px;top:150px;width:200px;height:120px">` +
		`<img src="` + tinyPNGDataURI + `" style="position:absolute;left:400px;top:150px;width:50px;height:50px">` +
		`</section></body></html>`

	path := filepath.Join(t.TempDir(), "img.html")
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Extract(context.Background(), path)
	if err != nil {
		t.Skipf("chrome unavailable: %v", err)
	}
	if len(res.Scenes) != 1 {
		t.Fatalf("scenes: got %d want 1", len(res.Scenes))
	}
	sc := res.Scenes[0]

	var img *Element
	for i := range sc.Elements {
		if sc.Elements[i].Kind == "image" {
			if img != nil {
				t.Fatal("more than one image element — the non-data-embed img should be skipped")
			}
			img = &sc.Elements[i]
		}
	}
	if img == nil {
		t.Fatal("no image element extracted for <img data-embed>")
	}
	if img.X != 100 || img.Y != 150 || img.W != 200 || img.H != 120 {
		t.Errorf("image geometry: got (%v,%v) %vx%v want (100,150) 200x120", img.X, img.Y, img.W, img.H)
	}
	if img.PNG == "" {
		t.Error("image payload empty")
	}

	warned := false
	for _, w := range sc.Warnings {
		if strings.Contains(w, "data-embed") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("plain <img> should warn with the data-embed hint; warnings: %v", sc.Warnings)
	}

	deck, _ := MapScenes(res)
	var pics int
	for _, sh := range deck.Slides[0].Shapes {
		if p, ok := sh.(ir.Picture); ok {
			pics++
			if len(p.PNG) == 0 {
				t.Error("mapped Picture has no PNG bytes")
			}
		}
	}
	if pics != 1 {
		t.Errorf("mapped pictures: got %d want 1", pics)
	}
}
