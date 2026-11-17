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

func TestWriteGoogleSlidesScalesBulletsSpacingTrackingAndRowHeights(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{Shapes: []ir.Shape{
		ir.Text{
			Frame: ir.Frame{X: 80, Y: 82, W: 1440, H: 60}, DefSize: 30, DefColor: "202124", DefFont: "Roboto",
			WrapSlack: true,
			Paras: []ir.Para{{Runs: []ir.Run{
				{Text: "TRACKED", LetterSpacing: 1.2},
				{Text: " tight", LetterSpacing: -0.9},
			}}},
		},
		ir.Text{
			Frame: ir.Frame{X: 80, Y: 200, W: 420, H: 180}, DefSize: 12, DefColor: "5F6368", DefFont: "Roboto",
			Paras: []ir.Para{
				{Bullet: "disc", Level: 0, BulletIndent: 15.6, Spacing: 1.125, SpaceAfter: 4.8, Runs: []ir.Run{{Text: "first\vline2"}}},
				{Bullet: "disc", Level: 1, BulletIndent: 9.6, BulletMargin: 27.6, Spacing: 1.125, Runs: []ir.Run{{Text: "nested"}}},
			},
		},
		ir.Text{
			Frame: ir.Frame{X: 4, Y: 380, W: 100, H: 24}, DefSize: 10, DefColor: "202124", DefFont: "Roboto",
			WrapSlack: true,
			Paras:     []ir.Para{{Align: "r", Runs: []ir.Run{{Text: "right"}}}},
		},
		ir.Text{
			Frame: ir.Frame{X: 1500, Y: 380, W: 96, H: 24}, DefSize: 10, DefColor: "202124", DefFont: "Roboto",
			WrapSlack: true,
			Paras:     []ir.Para{{Align: "ctr", Runs: []ir.Run{{Text: "center"}}}},
		},
		ir.Table{
			X: 80, Y: 420, ColW: []float64{200, 300}, RowH: 60, RowHs: []float64{40, 100}, DefSize: 11,
			BorderColor: "DADCE0",
			Rows: [][]ir.Cell{
				{{Paras: []ir.Para{{Spacing: 1.125, Runs: []ir.Run{{Text: "R1", Color: "202124", Font: "Roboto"}}}}}, {Paras: []ir.Para{{Runs: []ir.Run{{Text: "Short", Color: "202124", Font: "Roboto"}}}}}},
				{{Paras: []ir.Para{{}} /* blank pre line in cell */}, {Paras: []ir.Para{{Runs: []ir.Run{{Text: "Tall", Color: "202124", Font: "Roboto"}}}}}},
			},
		},
	}}}}
	var out bytes.Buffer
	if err := WriteGoogleSlidesTo(&out, d); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var xml string
	for _, f := range zr.File {
		if f.Name == "ppt/slides/slide1.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatal(err)
			}
			xml = string(b)
		}
	}
	wants := []string{
		// 80 + 1520 clamped to 1600 -> x=457200, cx=8686800 (457200+8686800 == 9144000)
		`<a:off x="457200" y="468630"/><a:ext cx="8686800" cy="342900"/>`,
		// 1.2pt * 0.75 = 0.9pt -> spc="90"; -0.9pt * 0.75 = -0.675pt -> spc="-68"
		`spc="90"`,
		`spc="-68"`,
		// 15.6pt * 0.75 = 11.7pt -> 148590 EMU (26 grid-px); non-uniform nested 27.6pt*0.75=20.7pt -> 262890 EMU, step 9.6pt*0.75=7.2pt -> 91440 EMU
		`marL="148590" indent="-148590"`,
		`marL="262890" indent="-91440"`,
		// Soft break <a:br> inside single bullet paragraph
		`<a:t>first</a:t></a:r><a:br><a:rPr lang="en-US" sz="900"`,
		// Right-aligned near X=0 clamped to x="0", cx=104*0.75*7620=594360; centered near X+W=1600 clamped symmetrically by 4 units -> X=1496, W=104
		`<a:off x="0" y="2171700"/><a:ext cx="594360" cy="137160"/>`,
		`<a:off x="8549640" y="2171700"/><a:ext cx="594360" cy="137160"/>`,
		// Blank pre spacer paragraph inside table cell gets endParaRPr at scaled DefSize (11*0.75=8.25 -> 825)
		`<a:p><a:endParaRPr lang="en-US" sz="825"/></a:p>`,
		// OOXML pPr sequence: lnSpc -> spcAft (4.8pt * 0.75 = 3.6pt -> 360) -> buFont
		`<a:lnSpc><a:spcPct val="112500"/></a:lnSpc><a:spcAft><a:spcPts val="360"/></a:spcAft><a:buFont`,
		// Per-row table heights: 40*0.75*7620 = 228600, 100*0.75*7620 = 571500, sum = 800100
		`<a:ext cx="2857500" cy="800100"/>`,
		`<a:tr h="228600">`,
		`<a:tr h="571500">`,
	}
	for _, want := range wants {
		if !strings.Contains(xml, want) {
			t.Errorf("missing %q in Google-import XML:\n%s", want, xml)
		}
	}
}
