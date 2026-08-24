package gslides_test

import (
	"testing"

	"github.com/sidthe/google-slides-rendering-engine/gslides"
	"github.com/sidthe/google-slides-rendering-engine/ir"
)

// TestGoldenDiagramPrimitives covers the connector/outline features that
// diagram-style decks (sequence, architecture) rely on: start/both-end
// arrows, dashed outlines on boxes, and diagonal lines.
func TestGoldenDiagramPrimitives(t *testing.T) {
	d := &ir.Deck{Slides: []*ir.Slide{{Shapes: []ir.Shape{
		// dashed group boundary
		ir.Rect{Frame: ir.Frame{X: 80, Y: 100, W: 600, H: 400}, Line: "80868B", LineWpt: 1, Dashed: true, Radius: 0.04},
		// reply arrow: head at the left end
		ir.Line{X1: 700, Y1: 200, X2: 100, Y2: 200, Color: "5F6368", Wpt: 1, Arrow: true, Dashed: true},
		// bidirectional link
		ir.Line{X1: 100, Y1: 300, X2: 700, Y2: 300, Color: "1967D2", Wpt: 1.5, Arrow: true, ArrowStart: true},
		// diagonal connector
		ir.Line{X1: 120, Y1: 420, X2: 640, Y2: 180, Color: "188038", Wpt: 1, Arrow: true},
	}}}}
	checkGolden(t, "diagram", gslides.Build(d))
}
