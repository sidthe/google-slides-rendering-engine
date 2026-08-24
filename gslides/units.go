package gslides

import (
	"fmt"
	"math"
	"strconv"
)

// API-created presentations are fixed at 10in x 5.625in (9144000 x 5143500
// EMU), smaller than the 13.333in PPTX page. The 1600x900 design grid maps
// to exactly 5715 EMU per unit, and every point magnitude (font sizes, line
// weights) scales by 0.75 (= 9144000/12192000) so decks are proportionally
// identical to the PPTX output, not point-identical.
const (
	emuPerUnit = 5715.0
	ptScale    = 0.75

	// Default text insets of a Slides text box (PowerPoint convention:
	// 0.1in left/right, 0.05in top/bottom). The API cannot zero insets, so
	// text-box frames are compensated by these amounts; calibrated against
	// real renders.
	insetLR = 91440.0 // EMU
	insetTB = 45720.0 // EMU
)

func emu(v float64) float64 { return math.Round(v * emuPerUnit) }

func round6(v float64) float64 { return math.Round(v*1e6) / 1e6 }

// pt scales a PPTX point magnitude onto the smaller Slides page, rounded to
// 4 decimals for stable JSON.
func pt(v float64) float64 { return math.Round(v*ptScale*1e4) / 1e4 }

func ptDim(v float64) *Dimension  { return &Dimension{Magnitude: pt(v), Unit: "PT"} }
func emuDim(v float64) Dimension  { return Dimension{Magnitude: v, Unit: "EMU"} }
func gridDim(v float64) Dimension { return emuDim(emu(v)) }

// len16 returns the length of s in UTF-16 code units — the index space of
// every Slides API text range. Characters outside the BMP (emoji, some CJK)
// count as two units.
func len16(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// rgb converts a 6-digit hex color (no '#') to the API's float RGB,
// rounded to 6 decimals for stable JSON. Invalid input maps to black.
func rgb(hex string) *OpaqueColor {
	c := &RgbColor{}
	if len(hex) == 6 {
		if v, err := strconv.ParseUint(hex, 16, 32); err == nil {
			round := func(x float64) float64 { return math.Round(x*1e6) / 1e6 }
			c.Red = round(float64(v>>16&0xFF) / 255)
			c.Green = round(float64(v>>8&0xFF) / 255)
			c.Blue = round(float64(v&0xFF) / 255)
		}
	}
	return &OpaqueColor{RgbColor: c}
}

func solid(hex string) *SolidFill { return &SolidFill{Color: rgb(hex)} }

func transparentFill() *SolidFill {
	a := 0.0
	return &SolidFill{Color: rgb("FFFFFF"), Alpha: &a}
}

func intp(v int) *int { return &v }

func objectID(slide, elem int) string {
	if elem == 0 {
		return fmt.Sprintf("sl%03d", slide)
	}
	return fmt.Sprintf("sl%03de%03d", slide, elem)
}
