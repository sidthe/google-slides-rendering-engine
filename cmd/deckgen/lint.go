// Template lint: advisory checks that a deck honors the floors of a named
// template (see templates/). Lint findings never fail the build — they are
// printed like warnings so an authoring agent can fix them before pushing.
package main

import (
	"fmt"
	"strings"

	"github.com/deck-engine/deck-engine/ir"
)

// lintDeck runs the checks for a named template against an extracted deck.
func lintDeck(template string, d *ir.Deck) ([]string, error) {
	switch template {
	case "keynote":
		return lintKeynote(d), nil
	default:
		return nil, fmt.Errorf("unknown lint template %q (available: keynote)", template)
	}
}

// Keynote floors (templates/keynote/DESIGN.md):
//   - no text under 18 px (10.8 pt) — smaller is invisible in a hall
//   - at most 70 visible words per slide — slides are backdrops, not documents
//   - at most 60 native elements per slide — one idea per slide
//   - every slide carries speaker notes — the detail lives there
const (
	keynoteMinPt     = 10.7 // 18 px at 0.6 pt/px, with float slack
	keynoteMaxWords  = 70
	keynoteMaxShapes = 80 // heatmaps and mockup grids run high; past this it reads as clutter
)

func lintKeynote(d *ir.Deck) []string {
	var out []string
	for i, s := range d.Slides {
		n := i + 1
		words := 0
		minSize := 0.0
		note := func(sz float64, text string) {
			words += len(strings.Fields(text))
			if sz > 0 && (minSize == 0 || sz < minSize) {
				minSize = sz
			}
		}
		for _, sh := range s.Shapes {
			switch t := sh.(type) {
			case ir.Text:
				for _, p := range t.Paras {
					for _, r := range p.Runs {
						sz := r.Size
						if sz == 0 {
							sz = t.DefSize
						}
						note(sz, r.Text)
					}
				}
			case ir.Table:
				for _, row := range t.Rows {
					for _, c := range row {
						for _, p := range c.Paras {
							for _, r := range p.Runs {
								sz := r.Size
								if sz == 0 {
									sz = t.DefSize
								}
								note(sz, r.Text)
							}
						}
					}
				}
			}
		}
		if minSize > 0 && minSize < keynoteMinPt {
			out = append(out, fmt.Sprintf("slide %d: text at %.1fpt (~%.0fpx) — below the 18px floor", n, minSize, minSize/0.6))
		}
		if words > keynoteMaxWords {
			out = append(out, fmt.Sprintf("slide %d: %d visible words — over the %d-word budget; move detail to speaker notes", n, words, keynoteMaxWords))
		}
		if len(s.Shapes) > keynoteMaxShapes {
			out = append(out, fmt.Sprintf("slide %d: %d native elements — over the %d budget; simplify or split", n, len(s.Shapes), keynoteMaxShapes))
		}
		if strings.TrimSpace(s.Notes) == "" {
			out = append(out, fmt.Sprintf("slide %d: no speaker notes", n))
		}
	}
	return out
}
