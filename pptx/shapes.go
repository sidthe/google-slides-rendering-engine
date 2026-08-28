package pptx

// Per-shape DrawingML emission, moved verbatim from the pre-IR root
// package; only the data source changed (ir fields instead of method
// parameters). Byte-for-byte output compatibility is enforced by the root
// golden test.

import (
	"fmt"
	"strings"

	"github.com/deck-engine/deck-engine/ir"
)

func dashXML(dashed bool) string {
	if dashed {
		return `<a:prstDash val="dash"/>`
	}
	return ""
}

// fillXML renders a solid fill, with optional partial transparency.
func fillXML(color string, alpha float64) string {
	if alpha > 0 && alpha < 1 {
		return fmt.Sprintf(`<a:solidFill><a:srgbClr val="%s"><a:alpha val="%d"/></a:srgbClr></a:solidFill>`,
			color, int(alpha*100000+0.5))
	}
	return fmt.Sprintf(`<a:solidFill><a:srgbClr val="%s"/></a:solidFill>`, color)
}

// slideRels accumulates per-slide relationships (hyperlinks); rId1 is
// always the layout, so links start at rId2.
type slideRels struct {
	links []string
}

func (r *slideRels) linkID(url string) string {
	for i, u := range r.links {
		if u == url {
			return fmt.Sprintf("rId%d", i+2)
		}
	}
	r.links = append(r.links, url)
	return fmt.Sprintf("rId%d", len(r.links)+1)
}

// shapeXML renders one IR shape as a DrawingML fragment. id is the shape's
// non-visual drawing ID (per-slide, starting at 2).
func shapeXML(sh ir.Shape, id int, rels *slideRels) string {
	switch v := sh.(type) {
	case ir.Rect:
		return rectXML(v, id)
	case ir.Oval:
		return ovalXML(v, id)
	case ir.AutoShape:
		return autoShapeXML(v, id)
	case ir.Text:
		return textXML(v, id, rels)
	case ir.Line:
		return lineXML(v, id)
	case ir.Elbow:
		return elbowXML(v, id)
	case ir.Table:
		return tableXML(v, id, rels)
	}
	return ""
}

func rectXML(v ir.Rect, id int) string {
	adj := int(v.Radius * 100000)
	lw := 12700
	if v.LineWpt > 0 {
		lw = int(v.LineWpt * 12700)
	}
	ln := "<a:ln><a:noFill/></a:ln>"
	if v.Line != "" {
		ln = fmt.Sprintf(`<a:ln w="%d"><a:solidFill><a:srgbClr val="%s"/></a:solidFill>%s</a:ln>`, lw, v.Line, dashXML(v.Dashed))
	}
	fl := "<a:noFill/>"
	if v.Fill != "" {
		fl = fillXML(v.Fill, v.FillAlpha)
	}
	return fmt.Sprintf(`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="rect"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>
<a:prstGeom prst="roundRect"><a:avLst><a:gd name="adj" fmla="val %d"/></a:avLst></a:prstGeom>
%s%s</p:spPr><p:txBody><a:bodyPr/><a:p/></p:txBody></p:sp>`,
		id, emu(v.X), emu(v.Y), emu(v.W), emu(v.H), adj, fl, ln)
}

func autoShapeXML(v ir.AutoShape, id int) string {
	ln := "<a:ln><a:noFill/></a:ln>"
	if v.Line != "" {
		lw := 12700
		if v.LineWpt > 0 {
			lw = int(v.LineWpt * 12700)
		}
		ln = fmt.Sprintf(`<a:ln w="%d"><a:solidFill><a:srgbClr val="%s"/></a:solidFill>%s</a:ln>`, lw, v.Line, dashXML(v.Dashed))
	}
	fl := "<a:noFill/>"
	if v.Fill != "" {
		fl = fillXML(v.Fill, v.FillAlpha)
	}
	return fmt.Sprintf(`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="%s"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>
<a:prstGeom prst="%s"><a:avLst/></a:prstGeom>
%s%s</p:spPr><p:txBody><a:bodyPr/><a:p/></p:txBody></p:sp>`,
		id, esc(v.Preset), emu(v.X), emu(v.Y), emu(v.W), emu(v.H), esc(v.Preset), fl, ln)
}

func ovalXML(v ir.Oval, id int) string {
	ln := "<a:ln><a:noFill/></a:ln>"
	if v.Line != "" {
		ln = fmt.Sprintf(`<a:ln w="%d"><a:solidFill><a:srgbClr val="%s"/></a:solidFill>%s</a:ln>`,
			int(v.LineWpt*12700), v.Line, dashXML(v.Dashed))
	}
	return fmt.Sprintf(`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="oval"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>
<a:prstGeom prst="ellipse"><a:avLst/></a:prstGeom>
%s%s</p:spPr>
<p:txBody><a:bodyPr/><a:p/></p:txBody></p:sp>`,
		id, emu(v.X), emu(v.Y), emu(v.W), emu(v.H), fillXML(v.Fill, v.FillAlpha), ln)
}

// runAttrs renders the optional rPr attributes shared by text boxes and
// table cells; empty strings for defaults keep pre-feature output
// byte-identical.
func runAttrs(r ir.Run) (italic, underline, strike string) {
	if r.Italic {
		italic = ` i="1"`
	}
	if r.Underline {
		underline = ` u="sng"`
	}
	if r.Strike {
		strike = ` strike="sngStrike"`
	}
	return
}

// bulletXML renders the pPr attribute prefix and child elements that turn
// a paragraph into a list item at the given nesting level.
func bulletXML(p ir.Para) (attrs, children string) {
	if p.Bullet == "" {
		return "", ""
	}
	attrs = fmt.Sprintf(` marL="%d" indent="-342900"`, 342900*(p.Level+1))
	switch p.Bullet {
	case "number":
		children = `<a:buFont typeface="Arial"/><a:buAutoNum type="arabicPeriod"/>`
	default: // disc
		children = `<a:buFont typeface="Arial"/><a:buChar char="&#8226;"/>`
	}
	return
}

func linkXML(r ir.Run, rels *slideRels) string {
	if r.Link == "" || rels == nil {
		return ""
	}
	return fmt.Sprintf(`<a:hlinkClick xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:id="%s"/>`, rels.linkID(r.Link))
}

func textXML(v ir.Text, id int, rels *slideRels) string {
	var body strings.Builder
	for _, p := range v.Paras {
		algn := ""
		if p.Align != "" {
			algn = fmt.Sprintf(` algn="%s"`, p.Align)
		}
		spc := ""
		if p.Spacing > 0 {
			spc = fmt.Sprintf(`<a:lnSpc><a:spcPct val="%d"/></a:lnSpc>`, int(p.Spacing*100000))
		}
		bAttrs, bChildren := bulletXML(p)
		body.WriteString(fmt.Sprintf(`<a:p><a:pPr%s%s>%s%s</a:pPr>`, bAttrs, algn, spc, bChildren))
		if len(p.Runs) == 0 {
			// size the blank spacer line like the surrounding text; without an
			// explicit sz it inherits the 18pt default and inflates the block
			body.WriteString(fmt.Sprintf(`<a:endParaRPr lang="en-US" sz="%d"/>`, int(v.DefSize*100)))
		}
		for _, r := range p.Runs {
			size := v.DefSize
			if r.Size > 0 {
				size = r.Size
			}
			color := v.DefColor
			if r.Color != "" {
				color = r.Color
			}
			font := v.DefFont
			if r.Font != "" {
				font = r.Font
			}
			b := "0"
			if r.Bold {
				b = "1"
			}
			i, u, st := runAttrs(r)
			body.WriteString(fmt.Sprintf(
				`<a:r><a:rPr lang="en-US" sz="%d" b="%s"%s%s%s dirty="0"><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:latin typeface="%s"/><a:cs typeface="%s"/>%s</a:rPr><a:t>%s</a:t></a:r>`,
				int(size*100), b, i, u, st, color, esc(font), esc(font), linkXML(r, rels), esc(r.Text)))
		}
		body.WriteString("</a:p>")
	}
	rot := ""
	if v.Rot != 0 {
		rot = fmt.Sprintf(` rot="%d"`, int(v.Rot*60000))
	}
	tx, tw := v.X, v.W
	if v.WrapSlack {
		// Chrome exposes the painted glyph width, while Office/Slides needs a
		// few extra pixels to keep a one-line label whole. Preserve alignment
		// while adding that breathing room: centered labels expand equally and
		// right-aligned labels expand to the left.
		extra := tw * 0.12
		switch textAlign(v.Paras) {
		case "ctr":
			tx -= extra / 2
		case "r":
			tx -= extra
		}
		tw += extra
	}
	return fmt.Sprintf(`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="text"/><p:cNvSpPr txBox="1"/><p:nvPr/></p:nvSpPr>
<p:spPr><a:xfrm%s><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>
<a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/></p:spPr>
<p:txBody><a:bodyPr wrap="square" lIns="0" tIns="0" rIns="0" bIns="0"><a:normAutofit/></a:bodyPr>%s</p:txBody></p:sp>`,
		id, rot, emu(tx), emu(v.Y), emu(tw), emu(v.H), body.String())
}

func textAlign(paras []ir.Para) string {
	if len(paras) == 0 || paras[0].Align == "" {
		return "l"
	}
	return paras[0].Align
}

func lineXML(v ir.Line, id int) string {
	// xfrm requires positive extents with flips
	fx, fy := "", ""
	ox, oy := v.X1, v.Y1
	w, h := v.X2-v.X1, v.Y2-v.Y1
	if w < 0 {
		ox, w, fx = v.X2, -w, ` flipH="1"`
	}
	if h < 0 {
		oy, h, fy = v.Y2, -h, ` flipV="1"`
	}
	dash := dashXML(v.Dashed)
	tail := ""
	if v.ArrowStart {
		tail = `<a:headEnd type="triangle" w="med" len="med"/>`
	}
	if v.Arrow {
		tail += `<a:tailEnd type="triangle" w="med" len="med"/>`
	}
	return fmt.Sprintf(`<p:cxnSp><p:nvCxnSpPr><p:cNvPr id="%d" name="line"/><p:cNvCxnSpPr/><p:nvPr/></p:nvCxnSpPr>
<p:spPr><a:xfrm%s%s><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>
<a:prstGeom prst="line"><a:avLst/></a:prstGeom>
<a:ln w="%d"><a:solidFill><a:srgbClr val="%s"/></a:solidFill>%s%s</a:ln></p:spPr></p:cxnSp>`,
		id, fx, fy, emu(ox), emu(oy), emu(w), emu(h), int(v.Wpt*12700), v.Color, dash, tail)
}

// elbowXML draws ir.Elbow as a native bentConnector2. The preset's
// unflipped path runs horizontal-first from the box's top-left to its
// bottom-right; ir.Elbow is vertical-first from (X1,Y1), which equals
// horizontal-first from (X2,Y2) — so draw from the swapped endpoints and
// put the arrows on the opposite connector ends (headEnd = path start =
// the elbow's (X2,Y2)).
func elbowXML(v ir.Elbow, id int) string {
	sx, sy := v.X2, v.Y2 // path start (horizontal-first)
	ex, ey := v.X1, v.Y1 // path end
	fx, fy := "", ""
	ox, oy := sx, sy
	w, h := ex-sx, ey-sy
	if w < 0 {
		ox, w, fx = ex, -w, ` flipH="1"`
	}
	if h < 0 {
		oy, h, fy = ey, -h, ` flipV="1"`
	}
	we, he := emu(w), emu(h)
	if we == 0 {
		we = 1
	}
	if he == 0 {
		he = 1
	}
	dash := dashXML(v.Dashed)
	heads := ""
	if v.Arrow { // head at (X2,Y2) = path start
		heads += `<a:headEnd type="triangle" w="med" len="med"/>`
	}
	if v.ArrowStart { // head at (X1,Y1) = path end
		heads += `<a:tailEnd type="triangle" w="med" len="med"/>`
	}
	return fmt.Sprintf(`<p:cxnSp><p:nvCxnSpPr><p:cNvPr id="%d" name="elbow"/><p:cNvCxnSpPr/><p:nvPr/></p:nvCxnSpPr>
<p:spPr><a:xfrm%s%s><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm>
<a:prstGeom prst="bentConnector2"><a:avLst/></a:prstGeom>
<a:ln w="%d"><a:solidFill><a:srgbClr val="%s"/></a:solidFill>%s%s</a:ln></p:spPr></p:cxnSp>`,
		id, fx, fy, emu(ox), emu(oy), we, he, int(v.Wpt*12700), v.Color, dash, heads)
}

func tableXML(v ir.Table, id int, rels *slideRels) string {
	var sb strings.Builder
	totW := 0.0
	for _, w := range v.ColW {
		totW += w
	}
	sb.WriteString(fmt.Sprintf(`<p:graphicFrame><p:nvGraphicFramePr><p:cNvPr id="%d" name="table"/><p:cNvGraphicFramePr/><p:nvPr/></p:nvGraphicFramePr>
<p:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></p:xfrm>
<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table"><a:tbl>
<a:tblPr firstRow="0" bandRow="0"/><a:tblGrid>`, id, emu(v.X), emu(v.Y), emu(totW), emu(v.RowH*float64(len(v.Rows)))))
	for _, w := range v.ColW {
		sb.WriteString(fmt.Sprintf(`<a:gridCol w="%d"/>`, emu(w)))
	}
	sb.WriteString("</a:tblGrid>")
	for _, row := range v.Rows {
		sb.WriteString(fmt.Sprintf(`<a:tr h="%d">`, emu(v.RowH)))
		for _, c := range row {
			sb.WriteString("<a:tc><a:txBody><a:bodyPr/>")
			if len(c.Paras) == 0 {
				sb.WriteString("<a:p/>")
			}
			for _, p := range c.Paras {
				if p.Align != "" {
					sb.WriteString(fmt.Sprintf(`<a:p><a:pPr algn="%s"/>`, p.Align))
				} else {
					sb.WriteString("<a:p>")
				}
				for _, r := range p.Runs {
					size := v.DefSize
					if r.Size > 0 {
						size = r.Size
					}
					b := "0"
					if r.Bold {
						b = "1"
					}
					i, u, st := runAttrs(r)
					sb.WriteString(fmt.Sprintf(
						`<a:r><a:rPr lang="en-US" sz="%d" b="%s"%s%s%s><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:latin typeface="%s"/>%s</a:rPr><a:t>%s</a:t></a:r>`,
						int(size*100), b, i, u, st, r.Color, r.Font, linkXML(r, rels), esc(r.Text)))
				}
				sb.WriteString("</a:p>")
			}
			fill := `<a:noFill/>`
			if c.Fill != "" {
				fill = fillXML(c.Fill, 0)
			}
			sb.WriteString(fmt.Sprintf(`</a:txBody><a:tcPr marL="91440" marR="91440" marT="45720" marB="45720"><a:lnB w="6350"><a:solidFill><a:srgbClr val="%s"/></a:solidFill></a:lnB>%s</a:tcPr></a:tc>`, v.BorderColor, fill))
		}
		sb.WriteString("</a:tr>")
	}
	sb.WriteString("</a:tbl></a:graphicData></a:graphic></p:graphicFrame>")
	return sb.String()
}
