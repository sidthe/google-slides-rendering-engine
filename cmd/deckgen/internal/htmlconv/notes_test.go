package htmlconv

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExtractSpeakerNotes(t *testing.T) {
	css, err := filepath.Abs(filepath.Join("..", "..", "..", "assets", "deck.css"))
	if err != nil {
		t.Fatal(err)
	}

	html := `<!DOCTYPE html><html><head><meta charset="utf-8">` +
		`<link rel="stylesheet" href="file://` + css + `"></head><body>` +
		`<section>` +
		`<div class="title">Slide Title</div>` +
		`<aside class="notes">` +
		`<p><b>Executive Summary:</b></p>` +
		`<ul>` +
		`<li>First bullet point<br>with a continuation line</li>` +
		`<li>Second bullet point` +
		`<ul><li>Nested sub-bullet</li></ul>` +
		`</li>` +
		`</ul>` +
		`<p></p>` +
		`<p>Numbered steps:</p>` +
		`<ol>` +
		`<li>Step one</li>` +
		`<li>Step two</li>` +
		`</ol>` +
		`</aside>` +
		`</section></body></html>`

	path := filepath.Join(t.TempDir(), "notes.html")
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

	got := strings.Split(res.Scenes[0].Notes, "\n")
	want := []string{
		"Executive Summary:",
		"• First bullet point",
		"  with a continuation line",
		"• Second bullet point",
		"  - Nested sub-bullet",
		"",
		"Numbered steps:",
		"1. Step one",
		"2. Step two",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("speaker notes lines mismatch:\ngot:  %#v\nwant: %#v", got, want)
	}
}
