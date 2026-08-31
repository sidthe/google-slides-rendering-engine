# T2 — White-space fidelity in the HTML extractor

Tier: **T2**. The change is contained to one function group in
`cmd/deckgen/internal/htmlconv/extract.js`, but it alters the text of every
converted deck, changes an asset stylesheet, and moves committed golden
fixtures. Anything that touches all text output is not T1.

## Problem

`extract.js` reduces the inline content of a block element to paragraphs of
styled runs. It handles white space with two lines:

```js
text = text.replace(/\s+/g, " ");
if (!text.trim()) return;
```

The extractor's contract is that it mirrors what Chrome paints, because the
geometry it emits comes from Chrome and the screenshot gate that authors
review is a Chrome render. These two lines break that contract in five ways.

Measured against a Chrome render of the same markup:

| Input | Chrome paints | `main` extracts |
|---|---|---|
| `a<span> </span>b` | `a b` | `ab` |
| `<b>Bold</b> <i>Ital</i>` | `Bold Ital` | `BoldItal` |
| `one <b>two</b>  <b>three</b> four` | `one two three four` | `one twothree four` |
| `Price:&nbsp;&nbsp;&nbsp;42` | `Price:   42` | `Price: 42` |
| `<div style="white-space:pre">def f():\n    return 1</div>` | two lines, indented | `def f(): return 1` |

Causes:

1. **A white-space-only text node is dropped.** `!text.trim()` returns early,
   so the word space between two inline elements disappears. Words run
   together in the exported deck. This is the most damaging of the five
   because it is silent and affects ordinary prose, not just code.
2. **`\s` matches U+00A0.** In JavaScript both `\s` and `String.trim()` treat a
   non-breaking space as white space. CSS does not — NBSP is never collapsed.
   Deliberate NBSP spacing and NBSP-based code indentation are destroyed.
3. **`white-space` is ignored.** Every element is collapsed as if it computed
   to `normal`, so `pre`, `pre-wrap`, `pre-line` and `break-spaces` all lose
   their line structure and their indentation.
4. **Blank lines are dropped.** `paras.filter((p) => p.runs.length)` removes
   every empty paragraph, so a blank line separating two stanzas of code
   cannot survive even if line breaks did.
5. **Collapsing is per-text-node, not per-paragraph.** Chrome collapses a
   space at the end of one run against a space at the start of the next.
   The extractor cannot, so `a </b><b> b` yields a double space.

`hasDirectInlineText` and `inlineTextRect` carry defects 1 and 2 as well: both
gate on `node.textContent.trim()`, so a text node holding only an NBSP is
invisible to them, and under a preserving element so is a text node holding
only spaces.

There is no `white-space` declaration on `.code` in `assets/deck.css`, so the
authoring class that exists precisely to show code has no way to keep its
indentation. Authors work around it with `&nbsp;` runs, which defect 2 then
eats.

## Concept

Implement the CSS white-space processing model instead of approximating it,
and read the mode from the computed style so that Chrome remains the single
source of truth.

CSS splits the behaviour into two independent switches, exposed by Chrome on
the `white-space-collapse` longhand:

| `white-space` | `white-space-collapse` | spaces/tabs | segment breaks |
|---|---|---|---|
| `normal`, `nowrap` | `collapse` | collapse | collapse to a space |
| `pre`, `pre-wrap` | `preserve` | preserve | preserve |
| `pre-line` | `preserve-breaks` | collapse | preserve |
| `break-spaces` | `break-spaces` | preserve | preserve |

The extractor reads that longhand per node, falling back to a shorthand
lookup on older Chrome builds. `white-space` inherits, so a `<span>` inside a
`<pre>` reports `preserve` without any special case.

Collapsible white space in CSS is space, tab, line feed, carriage return and
form feed. U+00A0 is not in that set and is never touched.

Cross-run collapsing is handled in a second pass. Each run is tagged at
creation with whether it came from a collapsing context; after a paragraph is
closed, one walk over its runs collapses a space that meets another space
across a run boundary, strips leading white space at the start of the
paragraph and trailing white space at the end, and skips any run that came
from a preserving context.

### Approaches rejected

**Detect code blocks by tag name, class name or font family.** This was the
shape of the change proposed in PR #4: treat an element as preformatted when
it is a `<pre>` or `<code>`, or carries `class="code"`, or computes to a
monospace font. It fails because the flag then contradicts the rendering.
A caption that merely uses a monospace font computes to `white-space: normal`,
so Chrome collapses its source newlines and paints one line, while the
extractor splits it into three paragraphs and prefixes each with the HTML
file's own source indentation. The exported deck disagrees with the
screenshot the author approved, and the element overflows a box measured for
one line. The same mechanism silently adds four leading spaces to every line
of a `.code` block, because HTML source indentation becomes literal text.
Any rule that infers intent rather than reading the computed style has this
failure mode.

**Preserve everything and let the deck writer sort it out.** The writer has no
layout information and cannot know which spaces Chrome painted.

**Keep the single collapse and special-case NBSP only.** Fixes defect 2 and
nothing else.

### Stylesheet change

`.code` in `assets/deck.css` gains `white-space: pre-wrap` and `margin: 0`.
`pre-wrap` rather than `pre` so a long line wraps in the preview instead of
overflowing silently, which matches how a Google Slides text box behaves.
`margin: 0` so `<pre class="code">` positions like the `<div>` form.

`<pre class="code">` becomes the documented authoring form. The HTML parser
drops one newline immediately after a `<pre>` start tag, so the block can be
written on its own lines without a leading blank line. A `<div class="code">`
keeps that leading newline, which Chrome renders as a blank first line — the
extractor now reports it because Chrome paints it, and the screenshot shows
it.

## Spec

### White-space mode

`wsOf(cs)` returns `{ spaces, breaks }`, both booleans, meaning "preserve".
It reads `cs.whiteSpaceCollapse` and falls back to mapping `cs.whiteSpace`.
An unrecognised value maps to `collapse`.

### Text transformation, per text node

Let `ws` be the mode of the text node's parent element.

- `breaks` false: replace every run of `[ \t\r\n\f]` with a single space.
  Emit as one run. A run that is entirely white space is emitted, not
  dropped; the paragraph pass decides whether it survives.
- `breaks` true: normalise `\r\n` and `\r` to `\n`. When `spaces` is false,
  collapse every run of `[ \t\f]` to a single space and then remove spaces
  adjacent to a `\n`. Split on `\n`; each part after the first closes the
  current paragraph and starts a new one. Empty parts emit no run.

U+00A0 is not in any character class above and always survives.

### Paragraph pass

For each closed paragraph, over its runs in order:

- Collapse a leading space of a collapsible run when the previous emitted
  character was a space.
- Remove leading collapsible spaces while no non-space content has been seen.
- Remove trailing collapsible spaces from the end of the paragraph backwards
  while no non-space content has been seen.
- Drop runs left empty.

Runs from a preserving context are never modified.

### Empty paragraphs

- A block whose own computed mode has `breaks` false keeps the existing
  behaviour: empty paragraphs are dropped.
- A block whose mode has `breaks` true keeps empty paragraphs, so blank lines
  inside code survive, and drops exactly one trailing empty paragraph.
  Chrome removes a single segment break at the end of a block: a `pre` div
  containing `x\n` is one line high, and `x\n\ny` is three.

### `hasDirectInlineText` and `inlineTextRect`

Both stop using `String.trim()`. A text node counts as content when it holds
any character outside `[ \t\r\n\f]`, or, when its parent preserves spaces,
any character at all. NBSP therefore counts everywhere.

### Edge cases

- `<br>` still closes a paragraph, in every mode. A `<br>` next to a newline
  inside a `pre` block produces two breaks, which is what Chrome paints.
- `nowrap` collapses exactly like `normal`. Only wrapping differs, and
  wrapping is Chrome's job.
- A `pre` element nested in a normal card is `display: block`, so
  `collectParas` already skips it and `walk` emits it as its own text block.
  Its own mode applies there.
- A table cell reaches `collectParas` by the same path and gets the same
  treatment.
- Bullet text from `extractListParas` goes through `collectParas` per `<li>`,
  so list items get cross-run collapsing too.

## Acceptance criteria

1. `a<span> </span>b` extracts as one run reading `a b`.
2. `<b>Bold</b> <i>Ital</i>` extracts as runs whose concatenation is
   `Bold Ital`.
3. `one <b>two</b>  <b>three</b> four` concatenates to `one two three four`,
   with exactly one space between each pair of words.
4. `Price:&nbsp;&nbsp;&nbsp;42` keeps all three U+00A0 characters.
5. A `white-space: pre` block containing `def f(x):\n    return x + 1`
   extracts as two paragraphs, the second beginning with four spaces.
6. A `white-space: pre` block containing `a\n\nb` extracts as three
   paragraphs, the middle one empty.
7. A `white-space: pre` block whose text ends in `\n` produces no trailing
   empty paragraph.
8. A `white-space: pre-line` block containing `Line one     with    spaces`
   extracts with the inner spaces collapsed to one, and with no leading
   indentation.
9. A block using a monospace font but computing to `white-space: normal`
   extracts as a single paragraph with no leading spaces, matching the
   Chrome render.
10. A `<pre class="code">` block written with hierarchical indentation
    extracts with that indentation intact and no extra leading spaces.
11. `deckgen build` on such a deck writes those leading spaces into the
    PPTX text runs.
12. The existing golden and feature fixtures still pass, or their diffs are
    reviewed and are corrections in the direction of the Chrome render.

Criteria 1 to 10 are covered by live-Chrome tests in `htmlconv`, which skip
when Chrome is absent, following the existing `TestExtractLive` pattern.

## Risks and open questions

- **Golden fixtures move.** `features-scene.json` and the golden deck are
  committed extractions. Any inter-inline space that was previously eaten
  now appears. Each diff must be checked against a screenshot rather than
  accepted blindly.
- **Decks authored around the bug.** A deck that inserted `&nbsp;` to work
  around collapsed spacing now gets both the NBSP and the real space. This
  is a correction, but it changes existing output.
- **Google Slides import of leading spaces: verified, closed.** The PPTX
  carries them in `<a:t>`, and a real import on 2026-08-31 kept them. Code
  blocks indent correctly in Slides with plain spaces, so no NBSP padding is
  needed in the writer.
- **`white-space-collapse` support.** Chrome 114 and later expose the
  longhand. The shorthand fallback covers older builds, and
  `preserve-spaces` has no shorthand spelling, so it is reachable only via
  the longhand.
