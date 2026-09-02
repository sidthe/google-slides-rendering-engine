# deck-engine

Turn HTML decks into **fully native, editable presentations** — PPTX files
and Google Slides — with no images, no screenshots, no embedded renders.
Every box becomes a real shape, every text a real text box, every table a
real table.

Built for AI coding assistants: the model writes HTML/CSS, headless Chrome
does the layout, and `deckgen` rebuilds the result natively.

## Read before authoring decks

**Agents must read [`docs/authoring.md`](docs/authoring.md) before creating or
editing a deck.** It is the complete authoring contract: slide geometry,
supported HTML/CSS, native-editability limits, visual design rules, validation
workflow, and fidelity boundaries. Agents using Gemini CLI should also read
[`docs/gemini.md`](docs/gemini.md).

## Templates

The authoring contract defines what renders correctly; a **template**
defines what reads well — a voice guide, composition rules, and a catalog
of slide archetypes with known-good geometry, plus a lint pass that keeps
decks inside the template's floors.

Available templates live under [`templates/`](templates/):

- [`templates/keynote/`](templates/keynote/) — conference-keynote style:
  one idea per slide, huge type, drawn diagrams/mockups/charts, plain
  human copy, detail in speaker notes. Start with its `README.md`; the
  23-slide `skeleton.html` shows every archetype with fictional content.

To author against a template, read its `VOICE.md` and `DESIGN.md`, copy
archetypes from its skeleton, and validate with the template lint:

```sh
deckgen check deck.html -lint keynote
```

```
 deck.html (1600×900 <section> per slide, any CSS)
     │  headless Chrome + injected extractor
     ▼
 scene graph (computed geometry + styles)
     │  pure Go mapper
     ▼
 ir.Deck (typed intermediate representation)
     ├─▶ pptx    → native .pptx (OOXML DrawingML)
     │              └─▶ Drive convert → docs.google.com/… URL   (import)
     └─▶ gslides → Slides API batchUpdate → docs.google.com/… URL (push)
```

## Quick start (CLI)

```sh
cd cmd/deckgen && go build -o deckgen .

./deckgen check      ../../examples/demo.html               # warnings report
./deckgen screenshot ../../examples/demo.html -o shots/     # per-slide PNGs
./deckgen requests   ../../examples/demo.html               # Slides batchUpdate JSON
./deckgen build      ../../examples/demo.html -o demo.pptx  # PPTX tuned for Slides import
./deckgen build      ../../examples/demo.html -target powerpoint -o demo-powerpoint.pptx
./deckgen push       ../../examples/demo.html -title Demo   # Google Slides via the API
./deckgen import     ../../examples/demo.html -title Demo   # Google Slides via PPTX upload
./deckgen snap       <presentationId> -o shots-slides/      # Slides service renders
```

Requirements: Chrome/Chromium on PATH (or `DECKGEN_CHROME=/path/to/chrome`).

`check`, `screenshot`, `requests` and `build` are entirely local. `build`
defaults to the `google-slides` target: a 10in Slides-compatible PPTX canvas
that preserves table and text geometry through the Google Slides import path.
Use `-target powerpoint` for a conventional 13.33in PowerPoint file.

## Two routes to Google Slides

| | `push` | `import` |
|---|---|---|
| How | creates every object through the Slides API | uploads a PPTX and lets Drive convert it |
| Scopes | `presentations` | `presentations` + `drive.file` |
| Corner radius | Slides' default rounding | preserved from the PPTX |
| Text insets | frames compensated for fixed insets | zeroed insets preserved |
| Table rows | `minRowHeight` only | row heights preserved |
| Needs | Slides API enabled | Slides **and** Drive API enabled |

`import` gives higher fidelity because PPTX carries properties the Slides API
cannot set. `push` needs one less API and one less scope, and it is the path
the Go library exposes directly.

Without the Drive API, run `deckgen build`, import the file through the Google
Slides UI, then use `deckgen snap <presentation-id>` to inspect the service's
rendered PNGs.

### Google Slides auth for `push`, `import` and `snap`

Credentials need the `https://www.googleapis.com/auth/presentations` scope,
plus `https://www.googleapis.com/auth/drive.file` for `import`. The
`drive.file` scope only covers files deckgen creates or opens. Two options:

1. **Application Default Credentials** (tried first):

   ```sh
   gcloud auth application-default login \
     --scopes=openid,https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/presentations,https://www.googleapis.com/auth/drive.file
   ```

2. **OAuth client JSON**: create an OAuth client (type: Desktop app) in
   the Cloud Console (with the Slides API enabled, and the Drive API for
   `import`), download the JSON, then pass
   `-oauth-client client_secret.json` (or save it as
   `~/.config/deckgen/client_secret.json`). A browser consent tab opens once;
   the token is cached at `~/.config/deckgen/token.json`. Pass `-reauth` to
   `import` to force a fresh consent when a cached token lacks a scope.

Never commit an OAuth client JSON or token file.

## Quick start (Go library)

The original builder API still works and stays dependency-free:

```go
d := deck.New()
s := d.Slide()
deck.Title(s, "Hello")
deck.Tile(s, 80, 180, 340, 150, "42", "the answer", deck.BLUE50, deck.BLUE7, 40)

d.Save("out.pptx")                                       // PowerPoint canvas
pptx.WriteGoogleSlides("import.pptx", d.IR())            // Slides import canvas
res, _ := gslides.Push(ctx, httpClient, d.IR(), "Hello") // Slides API
```

`gslides.ImportPPTX(ctx, httpClient, pptxBytes, title)` covers the other
route: it uploads the bytes and returns the converted presentation's ID and
URL.

Run the demo: `go run ./example demo.pptx`

## Layers

| Layer | Path | What it gives you |
|---|---|---|
| IR | `ir/` | typed deck model: `Rect`, `Oval`, `AutoShape`, `Text`, `Line`, `Elbow`, `Table` on a 1600×900 grid; the contract between front ends and emitters |
| PPTX emitter | `pptx/` | native OOXML package writer (stdlib only); `Write` for PowerPoint, `WriteGoogleSlides` for the Slides import canvas |
| Slides emitter | `gslides/` | pure `ir.Deck → batchUpdate` request plan, `Push`, `ImportPPTX`, `Thumbnails` (works with any authorized `*http.Client`; no provider SDK) |
| Builder API | root | `Deck`, `Slide`, shape methods, `P/R/B/BC/RC` text helpers |
| Style kit | `style.go` | Go-level house components (`Title`, `Tile`, `Card`, `BarChart`, …) |
| HTML front end | `cmd/deckgen/` | headless-Chrome extraction, scene→IR mapper, CLI, Slides auth |
| Starter CSS | `assets/deck.css` | the house look as CSS classes + palette variables |

The root module has **zero external dependencies**; Chrome, chromedp and
OAuth live only in the nested `cmd/deckgen` module.

## Google Slides fidelity

Slides presentations are 10in wide (PPTX: 13.33in): all geometry and point
sizes scale by exactly 0.75, so decks are proportionally identical. Both
routes apply that scale — `WriteGoogleSlides` bakes it into the PPTX, the
`gslides` emitter applies it per request.

Chrome reports the exact painted glyph width, and native engines wrap the same
run a few pixels earlier. Text frames the HTML front end derives from Chrome
bounds carry `ir.Text.WrapSlack`, and both emitters add 12% trailing width for
those frames without moving the text's visual alignment. Builder-authored
frames stay exact.

Known gaps in the `push` route:

| Gap | Cause | Effect |
|---|---|---|
| corner radius | API can't set geometry adjust values | Slides' default rounding |
| text insets | API can't zero text-box insets | frames auto-compensated by the default insets |
| table row height | `minRowHeight` only | overflowing text grows rows |
| table cell margins | not settable | slight text offset inside cells |
| fonts | Slides font catalog | unsupported families → Roboto via `gslides.Substitutions` |

The font substitution applies to both routes. The other four are why `import`
exists.

## Scope and limitations

- One blank master/layout; slides are free-form shape canvases.
- No images, no chart parts (charts are drawn from shapes, which keeps
  them editable), no animations.
- Speaker notes ARE supported (`<aside class="notes">` → native notes in
  both backends), as are bulleted/numbered lists, hyperlinks, preset
  diagram shapes (diamond/cylinder/hexagon/chevron/…), elbow connectors,
  semi-transparent fills and rotated text.
- Fonts referenced by name, not embedded — stay on Slides-available fonts.
- HTML conversion is deliberately lossy for non-flat styling: gradients,
  shadows and transforms are approximated or dropped, with warnings from
  `deckgen check`.

## Development

```sh
go test ./...                   # root module: golden PPTX + Slides request tests
cd cmd/deckgen && go test ./...  # mapper fixtures (Chrome-free) + live-Chrome test
DECKGEN_E2E=1 go test -run PushE2E ./cmd/deckgen  # real Slides push (needs auth)
```

CI runs `go vet` and `go test` for both modules on every push and pull
request.

Golden fixtures pin all three stages: PPTX bytes (`testdata/golden/`),
Slides request JSON (`gslides/testdata/`), and scene→IR mapping
(`cmd/deckgen/internal/htmlconv/testdata/`).

### Fidelity calibration

After any change to the Slides emitter (inset constants in
`gslides/units.go`, autofit, table borders, font map) or to the import scaling
in `pptx/google_import.go`, compare Chrome's render against the Slides service:

```sh
deckgen screenshot examples/demo.html -o shots/          # what Chrome rendered
deckgen push examples/demo.html -title "calibration"     # prints URL with <id>
deckgen snap <id> -o shots-slides/                       # what Slides rendered
```

Overlay `shots/slideNN.png` and `shots-slides/slideNN.png`; text should
land on the same grid positions. If text sits consistently offset, adjust
`insetLR`/`insetTB` in `gslides/units.go`.
