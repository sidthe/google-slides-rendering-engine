package pptx

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/deck-engine/deck-engine/ir"
)

// tinyPNG is a valid 1x1 transparent PNG.
var tinyPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
	0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}

func TestWritePicture(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{
		Shapes: []ir.Shape{
			ir.Picture{Frame: ir.Frame{X: 100, Y: 100, W: 400, H: 300}, PNG: tinyPNG},
		},
	}}}

	var buf bytes.Buffer
	if err := WriteTo(&buf, d); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
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

	media, ok := parts["ppt/media/image1.png"]
	if !ok {
		t.Fatal("ppt/media/image1.png missing from package")
	}
	if !bytes.Equal([]byte(media), tinyPNG) {
		t.Error("media part bytes differ from the source PNG")
	}
	if !strings.Contains(parts["[Content_Types].xml"], `<Default Extension="png" ContentType="image/png"/>`) {
		t.Error("png default missing from [Content_Types].xml")
	}
	slide := parts["ppt/slides/slide1.xml"]
	if !strings.Contains(slide, `<p:pic>`) || !strings.Contains(slide, `r:embed="rIdImg1"`) {
		t.Errorf("slide1.xml missing picture frame or embed rel:\n%s", slide)
	}
	rels := parts["ppt/slides/_rels/slide1.xml.rels"]
	if !strings.Contains(rels, `Id="rIdImg1"`) || !strings.Contains(rels, `Target="../media/image1.png"`) {
		t.Errorf("slide rels missing image relationship:\n%s", rels)
	}
}

// TestWriteNoPictureNoPngDefault keeps picture support zero-cost for decks
// without images: no png content-type default, no media parts.
func TestWriteNoPictureNoPngDefault(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{
		Shapes: []ir.Shape{ir.Rect{Frame: ir.Frame{X: 0, Y: 0, W: 100, H: 100}, Fill: "FFFFFF"}},
	}}}
	var buf bytes.Buffer
	if err := WriteTo(&buf, d); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "ppt/media/") {
			t.Errorf("unexpected media part %s", f.Name)
		}
		if f.Name == "[Content_Types].xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			if strings.Contains(string(b), `Extension="png"`) {
				t.Error("png default present without any pictures")
			}
		}
	}
}
