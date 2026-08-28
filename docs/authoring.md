# Authoring native decks with deckgen

This is the contract for creating a presentation with deckgen. Write an HTML
file with one `<section>` per slide, validate it, visually review it, then
convert it to a fully native, editable deck. Nothing is screenshotted: boxes
become shapes, text becomes text boxes, and tables remain tables.

## Required workflow

```sh
deckgen check      deck.html                  # dry run; fix every warning
deckgen screenshot deck.html -o shots/        # visually inspect each slide
deckgen build      deck.html -o deck.pptx     # native PPTX
deckgen push       deck.html -title "My deck" # native Google Slides
```

Use `deckgen requests deck.html` to inspect the Slides API request plan.
Always run `check` and `screenshot` before `build` or `push`.

## Slide contract

- Every slide is a `<section>` of exactly **1600 × 900 px**. One pixel is one
  deck grid unit.
- Use the starter structure below. Keep content between y=180 and y=800;
  the title, kicker and footer positions are defined by `assets/deck.css`.
- Use real text and styled `div`s. Do not use `<img>`, `<svg>`, `<canvas>` or
  `<video>`: media is omitted so the output stays natively editable.

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

Chrome computes the layout, so absolute positioning, flexbox and grid all
work. deckgen rebuilds the resulting geometry as native deck objects.

## Conversion rules

| HTML | Native output |
|---|---|
| `div` with a background or border | rectangle; border radius is preserved as rounding |
| `rgba(...)` background | semi-transparent fill |
| square `div` with `border-radius: 50%` | ellipse |
| `data-shape="..."` | native preset shape |
| text in block elements | text box with span-level formatting |
| `<a href>` | hyperlink on the text run |
| `<ul>` / `<ol>` | native bulleted or numbered list |
| `<br>` | paragraph break |
| `<table>` | native table |
| thin colored `div`, `<hr>`, or `data-line` | native line |
| `data-elbow` | L-shaped connector |
| text-only `rotate()` | rotated text box |
| `<aside class="notes">` | speaker notes |

Supported styling includes fills (including alpha), solid/dashed borders,
border radius, font family/size/weight/style, text decoration/alignment/line
height, and rotation for text or lines. Gradients, background images, shadows,
and box transforms are approximated or dropped with a warning.

## Shapes, connectors and notes

Set `data-shape` on a `div` for `diamond`, `cylinder` (or `db`), `hexagon`,
`parallelogram`, `cloud`, `trapezoid`, `chevron`, `pentagon`, `triangle`, or
`callout`. Put labels in a positioned text element over the shape.

Create an L-shaped connector with exactly one vertical and one horizontal
border. Add `data-arrow`, `data-arrow="start"`, or `data-arrow="both"` for
arrowheads, and `data-dashed` for a dashed connector.

```html
<div data-elbow data-arrow style="left:580px; top:460px; width:355px; height:165px;
     border-left:2px solid; border-bottom:2px solid; color:var(--ink-2);"></div>
```

Use a thin `data-line` element for direct connectors. A rotated line becomes a
native straight connector; set `background: currentColor` and `color` so its
preview arrowhead matches. For a diagonal, set `width` to √(dx²+dy²) and
`transform: rotate(atan2(dy, dx))`.

Use real list elements rather than typing bullet characters. Add notes to
every content slide when there are presentation talking points:

```html
<aside class="notes">
  <p>Open with the incident story.</p>
  <p>Mention the 94% cache-hit rate.</p>
</aside>
```

## Charts and tables

Draw charts from editable shapes; do not embed chart images. Build bar charts
from positioned `div`s over hairline gridlines. Build line charts from rotated
`data-line` segments and circular marker `div`s. Use bars, stacked bars, or
number tiles instead of pie and donut charts.

Tables convert natively. Keep cell text short, right-align numeric cells, and
leave enough vertical space because Slides can grow a row when text overflows.

## Visual design rules

1. Use the palette variables from `assets/deck.css`: `--ink`, `--ink-2`,
   `--ink-3`, `--line`, `--surface`, `--blue`, `--blue-dark`, `--blue-tint`,
   and the red/yellow/green trios.
2. Use at most two fonts: `var(--font)` (Roboto) and `var(--mono)` (Roboto
   Mono). Both are available in Google Slides.
3. Use x=80 as the left margin and x=1520 as the right content edge. Place
   the kicker at y=52, title at y=82, and footer at y=862.
4. Use 52px titles, 66–90px hero numbers, 23px card titles, 20–23px body,
   17px labels, and 16px footer/tick text. Do not use text below 15px.
5. Keep one idea per slide: at most three stat tiles in a row and five content
   blocks. Prefer whitespace to density.
6. Use sentence case except for uppercase kickers. Keep labels short and
   numbers rounded.

## Fidelity boundaries

Google Slides pages are 10in wide while conventional widescreen PPTX pages
are 13.33in. deckgen scales the Google Slides target proportionally, so the
visual result is the same.

Slides API limits affect a few details:

- Corner radius uses the service's default rounding.
- Text boxes use fixed insets; deckgen compensates for them.
- Table rows have minimum heights and can grow with overflow.
- Table cell margins are fixed.
- Multi-line text can wrap slightly differently from Chrome; leave 10–15%
  width slack and avoid relying on an exact wrap point.

## Reference examples

- `examples/demo.html`: title, statistics, bar chart, and code/table layouts.
- `examples/diagrams.html`: sequence and architecture diagrams.
- `examples/charts.html`: line, stacked-bar, and numeric-table charts.
- `examples/features.html`: preset shapes, connectors, lists, links, alpha
  overlays, rotated labels, and speaker notes.
