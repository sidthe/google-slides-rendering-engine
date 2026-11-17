# T2 — Text and table layout fidelity across Chrome, PPTX, and Google Slides

Tier: **T2**. This change updates geometry extraction in
`cmd/deckgen/internal/htmlconv/extract.js`, IR mapping in `mapscene.go`,
optional paragraph/run/table fields in `ir/ir.go`, and both native emitters
(`pptx/` and `gslides/`). It changes text frames, line spacing, list indents,
character tracking, and row heights across converted HTML decks while keeping
programmatic Go builder (`deck.New()`) output byte-identical.

## Problem

Authors validate deck layouts against headless Chrome PNG renders
(`deckgen screenshot`), then export via `deckgen build` / `deckgen import`
(PPTX imported into Google Slides) or `deckgen push` (Slides API). Decks that
render without overlap in Chrome PNGs exhibit systemic text wrapping, vertical
growth, sibling collisions, and card/table overflows in Google Slides.

Six distinct mechanisms in the `HTML -> Scene -> IR -> PPTX / Slides` pipeline
compound to produce these overlaps (`STRONGLY DATA BACKED` / `VERIFIED`):

| # | Mechanism | Chrome PNG | Current PPTX / Slides output | Observed failure mode |
|---|---|---|---|---|
| 1 | CSS `line-height` (`em`-multiple) emitted directly as OOXML `<a:spcPct>` / Slides `lineSpacing` (`single-line`-multiple) | `line-height: 1.40` on `20px` text = `28px`/line (`392px` for 14 lines) | `spcPct="140000"` (`140%` of `1.20 * fontSize` = `33.6px`/line, `470px` for 14 lines) | **+20% vertical height inflation** on multi-line prose, lists, and `.code` blocks; bottom lines spill past card borders |
| 2 | `inlineTextRect` discards parent block CSS content-box width and `WrapSlack` disables slack when `H > FontSize * 1.7` | 2-line `<p>` inside `402px` card content box wraps at greedy glyph width `366px`; `<br>` hero title in `1440px` div measures `684px` | Emits `w = 366px` and `w = 684px` with `WrapSlack = false` (`0px` margin) | Minor advance-width or `0.01pt` rounding differences wrap lines onto an **extra line**, colliding with sibling `<ul>` or subtitle below |
| 3 | `<ul>`/`<ol>` bullet indent hardcoded to `342900` EMU unscaled in `google_import.go` + `<li>` CSS vertical margins dropped | `deck.css` `ul { padding-left: 26px }`, `li { margin: 8px 0 }` | `342900` EMU = `60 grid-px` (`27pt`) per level on the 10in Slides canvas; `0pt` inter-item gap; `ul` top includes first `<li>` BFC top margin | Bullet text loses `34px+` of width per nesting level inside cards, forcing extra wraps |
| 4 | CSS `text-transform` ignored in `extract.js` | `.kicker` (`text-transform: uppercase`) paints `SECTION LABEL` | Emits raw source case `section label` | Lowercase text in Slides disagrees with Chrome screenshot and metrics |
| 5 | CSS `letter-spacing` ignored in `extract.js` and `pptx` | Positive tracking (`.kicker` `2px`) widens Chrome bounds; negative tracking (`-1px` on titles) tightens Chrome bounds | Emits `0pt` character spacing | Untracked titles with negative CSS `letter-spacing` render wider in Slides than Chrome's measured box and wrap onto extra lines |
| 6 | `extractTable` averages uniform `rowH = r.h / rows.length` and `collectParas(td, true)` merges nested blocks | 4-row table with heights `[44, 44, 96, 44]` totals `228px`; `<div>A</div><div>B</div>` inside `<td>` breaks lines | Sets `minRowHeight = 57px` on all 4 rows; tall row still grows to `96px` (`57+57+96+57 = 267px`); nested `<td>` divs concatenate as `AB` | Table grows `+39px` taller than Chrome and overlaps elements below it |

## Research and empirical measurements

Measured on 2026-11-17 against headless Chromium and live Google Slides PPTX
imports on the `10in x 5.625in` (`9144000 x 5143500` EMU, `5715` EMU per grid
unit, `0.45pt` per CSS `px`) target canvas:

1. **Normal single-line height ratio (`VERIFIED`)**:
   - Headless Chrome renders `line-height: normal` for `Roboto` (`20px`) at
     `24.00px` per line (`240.00px` across 10 `<br>` lines = `1.20 * fontSize`)
     and `Roboto Mono` (`15px`, `20px`) at `18.00px` and `24.00px` per line
     (`1.20 * fontSize`).
   - Google Slides imports a paragraph without `<a:lnSpc>` as `lineSpacing: 100`
     ("Single"), rendering 14 lines of `9pt` (`20px`) `Roboto` and `6.75pt`
     (`15px`) `Roboto Mono` at `1.20 * fontSize` per line (`336px` and `252px`
     respectively).
   - Dividing the CSS `em` multiple by `1.20` (`1.40 / 1.20 = 1.1667` ->
     `<a:spcPct val="116667"/>` / `lineSpacing: 116.67`) lands line 14 flush
     inside Chrome's exact `392px` (`Roboto`) and `294px` (`Roboto Mono`)
     reference boxes.
2. **OOXML `<a:spcPts>` import behavior in Google Slides (`VERIFIED`)**:
   - Emitting `<a:lnSpc><a:spcPts val="1260"/></a:lnSpc>` (`12.60pt` exact for
     `9.00pt` text) does **not** produce fixed `12.60pt` spacing in Google
     Slides. The PPTX importer divides `12.60 / 9.00 = 1.40` and stores
     `lineSpacing: 140` (`140%` of single spacing), reproducing the exact same
     +20% vertical overflow as `spcPct="140000"`. Calibrated proportional
     spacing (`spcPct`) is therefore required.
3. **OOXML `<a:spcAft>` and bullet `marL`/`indent` on Slides import (`VERIFIED`)**:
   - Placing `<a:spcAft><a:spcPts val="360"/></a:spcAft>` (`8px * 0.45pt/px =
     3.6pt`) after `<a:lnSpc>` and before `<a:buFont>` per the DrawingML
     `CT_TextParagraphProperties` sequence imports cleanly as `3.6pt`
     inter-item spacing between consecutive bullets.
   - Emitting `marL="148590" indent="-148590"` (`26px * 5715` EMU = `11.7pt` on
     the 10in canvas) imports as `indentStart: 11.7pt`, `indentFirstLine:
     4.5pt`, aligning bullet text flush with Chrome's `padding-left: 26px` line.
4. **Advance widths and `<a:rPr spc="..."/>` (`VERIFIED`)**:
   - Untracked `Roboto` (400, 700) and `Roboto Mono` (400) runs in Google
     Slides measure `0.99x–1.00x` of Linux headless Chrome's painted width.
   - `<a:rPr spc="90"/>` (`2px * 0.45pt/px = 0.90pt` in hundredths of a point)
     is honored visually by Google Slides on PPTX import, matching Chrome's
     `283.75px` `.kicker` box width.

## Concept

Keep Chrome as the layout authority and fix unit/box mismatches at the exact
boundary where each distortion enters:

```text
  CSS block element (card width 454px, padding 26px -> content box 402px)
  +-------------------------------------------------------------------+
  |  GKE's go-forward declarative API                                 |  <- painted width: 312px
  |  (cloud.google.com/v1) for node provisioning:                     |  <- painted width: 366px
  +---------------------------------------------------------+---------+
  |<------------------ inlineTextRect (366px) ------------->|         |
  |<------------------ CSS content box (402px) ---------------------->|

  Before: frame W = 366px, WrapSlack = false, Spacing = 140% (1.68x em)
  After:  frame W = 402px, Y/H from painted glyphs, Spacing = 116.7% (1.40x em)
```

### Approaches rejected

1. **Emit OOXML `<a:spcPts>` fixed point line-height instead of `<a:spcPct>`.**
   Rejected after live Google Slides import testing proved that Slides divides
   `spcPts / fontSize` into `lineSpacing` percentage (`12.6pt / 9pt -> 140%`),
   leaving the +20% vertical overflow unsolved, and the Slides REST API
   (`gslides`) has no fixed-point line-spacing field.
2. **Replace `inlineTextRect` with `el.getBoundingClientRect()` everywhere.**
   Rejected because `el.getBoundingClientRect()` includes padding, borders,
   vertical flex centering space, and preceding block children (`<b>Label</b>Body`
   in `TestExtractUsesPaintedBoundsForDirectText`), which overlays direct text
   on top of block labels and shifts flex-centered `.pill`/`.stage` text to the
   top-left corner.
3. **Add unconditional `+12%` `WrapSlack` to all multi-line text boxes.**
   Rejected because adding `+12%` to an already soft-wrapped paragraph wider
   than its container can unwrap words from line 2 onto line 1 and push slide
   titles past `x = 1600`. Restoring the true CSS content-box width solves
   soft-wrap narrowing within the author's exact container bounds, while
   `WrapSlack` remains restricted to text where each paragraph is a single
   visual line (and tight to its glyph bounds).
4. **Change the meaning of `ir.Para.Spacing` in `pptx` and `gslides`.**
   Rejected because the Go builder (`deck.New()`, `style.go`) already authors
   `Para.Spacing` in renderer single-line multiples (`1.0`, `1.1`, `1.15`,
   `1.2`), pinned byte-for-byte by `TestGoldenPackage` and `gslides` golden
   tests. Normalizing CSS `em` line-height in `htmlconv` (`mapscene.go`) aligns
   HTML extraction with the existing IR contract without touching builder
   output.

## Spec

### 1. CSS `line-height` normalization (`mapscene.go`, `pptx/shapes.go`)

- Define `const cssNormalLineHeight = 1.2` in `htmlconv`.
- `cssSpacing(s float64) float64`: when `s <= 0`, return `0`; otherwise return
  `math.Round((s/cssNormalLineHeight)*1e4) / 1e4`.
- Apply `cssSpacing` in `mapText` and `mapTable` (extracting per-cell
  `spacing: spacingOf(getComputedStyle(td))` in `extractTable`).
- In `pptx/shapes.go`, round `<a:spcPct val="%d"/>` with `math.Round` instead
  of integer truncation, and emit `<a:lnSpc>` on table cell paragraphs when
  `p.Spacing > 0`.

### 2. Content-box horizontal bounds and single-line-per-paragraph `WrapSlack` (`extract.js`, `scene.go`, `mapscene.go`, `pptx/shapes.go`, `gslides/emit.go`)

- In `extract.js`, add `textBoxRect(el, cs, textRect)`:
  - Preserve `top = textRect.top` and `height = textRect.height` in all cases.
  - Compute `[contentLeft, contentRight]` from `el.getBoundingClientRect()`
    inset by computed `borderLeftWidth + paddingLeft` and
    `borderRightWidth + paddingRight`, clamped to `[sRect.left, sRect.right]`
    whenever `textRect` itself lies inside the `<section>`.
  - For non-flex/grid block containers (`!cs.display.includes("flex") &&
    !cs.display.includes("grid")`) with `contentW >= textRect.width - 0.5`:
    - `align === "l"` and `textRect.left >= contentLeft - 2`: set
      `left = textRect.left <= contentLeft + 4 ? contentLeft : textRect.left`
      and `right = Math.max(contentRight, textRect.right)`.
    - `align === "ctr"` and center midpoint matches within `3px`: set
      `left = contentLeft, right = contentRight`.
    - `align === "r"` and `Math.abs(contentRight - textRect.right) <= 3`: set
      `left = contentLeft, right = contentRight`.
  - For flex/grid containers:
    - When `align === "ctr"` and center midpoint matches within `3px`, widen to
      `[contentLeft, contentRight]`.
    - When `align === "l"` and the text already wrapped across multiple lines
      (`textRect.height > px(cs.fontSize) * 1.7` and `contentRight > textRect.right`),
      extend `right = contentRight` while keeping `left = textRect.left`.
  - Set `noSlack: true` on the extracted `Element` when the resulting `width >=
    textRect.width * 1.12`.
- In `mapscene.go`, `needsWrapSlack(el)`:
  - Return `false` if `el.NoSlack`.
  - Return `true` if `el.FontSize <= 0` or `el.H <= el.FontSize*1.7`.
  - Return `true` when `len(el.Paras) > 1` and `el.H <=
    (float64(len(el.Paras))+0.45)*paraLinePx(el)` (every `<br>` or `pre`
    paragraph is a single visual line; constant `+0.45` tolerance avoids false
    positives on 3+ paragraph blocks where one paragraph soft-wrapped).
- In `pptx/shapes.go` and `gslides/emit.go`, when `WrapSlack` expands a text
  box whose original `X >= 0` and `X+W <= 1600`, clamp expansion across all
  alignments (`"l"`, `"ctr"`, `"r"`) so the expanded frame remains within
  `[0, 1600]` (with symmetric clamping for `"ctr"` to preserve center).

### 3. List geometry, scaled hanging indents, and inter-item spacing (`extract.js`, `ir/ir.go`, `pptx/`, `gslides/`)

- Add `BulletIndent float64` (points hanging step, `0` = default),
  `BulletMargin float64` (cumulative left margin in points when nested list
  indents are non-uniform, `0` = `BulletIndent * (Level + 1)`), and
  `SpaceAfter float64` (points below paragraph, `0` = none) to `ir.Para`
  (`json:",omitempty"`).
- In `extract.js`:
  - `extractListParas` records each `<li>`'s list `paddingLeft` (`px`),
    cumulative nested margin, `inlineTextRect(li, true)` (crossing non-list
    blocks like `<li><p>...</p></li>`), empty `<li></li>` items, and computed
    vertical margins. Intra-item breaks (`<li>A<br>B</li>`) join via `U+000B`
    soft breaks (`<a:br>` in PPTX, `\v` in Slides API) so ordered/bulleted
    lists keep one marker per `<li>`.
  - Position the list `Element` vertically at the first item's painted `top`
    (`firstRect.top`) with `h = lastRect.bottom - firstRect.top` so BFC top
    margins on positioned `<ul>` elements match Chrome's painted first line.
  - Between consecutive list items `i` and `i+1`, compute the inter-item CSS
    margin gap (`spaceAfter` on item `i`'s final paragraph) from collapsed
    `marginBottom`/`marginTop` (verified against painted rect deltas).
- In `pptx/google_import.go`: scale `Para.BulletIndent`, `Para.BulletMargin`,
  and `Para.SpaceAfter` by `googleImportScale` (`0.75`).
- In `pptx/shapes.go`:
  - Default bullet step (when `BulletIndent == 0`) is `171450` EMU (`13.5pt`,
    matching `22.5px` on 13.333in PPTX); when `BulletIndent > 0`, `step =
    int(math.Round(p.BulletIndent * 12700))` (`26px` -> `15.6pt` -> `11.7pt` =
    `148590` EMU on Slides). Emit `marL` (`BulletMargin` if set, else
    `step*(Level+1)`) and `indent = -min(step, marL)`.
  - When `p.SpaceAfter > 0`, emit `<a:spcAft><a:spcPts val="%d"/></a:spcAft>`
    immediately after `<a:lnSpc>` and before bullet child elements.
- In `gslides/emit.go` and `gslides/requests.go`:
  - Add `IndentStart`, `IndentFirstLine`, and `SpaceBelow` (`*Dimension`) to
    `ParagraphStyle`, emitted only when `BulletIndent > 0` or `SpaceAfter > 0`.

### 4. CSS `text-transform` and `letter-spacing` (`extract.js`, `ir/ir.go`, `pptx/`)

- In `extract.js`:
  - Record `textTransform` (`cs.textTransform`) per run and apply `uppercase`,
    `lowercase`, and word-boundary `capitalize` across each paragraph after
    white-space collapsing.
  - Record `letterSpacing` (`px`, `0` when `normal`) on each run.
- Add `LetterSpacing float64` (`pt`, `json:",omitempty"`) to `ir.Run`.
- Scale `Run.LetterSpacing` by `googleImportScale` in `pptx/google_import.go`
  and emit ` spc="%d"` (`int(math.Round(r.LetterSpacing * 100))`) on `<a:rPr>`
  in `pptx/shapes.go` when `r.LetterSpacing != 0`.

### 5. Per-row table heights and block breaks inside cells (`extract.js`, `ir/ir.go`, `pptx/`, `gslides/`)

- Add `RowHs []float64` (`json:",omitempty"`) to `ir.Table`. Emitters use
  `RowHs[ri]` when `len(RowHs) == len(Rows)` and fall back to uniform `RowH`
  otherwise.
- In `extract.js`:
  - Populate `rowHs` from each `<tr>`'s `getBoundingClientRect().height`.
  - In `collectParas(el, true)`, flush paragraph boundaries before and after
    nested non-inline elements so `<td><div>A</div><div>B</div></td>` produces
    two paragraphs instead of `AB`.
- Scale `RowHs` in `pptx/google_import.go`, emit `<a:tr h="%d">` per row in
  `pptx/shapes.go`, and group `UpdateTableRowProperties` by distinct height in
  `gslides/emit.go`.

## Milestones

1. **M1 — IR & emitters (`ir`, `pptx`, `gslides`)**: Add optional fields
   (`BulletIndent`, `SpaceAfter`, `LetterSpacing`, `RowHs`), update
   `pptx/shapes.go`, `pptx/google_import.go`, `gslides/requests.go`, and
   `gslides/emit.go`, and verify existing builder goldens (`TestGoldenPackage`,
   `gslides` goldens) remain byte-identical.
2. **M2 — HTML extractor & scene mapper (`extract.js`, `scene.go`, `mapscene.go`)**:
   Implement content-box width extraction, `cssSpacing`, list geometry/margins,
   `text-transform`, `letter-spacing`, per-row table heights, and `<td>` block
   flushing.
3. **M3 — Unit, live-Chrome, and golden fixture verification**: Add targeted
   regression tests across `htmlconv`, `pptx`, and `gslides`; refresh HTML
   scene/IR fixtures and verify every diff against Chrome renders.
4. **M4 — End-to-end Google Slides import visual gate & adversarial review**:
   Build a multi-archetype stress-test deck, compare `deckgen screenshot` PNGs
   against imported Google Slides PNGs (`0` overlaps/overflows), and complete
   adversarial verification passes.

## Acceptance criteria

1. CSS `line-height: 1.40` maps to `ir.Para.Spacing` `1.1667` (`<a:spcPct
   val="116667"/>` in PPTX, `lineSpacing: 116.67` in Slides API), and a 14-line
   `Roboto` or `Roboto Mono` block fits within Chrome's rendered box height
   after Google Slides import.
2. A 2-line soft-wrapped `<p>` inside a padded `454px` card (`padding: 26px`)
   extracts with `w = 402` (content-box width) rather than the narrower painted
   glyph union, while keeping `y` and `h` from painted glyph bounds.
3. A `<div class="card"><b>Label</b>Body text</div>` card with `text-align:
   center` still places `Body text` below `Label` (`bodyY > labelY`) and keeps
   it centered at the card midpoint.
4. Flex-centered single-line pills (`.pill`, `.stage`, `.actor`) without
   `text-align: center` retain their horizontally and vertically centered glyph
   positions.
5. Multi-paragraph `<br>` and `<pre class="code">` blocks where no paragraph
   soft-wraps retain `WrapSlack` when tight to glyph bounds.
6. `<ul>` with `padding-left: 26px` and `li { margin: 8px 0; line-height:
   1.35 }` emits scaled `marL="148590" indent="-148590"` (`26 grid-px`),
   `spcPct="112500"`, and `spcAft="360"` (`3.6pt = 8 grid-px`) in
   `WriteGoogleSlides`.
7. CSS `text-transform: uppercase`, `lowercase`, and `capitalize` transform
   extracted run text to match Chrome's rendered glyphs.
8. CSS positive and negative `letter-spacing` emit scaled `<a:rPr spc="..."/>`
   in `WriteGoogleSlides` (`2px` -> `spc="90"`, `-1.5px` -> `spc="-68"`).
9. A table with mixed single-line and multi-line rows emits distinct per-row
   heights matching each `<tr>`'s Chrome height, and `<td><div>A</div><div>B</div></td>`
   extracts as two paragraphs.
10. `TestGoldenPackage` and all existing `gslides` builder golden tests pass
    without modification.

## Performance

- `extract.js` performs `O(N)` DOM walks with no extra layout reflows (`getComputedStyle`
  and `getBoundingClientRect` read cached layout state alongside existing calls).
- Target extraction + build latency on the 3-slide `examples/demo.html` and
  `examples/features.html` decks remains unchanged (~0.6–1.2s dominated by
  headless Chrome startup).

## Risks and learnings

- **Google Slides `<a:spcPts>` normalization**: Google Slides converts fixed
  point line spacing in OOXML (`<a:spcPts>`) to proportional `lineSpacing` by
  dividing by run `fontSize`. Emitters targeting Slides import must calibrate
  `<a:spcPct>` against `1.20 * fontSize` rather than emitting `<a:spcPts>`.
- **DrawingML `<a:pPr>` child element sequence**: OOXML requires strict child
  ordering inside `<a:pPr>`: `lnSpc`, `spcBef`, `spcAft`, then `buFont` /
  `buChar` / `buAutoNum`. Out-of-order children are silently dropped by
  importers.
- **Flexbox centering without `text-align: center`**: Starter components like
  `.pill` center single-line labels via `display: flex; justify-content: center`
  while leaving `text-align: start`. Expanding `x`/`w` to `contentLeft`/`contentW`
  on such flex containers would left-shift the label; restricting left-aligned
  container expansion to block formatting contexts preserves flex alignment.
