# Gemini CLI guide

Before generating or changing a deck, read the complete authoring contract in
[`authoring.md`](authoring.md). It defines the HTML contract, the validation
workflow, design rules, and Google Slides fidelity boundaries.

Quick reference:

```sh
deckgen check      deck.html                  # extract and fix warnings
deckgen screenshot deck.html -o shots/        # visual self-review
deckgen build      deck.html -o deck.pptx     # native PPTX
deckgen push       deck.html -title "Title"   # native Google Slides
```

Slides are 1600×900 `<section>`s styled with `assets/deck.css`; every
supported element becomes a native, editable deck object rather than an image.
