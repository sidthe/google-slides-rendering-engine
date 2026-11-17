# T2 — Readable typography and shallower bullet indents

Tier: **T2**. The change touches the starter stylesheet (`assets/deck.css`), the
PPTX bullet emitter (`pptx/shapes.go`), the authoring contract, all HTML
examples, the keynote template, and committed scene/IR golden fixtures.

## Problem

Sub-heading and body text across `assets/deck.css`, `examples/*.html`, and
`templates/keynote/` sits at `15px–20px` CSS px. On Google Slides' 10-inch
import canvas, `0.45 pt/px` maps `15px–20px` to `6.75pt–9.0pt` — too small
to read comfortably in a meeting or presentation.

At the same time, bulleted lists use a deep horizontal indent (`padding-left:
26px` in `assets/deck.css`, and `342900` EMU = `60` grid-px on Slides in
`pptx/shapes.go`). In narrow cards and multi-column layouts, that deep hanging
indent wastes horizontal space and forces early line wraps.

## Concept

Raise the small end of the typography ladder by `1px–2px` across the starter
stylesheet, authoring guides, examples, and keynote template, and cut the list
indentation in half so lines stay compact without crowding bullet markers.

Specifically:

- **Starter stylesheet (`assets/deck.css`)**: bump `.card .card-title` (`23px`
  -> `25px`), `ul, ol` (`20px` -> `22px`, `padding-left: 26px` -> `18px`),
  `.tile .label` (`17px` -> `19px`), `.pill` (`15px` -> `17px`, height `44px`
  -> `46px`, `border-radius: 23px`), `table` (`18px` -> `20px`), `.code`
  (`20px` -> `21px`, padding `30px 36px` -> `28px 30px`), and `.foot` (`16px`
  -> `18px`).
- **PPTX bullet hanging indent (`pptx/shapes.go`)**: change `bulletXML` from
  `342900` EMU to `137160` EMU (`18px * 7620` EMU/px on PowerPoint; `24`
  grid-px on the 10-inch Slides import canvas), scaling level indentation as
  `137160 * (p.Level + 1)`.
- **Box sizing**: expand any tile, table, label box, or note container whose
  larger text would otherwise wrap early or collide with neighboring elements.
- **Legacy Go builder untouched**: `internal/demo/demo.go` and `style.go` stay
  unchanged so `TestGoldenPackage` in `golden_test.go` remains byte-identical.

## Spec

### Starter stylesheet type scale

| Selector | Property | Before | After |
|---|---|---|---|
| `.card .card-title` | `font-size` | `23px` | `25px` |
| `ul, ol` | `font-size` | `20px` | `22px` |
| `ul, ol` | `padding-left` | `26px` | `18px` |
| `.tile .label` | `font-size` | `17px` | `19px` |
| `.pill` | `font-size` / `height` / `border-radius` | `15px` / `44px` / `22px` | `17px` / `46px` / `23px` |
| `table` | `font-size` | `18px` | `20px` |
| `.code` | `font-size` / `padding` | `20px` / `30px 36px` | `21px` / `28px 30px` |
| `.foot` | `font-size` | `16px` | `18px` |

### PPTX bullet hanging indent

`bulletXML` in `pptx/shapes.go` defines `bulletIndentEMU = 137160` and emits:

```xml
<a:pPr marL="137160" indent="-137160">
```

for `Level == 0`, and `marL="274320" indent="-137160"` for `Level == 1`.

### Authoring guides and keynote template

- `AGENTS.md` and `docs/authoring.md` document the updated scale (`card titles
  25, body 22–24, labels 19, footers/ticks 18; nothing under 17px`) and the
  `18px` / `137160` EMU list indent.
- `templates/keynote/DESIGN.md` and `templates/keynote/skeleton.html` raise
  `18px–24px` support, annotation, caption, and mockup sizes by `1px–2px`
  while keeping `0` warnings and `0` findings under `-lint keynote`.

## Acceptance criteria

1. `pptx/google_import_test.go` includes a unit test pinning `marL="137160"
   indent="-137160"` for `Level: 0` and `marL="274320" indent="-137160"` for
   `Level: 1`.
2. `TestGoldenPackage` in `golden_test.go` passes without modification to
   `internal/demo/demo.go`, `style.go`, or `testdata/golden/pptx/`.
3. `deckgen check` reports `0` warnings on `examples/{demo,features,charts,diagrams}.html`
   and `0` warnings, `0` lint findings on `templates/keynote/skeleton.html -lint keynote`.
4. Visual inspection of every PNG rendered by `deckgen screenshot` on all four
   example decks confirms crisp legibility with no overlaps, truncation, or
   unwanted wrapping.
5. Regenerated scene and IR fixtures (`demo-scene.json`, `features-scene.json`,
   `diagrams-scene.json`, `demo-ir.json`) and both Go module test suites
   (`go test -count=1 ./...` at root and in `cmd/deckgen`) pass.

## Risks and open questions

- **Scene and IR fixtures move**: bumping `deck.css` and `examples/*.html`
  alters extracted bounding boxes and font sizes in the committed JSON
  fixtures. All scene assertions (`TestMapScenesShapes`, `TestMapScenesFeatures`,
  `TestMapScenesDiagrams`) must still pass after regeneration.
- **Line wrapping in narrow tiles**: increasing `.tile .label` from `17px` to
  `19px` can push long labels onto an extra line; tile widths/heights in
  `examples/demo.html` and `examples/charts.html` are expanded accordingly and
  verified by screenshot inspection.
