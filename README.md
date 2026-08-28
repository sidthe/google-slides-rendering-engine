# deck-engine

Turn HTML decks into **fully native, editable presentations** — PPTX files
and Google Slides — with no images, no screenshots, no embedded renders.
Every box becomes a real shape, every text a real text box, every table a
real table.

Built for AI harnesses (Gemini CLI, Claude Code, …): the model writes
HTML/CSS — which models are very good at — headless Chrome does the layout,
and `deckgen` rebuilds the result natively. See **AGENTS.md** for the
authoring guide a harness should follow.

```
 deck.html (1600×900 <section> per slide, any CSS)
     │  headless Chrome + injected extractor
     ▼
 scene graph (computed geometry + styles)
     │  pure Go mapper
     ▼
 ir.Deck (typed intermediate representation)
     ├─▶ pptx    → native .pptx (OOXML DrawingML)
     └─▶ gslides → Slides API batchUpdate → docs.google.com/… URL
```

## Quick start (CLI)

```sh
cd cmd/deckgen && go build -o deckgen .

./deckgen check      ../../examples/demo.html               # warnings report
./deckgen screenshot ../../examples/demo.html -o shots/     # per-slide PNGs
./deckgen build      ../../examples/demo.html -o demo.pptx  # PPTX tuned for Slides import
./deckgen build      ../../examples/demo.html -target powerpoint -o demo-powerpoint.pptx
./deckgen push       ../../examples/demo.html -title Demo   # Google Slides
```

Requirements: Chrome/Chromium on PATH (or `DECKGEN_CHROME=/path/to/chrome`).

`build` is entirely local. Its default `google-slides` target uses a 10in
Slides-compatible PPTX canvas and preserves table/text geometry through the
Google Slides import path. Use `-target powerpoint` for a conventional
PowerPoint-sized file.

### Google auth for `push` and optional `import`

`deckgen push` needs OAuth credentials with the
`https://www.googleapis.com/auth/presentations` scope. Two options:

1. **Application Default Credentials** (tried first):

   ```sh
    gcloud auth application-default login \
     --scopes=openid,https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/presentations,https://www.googleapis.com/auth/drive.file
   ```

2. **OAuth client JSON**: create an OAuth client (type: Desktop app) in
   Google Cloud Console (with the Slides API enabled), download the JSON,
   then `deckgen push … -oauth-client client_secret.json` (or save it as
   `~/.config/deckgen/client_secret.json`). A browser consent tab opens
once; the token is cached at `~/.config/deckgen/token.json`.

`deckgen import deck.html -title Demo` is optional: it uploads a locally
generated PPTX and converts it into a native Google Slides presentation. It
uses the narrowly scoped `drive.file` permission (only files deckgen creates
or opens), so the Google Drive API must be enabled in the OAuth project. If
you prefer not to enable Drive API, import the locally generated PPTX through
the Google Slides UI, then use `deckgen snap <presentation-id>` to inspect
Google's own rendered PNGs. Never commit an OAuth client JSON or token file.

## Quick start (Go library)

The original builder API still works and stays dependency-free:

```go
d := deck.New()
s := d.Slide()
deck.Title(s, "Hello")
deck.Tile(s, 80, 180, 340, 150, "42", "the answer", deck.BLUE50, deck.BLUE7, 40)
d.Save("out.pptx")                                  // PPTX
res, _ := gslides.Push(ctx, httpClient, d.IR(), "Hello") // Google Slides
```

Run the demo: `go run ./example demo.pptx`

## Layers

| Layer | Path | What it gives you |
|---|---|---|
| IR | `ir/` | typed deck model: `Rect`, `Oval`, `Text`, `Line`, `Table` on a 1600×900 grid; the contract between front ends and emitters |
| PPTX emitter | `pptx/` | native OOXML package writer (stdlib only) |
| Slides emitter | `gslides/` | pure `ir.Deck → batchUpdate` request plan + `Push` (works with any authorized `*http.Client`; no Google SDK) |
| Builder API | root | `Deck`, `Slide`, shape methods, `P/R/B/BC/RC` text helpers |
| Style kit | `style.go` | Go-level house components (`Title`, `Tile`, `Card`, `BarChart`, …) |
| HTML front end | `cmd/deckgen/` | headless-Chrome extraction, scene→IR mapper, CLI, Google auth |
| Starter CSS | `assets/deck.css` | the house look as CSS classes + palette variables |

The root module has **zero external dependencies**; Chrome, chromedp and
OAuth live only in the nested `cmd/deckgen` module.

## Google Slides fidelity

Slides API presentations are 10in wide (PPTX: 13.33in): all geometry and
point sizes scale by exactly 0.75, so decks are proportionally identical.
Known gaps (also in AGENTS.md, so harnesses design around them):

| Gap | Cause | Effect |
|---|---|---|
| corner radius | API can't set geometry adjust values | Slides' default rounding |
| text insets | API can't zero text-box insets | frames auto-compensated by the default insets |
| table row height | `minRowHeight` only | overflowing text grows rows |
| table cell margins | not settable | slight text offset inside cells |
| fonts | Slides font catalog | `Google Sans` → Roboto etc. via `gslides.Substitutions` |

## Scope and limitations

- One blank master/layout; slides are free-form shape canvases.
- No images, no chart parts (charts are drawn from shapes, which keeps
  them editable), no animations.
- Speaker notes ARE supported (`<aside class="notes">` → native notes in
  both backends), as are bulleted/numbered lists, hyperlinks, preset
  diagram shapes (diamond/cylinder/hexagon/chevron/…), elbow connectors,
  semi-transparent fills and rotated text — see AGENTS.md.
- Fonts referenced by name, not embedded — stay on Slides-available fonts.
- HTML conversion is deliberately lossy for non-flat styling: gradients,
  shadows and transforms are approximated or dropped, with warnings from
  `deckgen check`.

## Development

```sh
go test ./...                  # root module: golden PPTX + Slides request tests
cd cmd/deckgen && go test ./...# mapper fixtures (Chrome-free) + live-Chrome test
DECKGEN_E2E=1 go test -run PushE2E ./cmd/deckgen  # real Slides push (needs auth)
```

Golden fixtures pin all three stages: PPTX bytes (`testdata/golden/`),
Slides request JSON (`gslides/testdata/`), and scene→IR mapping
(`cmd/deckgen/internal/htmlconv/testdata/`).

### Fidelity calibration

After any change to the Slides emitter (inset constants in
`gslides/units.go`, autofit, table borders, font map), compare Chrome's
render against Google's:

```sh
deckgen screenshot examples/demo.html -o shots/          # what Chrome rendered
deckgen push examples/demo.html -title "calibration"     # prints URL with <id>
deckgen snap <id> -o shots-slides/                       # what Google rendered
```

Overlay `shots/slideNN.png` and `shots-slides/slideNN.png`; text should
land on the same grid positions. If text sits consistently offset, adjust
`insetLR`/`insetTB` in `gslides/units.go`.
