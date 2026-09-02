# keynote — voice guide (for the authoring agent)

You are writing words a speaker will say next to, not a document the
audience will read. Apply every rule below before running `deckgen check`.

## Hard budgets

- One idea per slide. If a slide makes two points, split it.
- ≤ 12 words for the idea itself (headline / statement / number label).
- ≤ 70 visible words per slide, total (`-lint keynote` enforces this).
- Everything you cut goes into `<aside class="notes">` — 2–4 sentences a
  speaker can actually say. Every slide has notes.

## Banned constructions — with the fix

| Banned pattern | Example (do not write) | Write instead |
|---|---|---|
| Paired aphorism | "Adoption is solved. Industrialization is not." | One plain claim: "Most teams have not industrialized this." |
| Antithesis pair | "People do not want more to read. They want permission to skip." | Keep one half: "People want permission to skip." |
| Dramatic appositive | "the method, learned the hard way" · "The nightly job, from the inside." | "method" · "The nightly job." |
| Indirection ("what keeps…") | "what keeps an agent honest" | "predictable outcomes" |
| Nominalization | "salience decays across the session" | "models forget as the session grows" |
| Jargon as label | "append-only" | "never rewritten" |
| Software as bureaucrat | "Installation refuses to proceed" | "It won't install" |
| X-not-Y slogan tail | "Gates are code, not prose." | "Gates are code." |
| Colon agenda headline | "governance first, then execution, then context" | "three layers" |
| Poetic em-dash tagline | invented "X — where Y meets Z" lines | say what it does |

## What good copy sounds like

- Labels, not literary clauses. A section is "method", not a sentence about it.
- Plain verbs, first person where it is the speaker's own story:
  "I stopped enjoying my pile" beats "the pile ceased to serve".
- Numbers carry their meaning in the label: "209 — of them with zero human
  review", never a bare "209".
- Contrast is allowed when it is information, not rhythm: "The check runs
  on your machine, and again before anything merges."
- Questions are allowed as openers and act turns: "What if finishing was
  the default?"

## Mechanical final pass (do this — reading is not enough)

The banned patterns creep back in while you write. Before `deckgen check`,
search the visible slide text (not notes, not code comments) for each of
these and justify every survivor:

- `", not "` and `". Not "` — almost always an X-not-Y tail; keep the X.
- `" — "` — keep only when the clause after it adds information a reader
  needs; delete it when it adds rhythm.
- `", from the"`, `", in three"`, `", learned"` — appositive titles; cut
  the clause, keep the noun.
- Two consecutive short sentences with mirrored shape — keep one.

If a sentence sounds like a poster, rewrite it until it sounds like a
person answering a question.

## Quotes

Verbatim only, with attribution and a date. Trim with ellipses; never
rewrite. One quote per slide, nothing else competing with it.

## Humor

Allowed and encouraged, one beat per act: self-deprecating icebreakers
("How was your summer? Let me show you mine :)"), an infomercial teaser
("Oh, and that's not all."), a wry third label in a number trio. Never
sarcasm at the audience's expense; never a joke on a data slide.

## Narrative patterns

- Open with a question or a claim the closing slide answers.
- Signpost the journey once (an agenda slide), then never label a section
  again — no kickers, no footers, no page numbers, anywhere.
- Plant and pay off: name a mechanism early, point back when it saves the
  day ("the ratchet from act one, live").
- Journey stages carry rails: *enter:* what failure forces you in, *exit:*
  what tells you it is done (green italic / gray italic, bottom of slide).
- Be honest on status slides: "Status: this part doesn't fully work yet."
- Close by answering the opening, then a three-item, imperative,
  doable-this-week checklist.
