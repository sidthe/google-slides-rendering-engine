# keynote — design guide (for the authoring agent)

Minimal keynote style: the speaker is the show, slides are backdrops.
Every rule here is checkable; `deckgen check -lint keynote` enforces the
floors. Copy geometry from `skeleton.html` — each slide there is one
archetype, labelled in a comment.

## Global rules

1. **Type ladder** (px): statements 76–112 · hero numbers 110–280 ·
   headlines 44–56 · core lines 26–30 · support 22–24 · annotations 20 ·
   mockup mono 18. **Floor: 18px — nothing smaller, anywhere.** Two sizes
   on one slide must differ by ≥ 20%.
2. **One idea per slide.** ≤ 70 visible words, ≤ 80 native elements,
   ≤ 5 content blocks. Margins 80px; one focal mass, off-center;
   whitespace is deliberate.
3. **No chrome.** No kickers, footers, page numbers, or section labels.
   Slide comments in the HTML are for authors, not viewers.
4. **Color discipline.** Ink on white; **one accent per slide**, chosen by
   meaning: blue = the system, green = shipped/human decision,
   red = failure, yellow = caution/waiting. Dark act-break slides are
   full-bleed `--blue-dark` (use `section.dark`), 3–4 per hour. **No black
   objects** — `--dark` is reserved for drawn terminals only. On dark
   slides: white, secondary at `rgba(255,255,255,.85)`, never below `.70`.
5. **Connectors** are 3px (`.line`), start and end within 4px of what they
   connect, `data-arrow` for direction, `data-dashed` for return/boundary
   paths. Diagonals: width = √(dx²+dy²), rotate = atan2(dy,dx),
   `transform-origin: 0 50%`.
6. **Draw, don't paste.** Mockups (windows, terminals, menu bars, cards)
   and charts are built from divs so every element stays editable.
   `<img>` is a last resort for real logos/photos and is skipped in native
   export — budget for re-inserting those after export.
7. **Speaker notes on every slide** — that is where the cut detail lives.

## Archetype catalog (→ skeleton.html slide)

| # | Archetype | Use when | Key spec |
|---|---|---|---|
| A1 | Title | opening | claim as title, 112px; brand dots; byline 20px |
| A2 | Statement | one claim carries the beat | 76px, left 80, top third; optional 30px second line |
| A3 | Dark act break | a turn in the story | `section.dark`; setup 44px @.85, claim 84px white |
| A4 | Big number | one metric is the message | 280px number, 34px label under, source line 22px |
| A5 | Number trio | three related figures | 190px, same baseline, left-aligned columns |
| A6 | Chevron flow | a linear process | pentagon + chevrons, 116px tall; one word each; 20px captions under; payoff stage in green |
| A7 | Stage | step N of a journey | 110px blue number; 52px name; 30px core; 3 facts at 23px; *enter/exit* rails at bottom |
| A8 | Decision flow | pipeline with a branch | mono trigger block → nodes → diamond → green/yellow outcomes |
| A9 | Layered stack | a system's layers | 82px slabs, one accented; swappable part = dashed circle + arrow |
| A10 | Mind map | one center, many satellites | spokes drawn first (under the circle); plain text tips, no boxes |
| A11 | Spectrum | audience self-location | `data-arrow="both"` line, 3 dots, labels above; invitation line below |
| A12 | Comparison split | old way vs new way | hairline divider; old side rendered gray; new side is a drawn artifact |
| A13 | Quote | someone said it better | 44px italic, 6px blue left border, attribution + date |
| A14 | Minimal table | ≤ 6 short mappings | 2px header rule, 1px row rules, no cell fills, mono for artifact names |
| A15 | App window | show the product | `.win` chrome, mono content, one accent button, outcome card + arrow |
| A16 | Terminal | show the machinery | `--dark` background (the black exception); output has an arc: snag in yellow, recovery in green |
| A17 | Menu bar | ambient utility | light `--surface` strip, icon, panel hanging beneath |
| A18 | Bar chart | trend, few periods | hairline gridlines, 2px baseline, values above bars, story bar in second shade |
| A19 | Line chart | trend, continuous | rotated `.line` segments dot-to-dot, 14px markers, labels at points |
| A20 | Stacked bars | two-part composition | two shades of one hue, totals above, legend as one caption line |
| A21 | Heatmap | habit / density | 24px cells, 6px gaps, 4 intensity levels of one hue, month labels |
| A22 | Video placeholder | a demo clip goes here | `--surface` full-bleed, 200px play circle, mono caption |
| A23 | Checklist close | the ending | answer the opening; 3 drawn checkboxes, imperative, mono 24px |

## Self-review before `deckgen check`

- [ ] Can the one idea be read from ten meters? (squint test on the screenshot)
- [ ] Anything under 18px? Any annotation in `--ink-3` under 22px?
- [ ] More than one accent color doing work on one slide?
- [ ] Any arrow that does not touch what it points at?
- [ ] Any slide where the audience must *read* rather than *listen*?
- [ ] Notes present and speakable on every slide?

Then: `deckgen check deck.html -lint keynote` → fix every warning and
finding → `deckgen screenshot` → look at every PNG → build/push.
