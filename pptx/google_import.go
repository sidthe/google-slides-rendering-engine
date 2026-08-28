package pptx

import (
	"io"
	"os"

	"github.com/sidthe/google-slides-rendering-engine/ir"
)

// Google Slides imports widescreen PPTX files onto its fixed 10in x 5.625in
// canvas. It scales shape geometry during that conversion but keeps DrawingML
// point magnitudes (fonts and strokes) unchanged. That mismatch makes imported
// text wrap early and overlap nearby labels. WriteGoogleSlides emits a PPTX on
// the native Slides canvas and scales the point-based properties to preserve
// the source HTML's proportions after import.
const googleImportScale = 0.75

const (
	googleSlidesW = 9144000 // 10in
	googleSlidesH = 5143500 // 5.625in
)

// WriteGoogleSlides writes a PPTX intended for Google Slides import.
func WriteGoogleSlides(path string, d *ir.Deck) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteGoogleSlidesTo(f, d)
}

// WriteGoogleSlidesTo is WriteGoogleSlides for an arbitrary destination.
func WriteGoogleSlidesTo(w io.Writer, d *ir.Deck) error {
	return writeTo(w, scaleForGoogleImport(d), googleSlidesW, googleSlidesH)
}

func scaleForGoogleImport(d *ir.Deck) *ir.Deck {
	out := &ir.Deck{Theme: d.Theme, Slides: make([]*ir.Slide, len(d.Slides))}
	for si, sl := range d.Slides {
		copySlide := &ir.Slide{Notes: sl.Notes, Shapes: make([]ir.Shape, 0, len(sl.Shapes))}
		for _, sh := range sl.Shapes {
			copySlide.Shapes = append(copySlide.Shapes, scaleShape(sh))
		}
		out.Slides[si] = copySlide
	}
	return out
}

func scaleFrame(f ir.Frame) ir.Frame {
	return ir.Frame{X: f.X * googleImportScale, Y: f.Y * googleImportScale, W: f.W * googleImportScale, H: f.H * googleImportScale}
}

func scaleParas(in []ir.Para) []ir.Para {
	out := make([]ir.Para, len(in))
	for i, p := range in {
		out[i] = p
		out[i].Runs = make([]ir.Run, len(p.Runs))
		for j, r := range p.Runs {
			out[i].Runs[j] = r
			if r.Size > 0 {
				out[i].Runs[j].Size = r.Size * googleImportScale
			}
		}
	}
	return out
}

func scaleShape(sh ir.Shape) ir.Shape {
	switch v := sh.(type) {
	case ir.Rect:
		v.Frame = scaleFrame(v.Frame)
		v.LineWpt *= googleImportScale
		return v
	case ir.Oval:
		v.Frame = scaleFrame(v.Frame)
		v.LineWpt *= googleImportScale
		return v
	case ir.AutoShape:
		v.Frame = scaleFrame(v.Frame)
		v.LineWpt *= googleImportScale
		return v
	case ir.Text:
		v.Frame = scaleFrame(v.Frame)
		v.Paras = scaleParas(v.Paras)
		v.DefSize *= googleImportScale
		return v
	case ir.Line:
		v.X1 *= googleImportScale
		v.Y1 *= googleImportScale
		v.X2 *= googleImportScale
		v.Y2 *= googleImportScale
		v.Wpt *= googleImportScale
		return v
	case ir.Elbow:
		v.X1 *= googleImportScale
		v.Y1 *= googleImportScale
		v.X2 *= googleImportScale
		v.Y2 *= googleImportScale
		v.Wpt *= googleImportScale
		return v
	case ir.Table:
		v.X *= googleImportScale
		v.Y *= googleImportScale
		v.RowH *= googleImportScale
		v.DefSize *= googleImportScale
		v.ColW = append([]float64(nil), v.ColW...)
		for i := range v.ColW {
			v.ColW[i] *= googleImportScale
		}
		// Keep the source rows while constructing the scaled deep copy. Ranging
		// over v.Rows after replacing it produces the right number of empty rows
		// — and silently drops every cell during Google Slides import.
		rows := v.Rows
		v.Rows = make([][]ir.Cell, len(rows))
		for ri, row := range rows {
			v.Rows[ri] = make([]ir.Cell, len(row))
			for ci, c := range row {
				v.Rows[ri][ci] = c
				v.Rows[ri][ci].Paras = scaleParas(c.Paras)
			}
		}
		return v
	default:
		return sh
	}
}
