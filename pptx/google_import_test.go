package pptx

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/deck-engine/deck-engine/ir"
)

func TestWriteGoogleSlidesScalesCanvasAndPointMagnitudes(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{Shapes: []ir.Shape{
		ir.Rect{Frame: ir.Frame{X: 80, Y: 100, W: 400, H: 120}, Line: "000000", LineWpt: 2},
		ir.Text{Frame: ir.Frame{X: 80, Y: 140, W: 400, H: 30}, DefSize: 12, DefColor: "000000", DefFont: "Roboto", Paras: []ir.Para{{Runs: []ir.Run{{Text: "fits after import"}}}}},
	}}}}
	var out bytes.Buffer
	if err := WriteGoogleSlidesTo(&out, d); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{}
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
		parts[f.Name] = string(b)
	}
	if !strings.Contains(parts["ppt/presentation.xml"], `p:sldSz cx="9144000" cy="5143500"`) {
		t.Fatalf("Google Slides canvas not emitted: %s", parts["ppt/presentation.xml"])
	}
	slide := parts["ppt/slides/slide1.xml"]
	if !strings.Contains(slide, `<a:off x="457200" y="571500"/>`) { // 80x100 grid units at 0.75 scale
		t.Fatalf("geometry not scaled for Google import: %s", slide)
	}
	if !strings.Contains(slide, `sz="900"`) { // 12pt * 0.75
		t.Fatalf("font size not scaled for Google import: %s", slide)
	}
	if !strings.Contains(slide, `<a:ln w="19050"`) { // 2pt * 0.75
		t.Fatalf("stroke width not scaled for Google import: %s", slide)
	}
}

func TestWriteGoogleSlidesPreservesTableCells(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{Shapes: []ir.Shape{ir.Table{
		X: 80, Y: 100, ColW: []float64{200, 300}, RowH: 40, DefSize: 12,
		BorderColor: "DADCE0",
		Rows: [][]ir.Cell{{
			{Fill: "202124", Paras: []ir.Para{{Runs: []ir.Run{{Text: "Header", Color: "FFFFFF", Font: "Roboto", Bold: true}}}}},
			{Paras: []ir.Para{{Runs: []ir.Run{{Text: "Value", Color: "202124", Font: "Roboto"}}}}},
		}},
	}}}}}
	var out bytes.Buffer
	if err := WriteGoogleSlidesTo(&out, d); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if f.Name != "ppt/slides/slide1.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		xml := string(b)
		for _, want := range []string{"<a:tc>", "Header", "Value", `w="1143000"`} { // first column: 200 × .75
			if !strings.Contains(xml, want) {
				t.Fatalf("Google-import table lost %q: %s", want, xml)
			}
		}
		return
	}
	t.Fatal("slide XML not found")
}
