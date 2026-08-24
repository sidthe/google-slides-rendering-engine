package htmlconv

import (
	"fmt"
	"strings"

	"github.com/sidthe/google-slides-rendering-engine/gslides"
	"github.com/sidthe/google-slides-rendering-engine/ir"
)

// The HTML deck contract: sections are 1600x900 CSS px, so px == design
// grid units 1:1. Point values differ: the 1600-unit slide is 13.333in wide
// (120 px/in), so 1 CSS px = 72/120 = 0.6 pt.
const pxToPt = 0.6

// Palette fallbacks matching the root package's defaults, used where HTML
// gave no color.
const (
	fallbackInk     = "5F6368" // INK2
	fallbackLine    = "DADCE0" // LINEC
	fallbackFill    = "FFFFFF"
	fallbackFont    = "Roboto"
	sceneBackground = "FFFFFF"
)

// MapScenes converts extracted scenes to IR. The second return value lists
// every lossy or suspicious conversion; it never fails — a warning-heavy
// deck still renders.
func MapScenes(res *Result) (*ir.Deck, []string) {
	var warns []string
	warns = append(warns, res.Warnings...)
	d := &ir.Deck{Theme: defaultTheme()}
	for si, sc := range res.Scenes {
		slide := &ir.Slide{Notes: sc.Notes}
		prefix := fmt.Sprintf("slide %d: ", si+1)
		if sc.W < 1590 || sc.W > 1610 || sc.H < 890 || sc.H > 910 {
			warns = append(warns, fmt.Sprintf("%ssection is %.0fx%.0f, expected 1600x900 — geometry maps 1px=1grid-unit regardless", prefix, sc.W, sc.H))
		}
		if sc.Background != "" && sc.Background != sceneBackground {
			slide.Shapes = append(slide.Shapes, ir.Rect{
				Frame: ir.Frame{X: 0, Y: 0, W: 1600, H: 900},
				Fill:  sc.Background,
			})
		}
		for _, w := range sc.Warnings {
			warns = append(warns, prefix+w)
		}
		for _, el := range sc.Elements {
			if el.Kind != "line" && (el.X < -5 || el.Y < -5 || el.X+el.W > 1605 || el.Y+el.H > 905) {
				warns = append(warns, fmt.Sprintf("%s%s at (%.0f,%.0f) %.0fx%.0f extends outside the slide", prefix, el.Kind, el.X, el.Y, el.W, el.H))
			}
			switch el.Kind {
			case "box":
				slide.Shapes = append(slide.Shapes, mapBox(el, &warns, prefix))
			case "line":
				slide.Shapes = append(slide.Shapes, ir.Line{
					X1: el.X1, Y1: el.Y1, X2: el.X2, Y2: el.Y2,
					Color: el.Color, Wpt: el.WidthPx * pxToPt,
					Arrow: el.Arrow, ArrowStart: el.ArrowStart, Dashed: el.Dashed,
				})
			case "elbow":
				slide.Shapes = append(slide.Shapes, ir.Elbow{
					X1: el.X1, Y1: el.Y1, X2: el.X2, Y2: el.Y2,
					Color: el.Color, Wpt: el.WidthPx * pxToPt,
					Arrow: el.Arrow, ArrowStart: el.ArrowStart, Dashed: el.Dashed,
				})
			case "text":
				slide.Shapes = append(slide.Shapes, mapText(el))
			case "table":
				slide.Shapes = append(slide.Shapes, mapTable(el))
			default:
				warns = append(warns, prefix+"unknown element kind "+el.Kind)
			}
		}
		d.Slides = append(d.Slides, slide)
	}
	return d, warns
}

// presetNames maps data-shape values (with a few friendly aliases) to IR
// preset names (PPTX prstGeom vocabulary).
var presetNames = map[string]string{
	"diamond":       "diamond",
	"cylinder":      "can",
	"can":           "can",
	"db":            "can",
	"hexagon":       "hexagon",
	"parallelogram": "parallelogram",
	"cloud":         "cloud",
	"trapezoid":     "trapezoid",
	"chevron":       "chevron",
	"pentagon":      "homePlate",
	"homeplate":     "homePlate",
	"triangle":      "triangle",
	"callout":       "wedgeRectCallout",
}

func mapBox(el Element, warns *[]string, prefix string) ir.Shape {
	if el.Shape != "" {
		preset, ok := presetNames[strings.ToLower(el.Shape)]
		if !ok {
			*warns = append(*warns, prefix+"unknown data-shape "+el.Shape+", using rectangle")
			preset = ""
		}
		if preset != "" {
			return ir.AutoShape{
				Frame:  ir.Frame{X: el.X, Y: el.Y, W: el.W, H: el.H},
				Preset: preset,
				Fill:   el.Fill, Line: el.Border, LineWpt: el.BorderW * pxToPt,
				Dashed: el.Dashed, FillAlpha: el.FillAlpha,
			}
		}
	}
	if el.Oval {
		return ir.Oval{
			Frame: ir.Frame{X: el.X, Y: el.Y, W: el.W, H: el.H},
			Fill:  el.Fill, Line: el.Border, LineWpt: el.BorderW * pxToPt,
			Dashed: el.Dashed, FillAlpha: el.FillAlpha,
		}
	}
	return ir.Rect{
		Frame: ir.Frame{X: el.X, Y: el.Y, W: el.W, H: el.H},
		Fill:  el.Fill, Line: el.Border, LineWpt: el.BorderW * pxToPt,
		Radius: el.Radius, Dashed: el.Dashed, FillAlpha: el.FillAlpha,
	}
}

func mapText(el Element) ir.Text {
	return ir.Text{
		Frame:    ir.Frame{X: el.X, Y: el.Y, W: el.W, H: el.H},
		Paras:    mapParas(el.Paras, el.Align, el.Spacing),
		DefSize:  el.FontSize * pxToPt,
		DefColor: orDefault(el.FontColor, fallbackInk),
		DefFont:  gslides.ResolveFont(orDefault(el.FontFam, fallbackFont)),
		Rot:      el.Rot,
	}
}

func mapTable(el Element) ir.Table {
	rows := make([][]ir.Cell, len(el.Rows))
	for ri, row := range el.Rows {
		rows[ri] = make([]ir.Cell, len(row))
		for ci, c := range row {
			align := c.Align
			if align == "l" {
				align = "" // left is the default; keep IR minimal
			}
			paras := mapParas(c.Paras, align, 0)
			for pi := range paras {
				for qi := range paras[pi].Runs {
					r := &paras[pi].Runs[qi]
					r.Color = orDefault(r.Color, fallbackInk)
					r.Font = orDefault(r.Font, fallbackFont)
				}
			}
			rows[ri][ci] = ir.Cell{Fill: orDefault(c.Fill, fallbackFill), Paras: paras}
		}
	}
	return ir.Table{
		X: el.X, Y: el.Y,
		ColW: el.ColW, RowH: el.RowH,
		Rows:        rows,
		DefSize:     el.DefSize * pxToPt,
		BorderColor: orDefault(el.Border, fallbackLine),
	}
}

func mapParas(paras []Para, align string, spacing float64) []ir.Para {
	out := make([]ir.Para, len(paras))
	for i, p := range paras {
		runs := make([]ir.Run, len(p.Runs))
		for j, r := range p.Runs {
			runs[j] = ir.Run{
				Text: r.Text, Bold: r.Bold, Italic: r.Italic,
				Underline: r.Underline, Strike: r.Strike,
				Color: r.Color,
				Size:  r.Size * pxToPt,
				Font:  gslides.ResolveFont(orDefault(r.Font, fallbackFont)),
				Link:  r.Link,
			}
		}
		out[i] = ir.Para{Runs: runs, Align: align, Spacing: spacing, Bullet: p.Bullet, Level: p.Level}
	}
	return out
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func defaultTheme() ir.Theme {
	return ir.Theme{
		Name: "deck-engine", Font: fallbackFont,
		Dark1: "202124", Light1: "FFFFFF", Dark2: "5F6368", Light2: "F8F9FA",
		Accents:   [6]string{"4285F4", "EA4335", "FBBC04", "34A853", "1967D2", "188038"},
		Hyperlink: "1967D2", FollowedLink: "681DA8",
	}
}
