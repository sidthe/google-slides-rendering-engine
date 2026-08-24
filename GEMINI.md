# deck-engine — Gemini CLI guide

Read **AGENTS.md** in this repository: it is the complete guide for
generating decks here (HTML authoring contract, `deckgen` workflow, design
rules, Google Slides fidelity notes). Follow it exactly.

Quick reference:

```sh
deckgen check      deck.html                  # extract + warnings
deckgen screenshot deck.html -o shots/        # visual self-review
deckgen build      deck.html -o deck.pptx     # native PPTX
deckgen push       deck.html -title "Title"   # native Google Slides, prints URL
```

Slides are 1600×900 `<section>`s styled with `assets/deck.css`; every
element becomes a native, editable shape — no images.
