package htmlconv

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A block label followed by direct text is a common card pattern. The label
// and body must become separate, non-overlapping text boxes; emitting the
// parent's full frame puts the body back at the top of the card in Slides.
func TestExtractUsesPaintedBoundsForDirectText(t *testing.T) {
	if os.Getenv("DECKGEN_SKIP_CHROME") != "" {
		t.Skip("DECKGEN_SKIP_CHROME set")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "card.html")
	html := `<!doctype html><style>
section { position:relative; width:1600px; height:900px; font:20px Roboto, sans-serif; }
.card { position:absolute; left:100px; top:100px; width:400px; height:200px; text-align:center; }
.card b { display:block; margin-bottom:20px; }
</style><section><div class="card"><b>Label</b>Body text</div></section>`
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Extract(context.Background(), path)
	if err != nil {
		t.Skipf("chrome unavailable: %v", err)
	}
	var labelY, bodyY float64
	labelCount := 0
	for _, el := range res.Scenes[0].Elements {
		if el.Kind != "text" || len(el.Paras) == 0 || len(el.Paras[0].Runs) == 0 {
			continue
		}
		text := el.Paras[0].Runs[0].Text
		switch {
		case strings.Contains(text, "Label"):
			labelY, labelCount = el.Y, labelCount+1
		case strings.Contains(text, "Body text"):
			bodyY = el.Y
		}
	}
	if labelCount != 1 {
		t.Fatalf("label emitted %d time(s), want 1", labelCount)
	}
	if bodyY <= labelY {
		t.Fatalf("body y=%.1f should follow label y=%.1f", bodyY, labelY)
	}
}
