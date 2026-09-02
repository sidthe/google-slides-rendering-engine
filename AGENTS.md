# Generating decks with deckgen (guide for AI harnesses)

You are producing a presentation. You will write an **HTML file** (one
`<section>` per slide), check and visually review it, then convert it with
the `deckgen` CLI into a fully native, editable deck — a `.pptx` file or a
Google Slides presentation. Nothing is screenshotted: every box becomes a
real shape, every text a real text box, every `<table>` a real table.

## Workflow

```sh
deckgen check      deck.html                  # 1. convert dry-run: fix every warning
deckgen screenshot deck.html -o shots/        # 2. look at the PNGs; iterate until the design is right
deckgen build      deck.html -o deck.pptx     # 3a. PPTX output
deckgen push       deck.html -title "My deck" # 3b. Google Slides — prints the presentation URL
```

`deckgen requests deck.html` prints the Slides API batchUpdate JSON if you
need to inspect what will be created.

Always run `check` (fix warnings) and `screenshot` (look at your own
output) before `build`/`push`. The screenshots are exactly what Chrome
rendered; the deck output mirrors it with the fidelity notes below.

## The contract

- Each slide is `<section>` of **exactly 1600×900 px**. 1 px = 1 grid unit
  in the deck. Start from the starter kit:

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<link rel="stylesheet" href="assets/deck.css">
</head>
<body>
<section>
  <div class="kicker">section label</div>
  <div class="title">Slide title</div>
  <!-- content: y = 180..800 -->
  <div class="foot"><span>footer text</span><span>1</span></div>
</section>
</body>
</html>
```

- Any CSS layout works — absolute positioning, flexbox, grid. Chrome
  computes the geometry; deckgen rebuilds it natively.
- Use real text and styled `div`s only. **No `<img>`, `<svg>`, `<canvas>`,
  `<video>`** — media is skipped with a warning because the output must
  stay natively editable.

## What HTML becomes

| HTML | Deck output |
|---|---|
| `div` with `background` and/or `border` | native rectangle (border-radius preserved as corner rounding) |
| `rgba(...)` backgrounds | native semi-transparent fill (alpha preserved) |
| `border-radius: 50%` on a square-ish div | native ellipse |
| `data-shape="..."` div | native preset shape — see "Diagram shapes" below |
| text (any block element) | native text box; per-`<span>`/`<b>`/`<i>`/`<u>`/`<s>` bold, italics, underline, strikethrough, color, size, font |
| `<a href>` | native hyperlink on the run |
| `<ul>` / `<ol>` (nesting supported) | native bulleted / numbered list, one text box per list |
| `<br>` | paragraph break within one text box |
| `<table>` | native table with per-cell fills, alignment and hairline row borders |
| `<hr>`, or any div ≤3px thick with a background, or `data-line` | native line — see "Lines and arrows" below |
| `data-elbow` div with two borders | native L-shaped elbow connector |
| pure `rotate()` on a text-only div | native rotated text box (axis titles) |
| `<aside class="notes">` (hidden) | speaker notes on the slide |

Supported CSS per element: `background-color` (incl. rgba alpha),
`border` (solid/dashed), `border-radius`, `color`, `font-family`,
`font-size`, `font-weight`, `font-style`, `text-decoration`, `text-align`,
`line-height`, `transform: rotate()` on text/lines, plus anything that
only affects layout (position, flex, grid, padding, margin).

**Approximated or dropped (warned):** gradients and `background-image`
(solid fill), `box-shadow` (dropped), `transform` on boxes (bounding box
only — rotation is preserved only on `data-line` and text-only elements),
`text-shadow`.

## Diagram shapes (`data-shape`)

`<div data-shape="diamond" style="...">` becomes native preset geometry in
both PPTX and Google Slides. Values: `diamond` (decision), `cylinder`/`db`
(database), `hexagon`, `parallelogram` (data), `cloud`, `trapezoid`,
`chevron` and `pentagon` (process/roadmap stages), `triangle` (pyramids),
`callout` (quotes). Fill/border/dash come from CSS as usual; put labels in
a separate positioned div (or flex-center text inside — it becomes its own
text box on top). `deck.css` ships preview `clip-path`s so the browser
matches; angled edges clip borders in the preview only.

Build funnels/pyramids from stacked `trapezoid` + `triangle`; build
process flows from a `pentagon` followed by `chevron`s (see
`examples/features.html` slide 2).

## Elbow connectors (`data-elbow`)

An L-shaped connector is a div that draws the L with exactly two borders —
one vertical side (`border-left` or `border-right`) and one horizontal
side (`border-top` or `border-bottom`):

```html
<!-- down-then-right: from box A's bottom edge into box B's left edge -->
<div data-elbow data-arrow style="left:580px; top:460px; width:355px; height:165px;
     border-left:2px solid; border-bottom:2px solid; color:var(--ink-2);"></div>
```

The path runs from the free tip of the vertical border, through the
corner, to the free tip of the horizontal border. `data-arrow` puts the
head at the horizontal end, `data-arrow="start"` at the vertical end,
`"both"` at both; `data-dashed` dashes it. For the preview arrowhead
position set `--arrow-y:0%` (border-top) or `--arrow-y:100%`
(border-bottom). PPTX gets a real bent connector; the Slides API renders
it as two joined lines (its bent connector can't form an L).

## Bulleted lists

Use real `<ul>`/`<ol>`; nest lists inside `<li>` for sub-bullets. Keep
`<li>` content inline (text, `<b>`, `<a>`, …). Each list becomes ONE
native text box with real bullet/number formatting at the right indent
levels — never fake bullets with `•` characters.

## Speaker notes

Every content slide should carry talking points:

```html
<aside class="notes">
  <p>Open with the incident story.</p>
  <p>Mention the 94% cache-hit rate.</p>
</aside>
```

Hidden in the preview; exported as real speaker notes in both PPTX and
Google Slides (each `<p>` is a notes paragraph).

## Lines and arrows (diagrams)

Any thin div with a background is a line; `data-line` marks one
explicitly (required for rotated/diagonal lines):

- `data-arrow` — arrowhead at the right/bottom end
- `data-arrow="start"` — arrowhead at the left/top end (reply messages)
- `data-arrow="both"` — both ends
- `data-dashed` (or `border-style: dashed`) — dashed line
- **Diagonals**: `<div data-line style="left:X;top:Y;width:LEN;height:2px;
  transform:rotate(Adeg);transform-origin:0 50%;background:...">` — the
  converter computes the true rotated endpoints and draws a native
  straight connector. LEN = √(dx²+dy²), A = atan2(dy,dx) in degrees.
- `border: 2px dashed …` on a box → native dashed outline (group
  boundaries in architecture diagrams).
- Arrowheads in the browser preview come from `deck.css`
  (`[data-arrow]::after`); set `color` on the line div and use
  `background: currentColor` so the preview head matches the line color.
  The converter draws real arrowheads regardless.

## Diagrams

- **Sequence diagram** (`examples/diagrams.html` slide 1): participant
  boxes in a row; vertical dashed `data-line` lifelines under their
  centers; thin activation rects on the lifelines; horizontal `data-arrow`
  calls and `data-arrow="start" data-dashed` returns; label divs above
  each message; a `.note` box for commentary.
- **Architecture diagram** (slide 2): `.box` nodes, a `border: 2px dashed`
  `.group` boundary with an uppercase label, straight and rotated
  `data-line data-arrow` edges, `data-arrow="both"` for bidirectional
  links, small white-backed labels sitting on edges.

## Charts

Draw charts from shapes — they stay editable in the final deck. Never
embed chart images.

- **Bar / grouped bars**: positioned divs on hairline gridlines, value
  labels directly above bars (`examples/demo.html` slide 2).
- **Stacked bars**: stack two divs per column, total label on top
  (`examples/charts.html` slide 1).
- **Line chart**: rotated `data-line` segments dot-to-dot + 12px
  `border-radius:50%` marker divs + value labels (`examples/charts.html`
  slide 1).
- **Scatter**: circle divs positioned on the grid.
- **Pie/donut: not supported natively** (the Slides API cannot set wedge
  angles). Use a bar chart, stacked bar, or a big-number tile with a
  supporting bar instead.

## Tables

`<table>` becomes a native table. Style with `deck.css` defaults; use
`text-align: right` on numeric `td`/`th` (it converts to real cell
alignment), `<b>`/colored spans inside cells for emphasis
(`examples/charts.html` slide 2). Keep cell content short — rows grow if
text overflows (Slides has minimum row heights only).

## Design rules (follow these for a professional deck)

1. **Stay on the palette.** Use the CSS variables from `assets/deck.css`
   (`--ink`, `--ink-2`, `--ink-3`, `--line`, `--surface`, `--blue`,
   `--blue-dark`, `--blue-tint`, and the red/yellow/green trios). Tinted
   backgrounds pair with their dark text: `--green-tint` + `--green-dark`,
   `--blue-tint` + `--blue-dark`, etc.
2. **Two fonts maximum**: `var(--font)` (Roboto) for everything,
   `var(--mono)` (Roboto Mono) for code. Both exist in Google Slides — do
   not introduce fonts Slides lacks (no Google Sans, no Helvetica Neue;
   they'd silently fall back).
3. **Scaffold coordinates** (used by the starter classes): kicker at
   y=52, title at y=82, content between y=180 and y=800, footer at y=862.
   Margins: x=80 left, content ends at x=1520.
4. **Type scale** (px): title 52, big hero numbers 66–90, card titles 23,
   body 20–23, labels 17, footers/ticks 16. Nothing under 15px.
5. **Density**: one idea per slide, ≤3 stat tiles in a row, ≤5 content
   blocks per slide. Let whitespace work; do not fill every pixel.
6. **Charts are drawn, not embedded**: build bar charts from absolutely
   positioned divs on a hairline grid (see slide 2 of
   `examples/demo.html`), label values directly above bars. That keeps
   charts editable in the final deck.
7. Round numbers, short labels, sentence case everywhere except the
   uppercase kicker.

## Fidelity notes (Google Slides)

The Slides page is 10in wide (PPTX is 13.33in); everything is scaled
proportionally, so point sizes in the deck are 0.75× the PPTX values —
visually identical.

Slides API limits deckgen designs around; don't fight them:

- Corner radius is Slides' default rounding (the API can't set radii).
  Fully-round pills render slightly less round than in HTML.
- Table rows have *minimum* heights: overflowing cell text grows the row.
  Size table content so it fits.
- Table cell inner margins are fixed defaults; text boxes are
  inset-compensated automatically.
- Multi-line text may re-wrap slightly differently than Chrome. Leave
  10–15% slack in text box widths; don't rely on exact line breaks.

## Recipes

- **Title slide**: accent dots at (80,96), 90px title around y=292,
  30px `--ink-2` subtitle beneath, one pill, footer.
- **Stats + chart**: three 340×150 tiles at y=180 (x = 80, 440, 800), a
  full-width card y=370..800 containing a drawn bar chart.
- **Comparison**: two 700×540 cards side by side (x=80, x=820) with a
  7px accent strip div across each card's top, bulleted lines as separate
  positioned divs.
- **Table + code**: table at y=180 (th row styled by `deck.css`), dark
  `.code` block below with `<br>`-separated lines.

`examples/demo.html` implements the first, second and fourth;
`examples/diagrams.html` covers sequence + architecture diagrams;
`examples/charts.html` covers line/stacked-bar charts and numeric tables;
`examples/features.html` covers flowcharts (diamond/cylinder/elbows), GTM
shapes (chevron process, pyramid, callout), lists, links, emphasis, alpha
overlays, rotated labels and speaker notes. Copy their structure.

## Templates

This file is the mechanical contract. `templates/` adds opinionated
voice + design systems on top of it. If the user asks for a template by
name (e.g. "use the keynote template", "speaker style"), read that
template's `VOICE.md` and `DESIGN.md` before authoring, copy archetypes
from its `skeleton.html`, and validate with
`deckgen check deck.html -lint <template>`. Template lint findings are
advisory floors (type size, word and element budgets, notes present) —
fix them or justify the exception.
