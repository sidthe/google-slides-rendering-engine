package deck

import (
	"archive/zip"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveProducesValidPackage(t *testing.T) {
	d := New()
	s := d.Slide()
	Title(s, "Test")
	s.Rect(80, 200, 400, 100, BLUE50, LINEC, 0.1)
	s.Text(80, 320, 800, 40, []Para{P(B("bold "), R("plain"), BC("blue", BLUE7))}, 12, INK, FONT)
	s.Line(80, 400, 600, 400, INK3, 1, true, false)
	s.Table(80, 440, []float64{200, 200}, 40, [][]Cell{
		{{Paras: []Para{P(B("h1"))}, Fill: SURF2}, {Paras: []Para{P(B("h2"))}, Fill: SURF2}},
		{{Paras: []Para{P(R("a"))}}, {Paras: []Para{P(R("b"))}}},
	}, 11)
	d.Slide() // empty second slide

	out := filepath.Join(t.TempDir(), "t.pptx")
	if err := d.Save(out); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}
	defer zr.Close()
	want := map[string]bool{
		"[Content_Types].xml":   false,
		"ppt/presentation.xml":  false,
		"ppt/theme/theme1.xml":  false,
		"ppt/slides/slide1.xml": false,
		"ppt/slides/slide2.xml": false,
	}
	for _, f := range zr.File {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("%s: %v", f.Name, err)
			}
			rc.Close()
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing part %s", name)
		}
	}
}

func TestThemeCustomization(t *testing.T) {
	d := New()
	d.Theme.Font = "Inter"
	d.Theme.Accents[0] = "112233"
	if got := d.theme().xml(); !strings.Contains(got, "Inter") || !strings.Contains(got, "112233") {
		t.Error("theme xml does not reflect customization")
	}
	var zero Deck
	if got := zero.theme(); got.Font != FONT {
		t.Errorf("zero deck should fall back to default theme, got font %q", got.Font)
	}
}
