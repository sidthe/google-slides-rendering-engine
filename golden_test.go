package deck_test

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/deck-engine/deck-engine/internal/demo"
)

// TestGoldenPackage byte-compares every part of the demo deck's package
// against the golden capture in testdata/golden/pptx. The goldens were
// frozen from the pre-refactor string-emission implementation; this test is
// the acceptance gate that the IR refactor preserves output exactly.
func TestGoldenPackage(t *testing.T) {
	out := filepath.Join(t.TempDir(), "demo.pptx")
	if err := demo.Build().Save(out); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	got := map[string][]byte{}
	var order []string
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		got[f.Name] = b
		order = append(order, f.Name)
	}

	pf, err := os.Open("testdata/golden/pptx/parts.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer pf.Close()
	var want []string
	sc := bufio.NewScanner(pf)
	for sc.Scan() {
		if l := sc.Text(); l != "" {
			want = append(want, l)
		}
	}

	if len(order) != len(want) {
		t.Errorf("part count: got %d want %d", len(order), len(want))
	}
	for i, name := range want {
		if i < len(order) && order[i] != name {
			t.Errorf("part order[%d]: got %s want %s", i, order[i], name)
		}
		g, ok := got[name]
		if !ok {
			t.Errorf("missing part %s", name)
			continue
		}
		w, err := os.ReadFile(filepath.Join("testdata/golden/pptx", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(g, w) {
			t.Errorf("part %s differs from golden (%d vs %d bytes)", name, len(g), len(w))
		}
	}
}
