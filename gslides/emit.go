// Package gslides emits an ir.Deck as native Google Slides API batchUpdate
// requests: every IR shape becomes a real, editable Slides element (no
// images). Build is pure — no network — so the full request stream is
// golden-testable; Push (push.go) sends it with any authorized HTTP client.
package gslides

import (
	"math"
	"strings"

	"github.com/sidthe/google-slides-rendering-engine/ir"
)

// Group is an atomic run of requests for one object (create + text +
// styles) or one slide's bootstrap. Chunked sends must never split a group,
// so create-before-reference always holds.
type Group struct {
	Label    string
	Requests []Request
}

// Plan is the ordered request stream for a whole deck.
type Plan struct {
	Groups []Group
}

// Requests flattens the plan in order.
func (p *Plan) Requests() []Request {
	var out []Request
	for _, g := range p.Groups {
		out = append(out, g.Requests...)
	}
	return out
}

// Build converts the deck into its batchUpdate plan. It does not include
// deleting the presentation's default slide — only Push knows that slide's
// ID (it comes from the presentations.create response).
func Build(d *ir.Deck) *Plan {
	p := &Plan{}
	for si, sl := range d.Slides {
		slideID := objectID(si+1, 0)
		p.Groups = append(p.Groups, Group{Label: slideID, Requests: []Request{
			{CreateSlide: &CreateSlideRequest{
				ObjectID:             slideID,
				SlideLayoutReference: &LayoutReference{PredefinedLayout: "BLANK"},
			}},
			{UpdatePageProperties: &UpdatePagePropertiesRequest{
				ObjectID: slideID,
				PageProperties: &PageProperties{
					PageBackgroundFill: &PageBackgroundFill{SolidFill: solid("FFFFFF")},
				},
				Fields: "pageBackgroundFill.solidFill.color",
			}},
		}})
		for ei, sh := range sl.Shapes {
			id := objectID(si+1, ei+1)
			g := Group{Label: id}
			switch v := sh.(type) {
			case ir.Rect:
				g.Requests = emitRect(slideID, id, v)
			case ir.Oval:
				g.Requests = emitOval(slideID, id, v)
			case ir.AutoShape:
				g.Requests = emitAutoShape(slideID, id, v)
			case ir.Text:
				g.Requests = emitText(slideID, id, v)
			case ir.Line:
				g.Requests = emitLine(slideID, id, v)
			case ir.Elbow:
				g.Requests = emitElbow(slideID, id, v)
			case ir.Table:
				g.Requests = emitTable(slideID, id, v)
			}
			if len(g.Requests) > 0 {
				p.Groups = append(p.Groups, g)
			}
		}
	}
	return p
}

// ---------- shapes ----------

func frameProps(slideID string, f ir.Frame) *PageElementProperties {
	return &PageElementProperties{
		PageObjectID: slideID,
		Size:         &Size{Width: gridDim(f.W), Height: gridDim(f.H)},
		Transform: &AffineTransform{
			ScaleX: 1, ScaleY: 1,
			TranslateX: emu(f.X), TranslateY: emu(f.Y),
			Unit: "EMU",
		},
	}
}

// fillOutlineProps builds the updateShapeProperties request shared by
// Rect, Oval and AutoShape. lineWpt <= 0 keeps the API's default outline
// weight; fillAlpha in (0,1) makes the fill partially transparent.
func fillOutlineProps(id, fill, line string, lineWpt, fillAlpha float64, dashed bool, extra *ShapeProperties, extraFields []string) Request {
	sp := &ShapeProperties{}
	if extra != nil {
		*sp = *extra
	}
	fields := append([]string(nil), extraFields...)
	if fill != "" {
		sf := solid(fill)
		fields = append(fields, "shapeBackgroundFill.solidFill.color")
		if fillAlpha > 0 && fillAlpha < 1 {
			a := fillAlpha
			sf.Alpha = &a
			fields = append(fields, "shapeBackgroundFill.solidFill.alpha")
		}
		sp.ShapeBackgroundFill = &ShapeBackgroundFill{SolidFill: sf}
	} else {
		sp.ShapeBackgroundFill = &ShapeBackgroundFill{PropertyState: "NOT_RENDERED"}
		fields = append(fields, "shapeBackgroundFill.propertyState")
	}
	if line != "" {
		sp.Outline = &Outline{OutlineFill: &OutlineFill{SolidFill: solid(line)}}
		fields = append(fields, "outline.outlineFill.solidFill.color")
		if lineWpt > 0 {
			sp.Outline.Weight = ptDim(lineWpt)
			fields = append(fields, "outline.weight")
		}
		if dashed {
			sp.Outline.DashStyle = "DASH"
			fields = append(fields, "outline.dashStyle")
		}
	} else {
		sp.Outline = &Outline{PropertyState: "NOT_RENDERED"}
		fields = append(fields, "outline.propertyState")
	}
	return Request{UpdateShapeProperties: &UpdateShapePropertiesRequest{
		ObjectID:        id,
		ShapeProperties: sp,
		Fields:          strings.Join(fields, ","),
	}}
}

func emitRect(slideID, id string, v ir.Rect) []Request {
	shapeType := "RECTANGLE"
	if v.Radius > 0 {
		// The Slides API cannot set geometry adjust values, so the corner
		// radius is the ROUND_RECTANGLE default, not v.Radius.
		shapeType = "ROUND_RECTANGLE"
	}
	wpt := v.LineWpt
	if v.Line != "" && wpt <= 0 {
		wpt = 1 // historical PPTX default outline width
	}
	return []Request{
		{CreateShape: &CreateShapeRequest{ObjectID: id, ShapeType: shapeType, ElementProperties: frameProps(slideID, v.Frame)}},
		fillOutlineProps(id, v.Fill, v.Line, wpt, v.FillAlpha, v.Dashed, nil, nil),
	}
}

func emitOval(slideID, id string, v ir.Oval) []Request {
	return []Request{
		{CreateShape: &CreateShapeRequest{ObjectID: id, ShapeType: "ELLIPSE", ElementProperties: frameProps(slideID, v.Frame)}},
		fillOutlineProps(id, v.Fill, v.Line, v.LineWpt, v.FillAlpha, v.Dashed, nil, nil),
	}
}

// presetTypes maps IR preset names (PPTX prstGeom vocabulary) to Slides
// API shape types.
var presetTypes = map[string]string{
	"diamond":          "DIAMOND",
	"can":              "CAN",
	"hexagon":          "HEXAGON",
	"parallelogram":    "PARALLELOGRAM",
	"cloud":            "CLOUD",
	"trapezoid":        "TRAPEZOID",
	"chevron":          "CHEVRON",
	"homePlate":        "HOME_PLATE",
	"triangle":         "TRIANGLE",
	"wedgeRectCallout": "WEDGE_RECTANGLE_CALLOUT",
}

func emitAutoShape(slideID, id string, v ir.AutoShape) []Request {
	st, ok := presetTypes[v.Preset]
	if !ok {
		st = "RECTANGLE"
	}
	wpt := v.LineWpt
	if v.Line != "" && wpt <= 0 {
		wpt = 1
	}
	return []Request{
		{CreateShape: &CreateShapeRequest{ObjectID: id, ShapeType: st, ElementProperties: frameProps(slideID, v.Frame)}},
		fillOutlineProps(id, v.Fill, v.Line, wpt, v.FillAlpha, v.Dashed, nil, nil),
	}
}

// emitElbow renders an L-connector as two joined straight lines — the
// Slides API's BENT category is a fixed mid-point Z and its bend cannot be
// repositioned (no adjust values), so an L is not natively expressible.
// The PPTX backend uses a real bentConnector2 instead.
func emitElbow(slideID, id string, v ir.Elbow) []Request {
	vert := ir.Line{
		X1: v.X1, Y1: v.Y1, X2: v.X1, Y2: v.Y2,
		Color: v.Color, Wpt: v.Wpt, Dashed: v.Dashed, ArrowStart: v.ArrowStart,
	}
	horiz := ir.Line{
		X1: v.X1, Y1: v.Y2, X2: v.X2, Y2: v.Y2,
		Color: v.Color, Wpt: v.Wpt, Dashed: v.Dashed, Arrow: v.Arrow,
	}
	return append(emitLine(slideID, id+"a", vert), emitLine(slideID, id+"b", horiz)...)
}

func emitText(slideID, id string, v ir.Text) []Request {
	// Inset compensation: the API cannot zero a text box's internal insets
	// the way the PPTX writer does, so grow the frame by the default insets
	// to land the text itself on the grid position.
	w := emu(v.W) + 2*insetLR
	// Chrome's painted text bounds are exact, but Slides can need a little
	// more width for the same Roboto run. Add right-side slack to left-aligned
	// text only: titles and body copy avoid spurious wrapping while centered
	// diagram labels keep their visual center.
	if v.WrapSlack && isLeftAligned(v.Paras) {
		w += emu(v.W * 0.12)
	}
	h := emu(v.H) + 2*insetTB
	px, py := emu(v.X)-insetLR, emu(v.Y)-insetTB
	tr := &AffineTransform{ScaleX: 1, ScaleY: 1, TranslateX: px, TranslateY: py, Unit: "EMU"}
	if v.Rot != 0 {
		// Rotate about the box center (CSS default transform-origin), like
		// the PPTX emitter's xfrm rot.
		rad := v.Rot * math.Pi / 180
		cos, sin := round6(math.Cos(rad)), round6(math.Sin(rad))
		cx, cy := w/2, h/2
		tr = &AffineTransform{
			ScaleX: cos, ScaleY: cos, ShearX: -sin, ShearY: sin,
			TranslateX: math.Round(px + cx - (cos*cx - sin*cy)),
			TranslateY: math.Round(py + cy - (sin*cx + cos*cy)),
			Unit:       "EMU",
		}
	}
	props := &PageElementProperties{
		PageObjectID: slideID,
		Size:         &Size{Width: emuDim(w), Height: emuDim(h)},
		Transform:    tr,
	}
	reqs := []Request{
		{CreateShape: &CreateShapeRequest{ObjectID: id, ShapeType: "TEXT_BOX", ElementProperties: props}},
		{UpdateShapeProperties: &UpdateShapePropertiesRequest{
			ObjectID:        id,
			ShapeProperties: &ShapeProperties{Autofit: &Autofit{AutofitType: "NONE"}},
			Fields:          "autofit.autofitType",
		}},
	}
	return append(reqs, textRequests(id, nil, v.Paras, v.DefSize, v.DefColor, v.DefFont, true)...)
}

func isLeftAligned(paras []ir.Para) bool {
	return len(paras) == 0 || paras[0].Align == "" || paras[0].Align == "l"
}

func emitLine(slideID, id string, v ir.Line) []Request {
	dx, dy := v.X2-v.X1, v.Y2-v.Y1
	w, h := emu(math.Abs(dx)), emu(math.Abs(dy))
	if w == 0 {
		w = 1
	}
	if h == 0 {
		h = 1
	}
	sx, sy := 1.0, 1.0
	if dx < 0 {
		sx = -1
	}
	if dy < 0 {
		sy = -1
	}
	lp := &LineProperties{LineFill: &LineFill{SolidFill: solid(v.Color)}}
	fields := []string{"lineFill.solidFill.color"}
	if v.Wpt > 0 {
		lp.Weight = ptDim(v.Wpt)
		fields = append(fields, "weight")
	}
	if v.Dashed {
		lp.DashStyle = "DASH"
		fields = append(fields, "dashStyle")
	}
	if v.ArrowStart {
		lp.StartArrow = "FILL_ARROW"
		fields = append(fields, "startArrow")
	}
	if v.Arrow {
		lp.EndArrow = "FILL_ARROW"
		fields = append(fields, "endArrow")
	}
	return []Request{
		{CreateLine: &CreateLineRequest{
			ObjectID: id,
			Category: "STRAIGHT",
			ElementProperties: &PageElementProperties{
				PageObjectID: slideID,
				Size:         &Size{Width: emuDim(w), Height: emuDim(h)},
				Transform: &AffineTransform{
					ScaleX: sx, ScaleY: sy,
					TranslateX: emu(v.X1), TranslateY: emu(v.Y1),
					Unit: "EMU",
				},
			},
		}},
		{UpdateLineProperties: &UpdateLinePropertiesRequest{
			ObjectID:       id,
			LineProperties: lp,
			Fields:         strings.Join(fields, ","),
		}},
	}
}

func emitTable(slideID, id string, v ir.Table) []Request {
	rows, cols := len(v.Rows), len(v.ColW)
	if rows == 0 || cols == 0 {
		return nil
	}
	totW := 0.0
	for _, w := range v.ColW {
		totW += w
	}
	reqs := []Request{
		{CreateTable: &CreateTableRequest{
			ObjectID: id,
			ElementProperties: &PageElementProperties{
				PageObjectID: slideID,
				Size: &Size{
					Width:  gridDim(totW),
					Height: gridDim(v.RowH * float64(rows)),
				},
				Transform: &AffineTransform{
					ScaleX: 1, ScaleY: 1,
					TranslateX: emu(v.X), TranslateY: emu(v.Y),
					Unit: "EMU",
				},
			},
			Rows:    rows,
			Columns: cols,
		}},
	}
	// Column widths, grouping equal widths into one request.
	byWidth := map[float64][]int{}
	var widthOrder []float64
	for i, w := range v.ColW {
		if _, seen := byWidth[w]; !seen {
			widthOrder = append(widthOrder, w)
		}
		byWidth[w] = append(byWidth[w], i)
	}
	for _, w := range widthOrder {
		reqs = append(reqs, Request{UpdateTableColumnProperties: &UpdateTableColumnPropertiesRequest{
			ObjectID:              id,
			ColumnIndices:         byWidth[w],
			TableColumnProperties: &TableColumnProperties{ColumnWidth: gridDim(w)},
			Fields:                "columnWidth",
		}})
	}
	// Uniform (minimum) row height.
	allRows := make([]int, rows)
	for i := range allRows {
		allRows[i] = i
	}
	reqs = append(reqs, Request{UpdateTableRowProperties: &UpdateTableRowPropertiesRequest{
		ObjectID:           id,
		RowIndices:         allRows,
		TableRowProperties: &TableRowProperties{MinRowHeight: gridDim(v.RowH)},
		Fields:             "minRowHeight",
	}})
	// Borders: hairline between rows and below the table (matching the
	// PPTX writer's bottom-border-only cells), everything else transparent
	// to kill Slides' default 1pt black grid.
	hairline := &TableBorderProperties{
		TableBorderFill: &TableBorderFill{SolidFill: solid(v.BorderColor)},
		Weight:          ptDim(0.5),
	}
	invisible := &TableBorderProperties{
		TableBorderFill: &TableBorderFill{SolidFill: transparentFill()},
	}
	for _, pos := range []string{"INNER_HORIZONTAL", "BOTTOM"} {
		reqs = append(reqs, Request{UpdateTableBorderProperties: &UpdateTableBorderPropertiesRequest{
			ObjectID:              id,
			BorderPosition:        pos,
			TableBorderProperties: hairline,
			Fields:                "tableBorderFill.solidFill.color,weight",
		}})
	}
	for _, pos := range []string{"INNER_VERTICAL", "TOP", "LEFT", "RIGHT"} {
		reqs = append(reqs, Request{UpdateTableBorderProperties: &UpdateTableBorderPropertiesRequest{
			ObjectID:              id,
			BorderPosition:        pos,
			TableBorderProperties: invisible,
			Fields:                "tableBorderFill.solidFill.alpha",
		}})
	}
	// Cells: fill, then text. Colors and fonts were resolved by the builder.
	for ri, row := range v.Rows {
		for ci, c := range row {
			loc := &TableCellLocation{RowIndex: ri, ColumnIndex: ci}
			fill := solid(c.Fill)
			fields := "tableCellBackgroundFill.solidFill.color"
			if c.Fill == "" {
				fill = transparentFill()
				fields += ",tableCellBackgroundFill.solidFill.alpha"
			}
			reqs = append(reqs, Request{UpdateTableCellProperties: &UpdateTableCellPropertiesRequest{
				ObjectID: id,
				TableRange: &TableRange{
					Location:   *loc,
					RowSpan:    1,
					ColumnSpan: 1,
				},
				TableCellProperties: &TableCellProperties{
					TableCellBackgroundFill: &TableCellBackgroundFill{SolidFill: fill},
				},
				Fields: fields,
			}})
			reqs = append(reqs, textRequests(id, loc, c.Paras, v.DefSize, "", "", true)...)
		}
	}
	return reqs
}

// ---------- text ----------

var alignments = map[string]string{"l": "START", "ctr": "CENTER", "r": "END"}

// textRequests inserts and styles multi-paragraph text on a shape or table
// cell. All range indices are UTF-16 code units. Paragraph i's range
// includes its terminating newline (the last paragraph's newline is the
// implicit one Slides appends), which is what lets empty spacer paragraphs
// take the surrounding text size — the Slides analogue of the PPTX
// endParaRPr trick.
func textRequests(objID string, cell *TableCellLocation, paras []ir.Para, defSize float64, defColor, defFont string, withParaStyle bool) []Request {
	parts := make([]string, len(paras))
	tabs := make([]int, len(paras))
	for i, p := range paras {
		var b strings.Builder
		for _, r := range p.Runs {
			b.WriteString(r.Text)
		}
		parts[i] = b.String()
		if p.Bullet != "" {
			tabs[i] = p.Level
		}
	}

	// Inserted text carries Level leading tabs on bullet paragraphs;
	// createParagraphBullets reads them as nesting and consumes them, so
	// bullet ranges use insertion-time indices (shifted by tabs already
	// consumed by earlier bullet requests) while every later style request
	// uses tab-less indices.
	var fb strings.Builder
	for i, t := range parts {
		if i > 0 {
			fb.WriteString("\n")
		}
		fb.WriteString(strings.Repeat("\t", tabs[i]))
		fb.WriteString(t)
	}
	full := fb.String()
	if full == "" && len(paras) <= 1 {
		return nil
	}
	reqs := []Request{
		{InsertText: &InsertTextRequest{ObjectID: objID, CellLocation: cell, Text: full, InsertionIndex: 0}},
	}

	pre := make([]int, len(paras)) // paragraph start at insertion time
	pos := 0
	for i := range paras {
		pre[i] = pos
		pos += tabs[i] + len16(parts[i]) + 1
	}
	removed := 0
	for i := 0; i < len(paras); {
		if paras[i].Bullet == "" {
			i++
			continue
		}
		j, consumed := i, 0
		for j < len(paras) && paras[j].Bullet == paras[i].Bullet {
			consumed += tabs[j]
			j++
		}
		preset := "BULLET_DISC_CIRCLE_SQUARE"
		if paras[i].Bullet == "number" {
			preset = "NUMBERED_DIGIT_ALPHA_ROMAN"
		}
		end := pre[j-1] + tabs[j-1] + len16(parts[j-1]) + 1 - removed
		reqs = append(reqs, Request{CreateParagraphBullets: &CreateParagraphBulletsRequest{
			ObjectID: objID, CellLocation: cell,
			TextRange:    &Range{Type: "FIXED_RANGE", StartIndex: intp(pre[i] - removed), EndIndex: intp(end)},
			BulletPreset: preset,
		}})
		removed += consumed
		i = j
	}

	start := 0
	for i, p := range paras {
		paraLen := len16(parts[i])
		paraRange := &Range{Type: "FIXED_RANGE", StartIndex: intp(start), EndIndex: intp(start + paraLen + 1)}
		if withParaStyle {
			ps := &ParagraphStyle{}
			var fields []string
			if a, ok := alignments[p.Align]; ok && p.Align != "" {
				ps.Alignment = a
				fields = append(fields, "alignment")
			}
			if p.Spacing > 0 {
				ps.LineSpacing = math.Round(p.Spacing*100*1e4) / 1e4
				fields = append(fields, "lineSpacing")
			}
			if len(fields) > 0 {
				reqs = append(reqs, Request{UpdateParagraphStyle: &UpdateParagraphStyleRequest{
					ObjectID: objID, CellLocation: cell,
					Style: ps, TextRange: paraRange,
					Fields: strings.Join(fields, ","),
				}})
			}
		}
		if len(p.Runs) == 0 {
			// Blank spacer paragraph: size its newline like surrounding text.
			reqs = append(reqs, Request{UpdateTextStyle: &UpdateTextStyleRequest{
				ObjectID: objID, CellLocation: cell,
				Style:     &TextStyle{FontSize: ptDim(defSize)},
				TextRange: paraRange,
				Fields:    "fontSize",
			}})
		}
		rs := start
		for _, r := range p.Runs {
			runLen := len16(r.Text)
			if runLen == 0 {
				continue
			}
			size := defSize
			if r.Size > 0 {
				size = r.Size
			}
			color := defColor
			if r.Color != "" {
				color = r.Color
			}
			font := defFont
			if r.Font != "" {
				font = r.Font
			}
			bold := r.Bold
			style := &TextStyle{
				Bold:            &bold,
				FontSize:        ptDim(size),
				FontFamily:      ResolveFont(font),
				ForegroundColor: &OptionalColor{OpaqueColor: rgb(color)},
			}
			fields := "bold,fontSize,fontFamily,foregroundColor"
			if r.Italic {
				it := true
				style.Italic = &it
				fields += ",italic"
			}
			if r.Underline {
				u := true
				style.Underline = &u
				fields += ",underline"
			}
			if r.Strike {
				st := true
				style.Strikethrough = &st
				fields += ",strikethrough"
			}
			if r.Link != "" {
				style.Link = &Link{URL: r.Link}
				fields += ",link"
			}
			reqs = append(reqs, Request{UpdateTextStyle: &UpdateTextStyleRequest{
				ObjectID: objID, CellLocation: cell,
				Style:     style,
				TextRange: &Range{Type: "FIXED_RANGE", StartIndex: intp(rs), EndIndex: intp(rs + runLen)},
				Fields:    fields,
			}})
			rs += runLen
		}
		start += paraLen + 1
	}
	return reqs
}
