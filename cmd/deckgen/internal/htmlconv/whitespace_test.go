package htmlconv

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The extractor mirrors what Chrome paints, so these cases are pinned
// against a real Chrome render. Each case names the markup, then the
// paragraphs it must produce. See docs/specs/t2-whitespace-fidelity.md.
type wsCase struct {
	name string
	html string
	want []string // one entry per paragraph, runs concatenated
}

var wsCases = []wsCase{
	{
		"space between inline elements",
		`<div style="left:10px;top:10px">a<span> </span>b</div>`,
		[]string{"a b"},
	},
	{
		"space between two styled runs",
		`<div style="left:10px;top:10px"><b>Bold</b> <i>Ital</i></div>`,
		[]string{"Bold Ital"},
	},
	{
		"runs of spaces collapse across run boundaries",
		`<div style="left:10px;top:10px">one <b>two</b>  <b>three</b> four</div>`,
		[]string{"one two three four"},
	},
	{
		"leading and trailing white space is dropped",
		"<div style=\"left:10px;top:10px\">\n   spaced   \n</div>",
		[]string{"spaced"},
	},
	{
		"non-breaking spaces survive",
		`<div style="left:10px;top:10px">Price:&nbsp;&nbsp;&nbsp;42</div>`,
		[]string{"Price:   42"},
	},
	{
		"pre keeps line breaks and indentation",
		"<div style=\"left:10px;top:10px;white-space:pre\">def f(x):\n    return x + 1</div>",
		[]string{"def f(x):", "    return x + 1"},
	},
	{
		"pre keeps blank lines",
		"<div style=\"left:10px;top:10px;white-space:pre\">a\n\nb</div>",
		[]string{"a", "", "b"},
	},
	{
		"pre drops one trailing segment break",
		"<div style=\"left:10px;top:10px;white-space:pre\">a\nb\n</div>",
		[]string{"a", "b"},
	},
	{
		"pre keeps indentation inside styled spans",
		"<div style=\"left:10px;top:10px;white-space:pre\"><span style=\"color:#f00\">spec</span>:\n  <span style=\"color:#00f\">name</span>: demo</div>",
		[]string{"spec:", "  name: demo"},
	},
	{
		"pre-line keeps breaks but collapses spaces",
		"<div style=\"left:10px;top:10px;white-space:pre-line\">  Line one     with    spaces  \n  Line two  </div>",
		[]string{"Line one with spaces", "Line two"},
	},
	{
		"monospace alone does not preserve white space",
		"<div style=\"left:10px;top:10px;font-family:monospace\">\n    A mono caption that\n    wraps as one line.\n  </div>",
		[]string{"A mono caption that wraps as one line."},
	},
	{
		"br still starts a paragraph",
		`<div style="left:10px;top:10px">first<br>second</div>`,
		[]string{"first", "second"},
	},
}

func TestExtractWhiteSpaceMatchesChrome(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><style>` +
		`section{position:relative;width:1600px;height:900px}` +
		`section>div{position:absolute}</style></head><body>`)
	for _, c := range wsCases {
		b.WriteString("<section>" + c.html + "</section>")
	}
	b.WriteString("</body></html>")

	path := filepath.Join(t.TempDir(), "ws.html")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Extract(context.Background(), path)
	if err != nil {
		t.Skipf("chrome unavailable: %v", err)
	}
	if len(res.Scenes) != len(wsCases) {
		t.Fatalf("scenes: got %d want %d", len(res.Scenes), len(wsCases))
	}
	for i, c := range wsCases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			for _, el := range res.Scenes[i].Elements {
				for _, p := range el.Paras {
					var s strings.Builder
					for _, r := range p.Runs {
						s.WriteString(r.Text)
					}
					got = append(got, s.String())
				}
			}
			if len(got) != len(c.want) {
				t.Fatalf("paragraphs: got %q want %q", got, c.want)
			}
			for j := range got {
				if got[j] != c.want[j] {
					t.Errorf("para %d: got %q want %q", j, got[j], c.want[j])
				}
			}
		})
	}
}

// TestExtractCodeClassKeepsIndentation covers the authoring path: a .code
// block styled by assets/deck.css must reach the deck with its hierarchical
// indentation and nothing added in front of it. The <pre> form gets
// white-space from the UA stylesheet; the <div> form only works because
// deck.css sets it, so both are pinned.
func TestExtractCodeClassKeepsIndentation(t *testing.T) {
	css, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "assets", "deck.css"))
	if err != nil {
		t.Fatal(err)
	}
	// The HTML parser drops one newline after a <pre> start tag, so that form
	// can be written on its own line. A <div> keeps it, which Chrome paints as
	// a blank first line, so the div form starts flush against the tag.
	body := `<pre class="code" style="left:80px;top:100px;width:700px;height:300px">` + `
<span style="color:#569CD6">metadata</span>:
  <span style="color:#9CDCFE">name</span>: demo
  <span style="color:#9CDCFE">labels</span>:
    <span style="color:#CE9178">app</span>: web
</pre>` + `<div class="code" style="left:820px;top:100px;width:700px;height:300px"><span style="color:#569CD6">metadata</span>:
  <span style="color:#9CDCFE">name</span>: demo
  <span style="color:#9CDCFE">labels</span>:
    <span style="color:#CE9178">app</span>: web</div>`

	html := `<!DOCTYPE html><html><head><meta charset="utf-8">` +
		`<link rel="stylesheet" href="file://` + css + `"></head><body><section>` +
		body + `</section></body></html>`

	path := filepath.Join(t.TempDir(), "code.html")
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Extract(context.Background(), path)
	if err != nil {
		t.Skipf("chrome unavailable: %v", err)
	}
	block := []string{"metadata:", "  name: demo", "  labels:", "    app: web"}
	want := append(append([]string{}, block...), block...)
	var got []string
	for _, s := range res.Scenes {
		for _, el := range s.Elements {
			for _, p := range el.Paras {
				var b strings.Builder
				for _, r := range p.Runs {
					b.WriteString(r.Text)
				}
				got = append(got, b.String())
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("paragraphs: got %q want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("para %d: got %q want %q", i, got[i], want[i])
		}
	}
}
