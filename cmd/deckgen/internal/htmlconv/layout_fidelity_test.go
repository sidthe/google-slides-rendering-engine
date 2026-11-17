package htmlconv

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deck-engine/deck-engine/ir"
)

func TestCSSSpacingNormalizesEmToSingleLine(t *testing.T) {
	cases := []struct {
		css  float64
		want float64
	}{
		{0, 0},
		{1.2, 1.0},
		{1.35, 1.125},
		{1.4, 1.1667},
		{1.55, 1.2917},
		{1.05, 0.875},
	}
	for _, tc := range cases {
		got := cssSpacing(tc.css)
		if math.Abs(got-tc.want) > 1e-4 {
			t.Errorf("cssSpacing(%v) = %v, want %v", tc.css, got, tc.want)
		}
	}
}

func TestNeedsWrapSlackKeepsMultiParaSingleLines(t *testing.T) {
	hardBroken := Element{
		Kind: "text", W: 680, H: 112, FontSize: 48, Spacing: 1.15,
		Paras: []Para{
			{Runs: []Run{{Text: "First line"}}},
			{Runs: []Run{{Text: "Second line after br"}}},
		},
	}
	if got := needsWrapSlack(hardBroken); !got {
		t.Errorf("needsWrapSlack(hardBroken) = %v, want true", got)
	}

	containerBacked := hardBroken
	containerBacked.W = 1380
	containerBacked.NoSlack = true
	if got := needsWrapSlack(containerBacked); got {
		t.Errorf("needsWrapSlack(containerBacked) = %v, want false", got)
	}

	softWrapped := Element{
		Kind: "text", W: 366, H: 52, FontSize: 18, Spacing: 1.4,
		Paras: []Para{{Runs: []Run{{Text: "Single paragraph wrapped across two visual lines"}}}},
	}
	if got := needsWrapSlack(softWrapped); got {
		t.Errorf("needsWrapSlack(softWrapped) = %v, want false", got)
	}

	// 3 hard-broken paragraphs where one soft-wrapped onto a 4th visual line:
	// 4 * 24 = 96 > (3 + 0.45) * 24 = 82.8 -> must NOT gain WrapSlack.
	threeParasOneWrapped := Element{
		Kind: "text", W: 400, H: 96, FontSize: 20, Spacing: 1.2,
		Paras: []Para{
			{Runs: []Run{{Text: "Line 1"}}},
			{Runs: []Run{{Text: "Line 2"}}},
			{Runs: []Run{{Text: "Line 3 wrapped onto line 4"}}},
		},
	}
	if got := needsWrapSlack(threeParasOneWrapped); got {
		t.Errorf("needsWrapSlack(threeParasOneWrapped) = %v, want false", got)
	}
}

func TestExtractPreservesContainerWidthAndFlexCentering(t *testing.T) {
	if os.Getenv("DECKGEN_SKIP_CHROME") != "" {
		t.Skip("DECKGEN_SKIP_CHROME set")
	}
	html := `<!DOCTYPE html><html><head><meta charset="utf-8"><style>
* { margin:0; padding:0; box-sizing:border-box; }
section { position:relative; width:1600px; height:900px; font:20px Roboto, sans-serif; }
.card { position:absolute; left:80px; top:180px; width:454px; height:420px; border:2px solid #4285F4; }
.lead { position:absolute; left:24px; right:24px; top:40px; font-size:18px; line-height:1.4; }
.mono { font-family:"Roboto Mono", monospace; font-size:16px; }
.hero { position:absolute; left:80px; top:40px; width:1380px; font-size:56px; line-height:1.15; }
.pill { position:absolute; left:600px; top:200px; width:240px; height:44px; display:flex; align-items:center; justify-content:center; font-size:15px; font-weight:700; background:#E8F0FE; }
</style></head><body><section>
  <div class="hero">Declarative infrastructure<br>provisioning advancements</div>
  <div class="card">
    <div class="lead">Declarative Kubernetes API (<span class="mono">example.io/v1alpha1</span>) governing cluster scaling:</div>
  </div>
  <div class="pill">STATUS CHIP</div>
</section></body></html>`

	path := filepath.Join(t.TempDir(), "bounds.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Extract(context.Background(), path)
	if err != nil {
		t.Skipf("chrome unavailable: %v", err)
	}

	var hero, lead, pill *Element
	for i := range res.Scenes[0].Elements {
		el := &res.Scenes[0].Elements[i]
		if el.Kind != "text" || len(el.Paras) == 0 || len(el.Paras[0].Runs) == 0 {
			continue
		}
		txt := el.Paras[0].Runs[0].Text
		switch {
		case strings.Contains(txt, "Declarative infrastructure"):
			hero = el
		case strings.Contains(txt, "Declarative Kubernetes"):
			lead = el
		case strings.Contains(txt, "STATUS CHIP"):
			pill = el
		}
	}
	if hero == nil || lead == nil || pill == nil {
		t.Fatalf("missing extracted text elements: hero=%v lead=%v pill=%v", hero, lead, pill)
	}
	if math.Abs(hero.W-1380) > 1 {
		t.Errorf("hero width = %.2f, want 1380 (container content box)", hero.W)
	}
	// Card is 454px wide with 2px border -> 450px inner width; .lead has left:24, right:24 -> 402px.
	if math.Abs(lead.W-402) > 1 {
		t.Errorf("card lead width = %.2f, want 402 (container content box, not ragged glyph width)", lead.W)
	}
	// Flex-centered pill without text-align:center must stay horizontally centered inside [600, 840].
	if pill.X < 640 || pill.W > 180 {
		t.Errorf("flex pill shifted to container edge: x=%.2f w=%.2f", pill.X, pill.W)
	}
}

func TestExtractTextTransformLetterSpacingListsAndTables(t *testing.T) {
	if os.Getenv("DECKGEN_SKIP_CHROME") != "" {
		t.Skip("DECKGEN_SKIP_CHROME set")
	}
	html := `<!DOCTYPE html><html><head><meta charset="utf-8"><style>
* { margin:0; padding:0; box-sizing:border-box; }
section { position:relative; width:1600px; height:900px; font:20px Roboto, sans-serif; }
.up { position:absolute; left:80px; top:40px; text-transform:uppercase; letter-spacing:2px; font-weight:700; }
.low { position:absolute; left:80px; top:80px; text-transform:lowercase; }
.cap { position:absolute; left:80px; top:120px; text-transform:capitalize; letter-spacing:-1.5px; }
ul { position:absolute; left:80px; top:180px; width:420px; padding-left:26px; font-size:20px; }
li { margin:8px 0; line-height:1.35; }
table { position:absolute; left:600px; top:180px; width:500px; border-collapse:collapse; font-size:18px; }
td { padding:10px 16px; border-bottom:1px solid #DADCE0; line-height:1.35; vertical-align:top; }
</style></head><body><section>
  <div class="up">section kicker</div>
  <div class="low">MIXED Case Input</div>
  <div class="cap"><b>cloud</b> native-runtime overview</div>
  <ul>
    <li>First bullet item</li>
    <li>Second item with sub-list
      <ul>
        <li>Nested item</li>
      </ul>
    </li>
    <li>Final bullet item</li>
  </ul>
  <table>
    <tr><td style="width:200px;">Short</td><td style="width:300px;">One line</td></tr>
    <tr><td>Tall row</td><td>This cell wraps across several visual lines inside a three-hundred-pixel column to grow row two well beyond row one</td></tr>
    <tr><td>Blocks</td><td><div>First block</div><div>Second block</div></td></tr>
  </table>
</section></body></html>`

	path := filepath.Join(t.TempDir(), "fidelity.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Extract(context.Background(), path)
	if err != nil {
		t.Skipf("chrome unavailable: %v", err)
	}
	d, warns := MapScenes(res)
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}

	var texts []ir.Text
	var tbl *ir.Table
	for _, sh := range d.Slides[0].Shapes {
		switch v := sh.(type) {
		case ir.Text:
			texts = append(texts, v)
		case ir.Table:
			cp := v
			tbl = &cp
		}
	}
	if len(texts) != 4 || tbl == nil {
		t.Fatalf("got %d text shapes and table=%v, want 4 texts and 1 table", len(texts), tbl != nil)
	}

	// 1. uppercase + +2px (1.2pt) letter-spacing
	if got := texts[0].Paras[0].Runs[0].Text; got != "SECTION KICKER" {
		t.Errorf("uppercase: got %q, want %q", got, "SECTION KICKER")
	}
	if got := texts[0].Paras[0].Runs[0].LetterSpacing; math.Abs(got-1.2) > 0.01 {
		t.Errorf("positive letter-spacing: got %v, want 1.2", got)
	}

	// 2. lowercase
	if got := texts[1].Paras[0].Runs[0].Text; got != "mixed case input" {
		t.Errorf("lowercase: got %q, want %q", got, "mixed case input")
	}

	// 3. capitalize across run boundaries + -1.5px (-0.9pt) letter-spacing
	capText := texts[2].Paras[0].Runs[0].Text + texts[2].Paras[0].Runs[1].Text
	if capText != "Cloud Native-Runtime Overview" {
		t.Errorf("capitalize: got %q, want %q", capText, "Cloud Native-Runtime Overview")
	}
	if got := texts[2].Paras[0].Runs[0].LetterSpacing; math.Abs(got - -0.9) > 0.01 {
		t.Errorf("negative letter-spacing: got %v, want -0.9", got)
	}

	// 4. ul geometry, indent (26px = 15.6pt), inter-item SpaceAfter (8px = 4.8pt), and calibrated Spacing (1.35/1.2 = 1.125)
	list := texts[3]
	if list.Y <= 180 {
		t.Errorf("list Y = %.2f, want > 180 (first li BFC top margin)", list.Y)
	}
	if len(list.Paras) != 4 {
		t.Fatalf("list paras = %d, want 4", len(list.Paras))
	}
	for i, p := range list.Paras {
		if math.Abs(p.BulletIndent-15.6) > 0.05 {
			t.Errorf("para %d BulletIndent = %v, want 15.6", i, p.BulletIndent)
		}
		if math.Abs(p.Spacing-1.125) > 0.01 {
			t.Errorf("para %d Spacing = %v, want 1.125", i, p.Spacing)
		}
		wantAfter := 4.8
		if i == len(list.Paras)-1 {
			wantAfter = 0
		}
		if math.Abs(p.SpaceAfter-wantAfter) > 0.05 {
			t.Errorf("para %d SpaceAfter = %v, want %v", i, p.SpaceAfter, wantAfter)
		}
	}

	// 5. Table per-row heights and nested block cells
	if len(tbl.RowHs) != 3 {
		t.Fatalf("table RowHs = %v, want 3 entries", tbl.RowHs)
	}
	if tbl.RowHs[1] <= tbl.RowHs[0]*1.5 {
		t.Errorf("tall row height %.1f should exceed 1.5x short row %.1f", tbl.RowHs[1], tbl.RowHs[0])
	}
	blockCell := tbl.Rows[2][1].Paras
	if len(blockCell) != 2 || blockCell[0].Runs[0].Text != "First block" || blockCell[1].Runs[0].Text != "Second block" {
		t.Errorf("nested block cell paras = %+v, want [First block, Second block]", blockCell)
	}
}

func TestExtractListEdgeCasesBrBlockPNonUniformIndentAndEmptyLi(t *testing.T) {
	if os.Getenv("DECKGEN_SKIP_CHROME") != "" {
		t.Skip("DECKGEN_SKIP_CHROME set")
	}
	html := `<!DOCTYPE html><html><head><meta charset="utf-8"><style>
* { margin:0; padding:0; box-sizing:border-box; }
section { position:relative; width:1600px; height:900px; font:20px Roboto, sans-serif; }
ol { position:absolute; left:80px; top:80px; width:500px; padding-left:30px; margin-bottom:40px; }
ol ul { padding-left:16px; margin-bottom:14px; }
li { margin:6px 0; line-height:1.35; }
</style></head><body><section>
  <ol>
    <li>First line<br>Second line same item</li>
    <li><p>Paragraph-wrapped item</p>
      <ul>
        <li>Non-uniform nested indent</li>
      </ul>
    </li>
    <li></li>
  </ol>
</section></body></html>`

	path := filepath.Join(t.TempDir(), "list_edges.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Extract(context.Background(), path)
	if err != nil {
		t.Skipf("chrome unavailable: %v", err)
	}
	d, _ := MapScenes(res)
	list := d.Slides[0].Shapes[0].(ir.Text)
	if len(list.Paras) != 4 {
		t.Fatalf("got %d list paras, want 4 (br merged, <p> kept, nested kept, empty kept): %+v", len(list.Paras), list.Paras)
	}
	if got := list.Paras[0].Runs[0].Text; got != "First line\vSecond line same item" {
		t.Errorf("para 0 soft-break merge: got %q", got)
	}
	if got := list.Paras[1].Runs[0].Text; got != "Paragraph-wrapped item" {
		t.Errorf("para 1 <p>-wrapped li: got %q", got)
	}
	// Outer padding-left 30px + inner 16px = 46px (27.6pt); step = 16px (9.6pt).
	// Nested ul margin-bottom: 14px (8.4pt) applies on exit, NOT outer ol margin-bottom: 40px (24pt).
	if math.Abs(list.Paras[2].BulletIndent-9.6) > 0.05 || math.Abs(list.Paras[2].BulletMargin-27.6) > 0.05 {
		t.Errorf("para 2 non-uniform nested indent: step=%v margin=%v, want 9.6 / 27.6", list.Paras[2].BulletIndent, list.Paras[2].BulletMargin)
	}
	if math.Abs(list.Paras[2].SpaceAfter-8.4) > 0.05 {
		t.Errorf("para 2 nested exit SpaceAfter = %v, want 8.4 (nested ul margin-bottom 14px, not outer ol 40px)", list.Paras[2].SpaceAfter)
	}
	if len(list.Paras[3].Runs) != 0 || list.Paras[3].Bullet != "number" {
		t.Errorf("para 3 empty li: got %+v", list.Paras[3])
	}
}
