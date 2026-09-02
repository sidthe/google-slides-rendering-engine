# keynote — a speaker template for deckgen

A complete voice + design system for conference-keynote decks: minimal
slides, huge type, drawn diagrams and mockups, plain human copy, the
detail in speaker notes. The speaker is the show; the slides are
backdrops.

## What a template is

The authoring contract (`docs/authoring.md` / `AGENTS.md`) defines what
*renders correctly*. A template defines what *reads well*: tone of voice,
composition rules, and a catalog of slide archetypes with known-good
geometry. An agent authoring a deck picks a template the way a speaker
picks a style — and the lint pass keeps the deck inside it.

## Files

| File | What it is |
|---|---|
| `VOICE.md` | writing rules: budgets, banned constructions with fixes, humor, narrative patterns |
| `DESIGN.md` | design rules + the archetype catalog, keyed to the skeleton |
| `deck.css` | stylesheet — imports the house sheet, adds the template's classes |
| `skeleton.html` | 23-slide sample deck, one archetype per slide, fully fictional content |
| `shots/` | rendered previews — not committed; generate with `deckgen screenshot skeleton.html -o shots/` |

## How an agent uses it

1. Read `VOICE.md`, then `DESIGN.md`. Skim `skeleton.html` — the comments
   name each archetype.
2. Start the new deck from the skeleton's `<head>` (it links this
   directory's `deck.css`) or copy `deck.css` next to your deck and link
   it there.
3. For each idea in the talk, pick an archetype from the catalog, copy
   that slide's geometry, replace the words under the voice rules.
4. Validate:

   ```sh
   deckgen check deck.html -lint keynote   # warnings + template findings
   deckgen screenshot deck.html -o shots/  # then look at every PNG
   ```

   `-lint keynote` enforces the floors: 18px minimum text, ≤ 70 visible
   words and ≤ 80 elements per slide, speaker notes everywhere. Findings
   are advisory — deliberate exceptions (a dense heatmap) are allowed,
   silence is not.

## Ground rules baked into this template

No kickers, footers, or page numbers. One idea and one accent color per
slide. Dark slides are deep blue act breaks; black belongs only to drawn
terminals. Charts and mockups are drawn from divs so the exported deck
stays fully editable. Every number on a slide carries its meaning in its
label, and every slide carries notes the speaker can say out loud.
