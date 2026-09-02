# Example Prompts for Deck Generation

This directory contains reference prompt templates for instructing AI coding agents (such as Jetski, Claude Code, Cursor, or Codex) to author high-fidelity, natively editable presentations using `deckgen` and `sidthe/google-slides-rendering-engine`.

---

## The Agent Prompting Pattern

High-quality presentations require a **two-phase verification workflow**:
1. **Alignment & Outline Phase:** The agent must introspect the source material and produce a structured outline covering audience empathy, narrative arc, and slide-by-slide proposals *before* writing HTML.
2. **Visual Verification & Build Phase:** The agent authors HTML (`<section>` slides at 1600×900 px), runs `deckgen check` to resolve warnings and lint rules, takes screenshots with `deckgen screenshot`, visually inspects every rendered slide for layout flaws or text wrapping, and compiles the final `.pptx` or Google Slides deck.

---

## Example 1: Executive Engineering & Architecture Strategy Deck

Use this prompt pattern when converting complex technical design docs, RFCs, or architecture proposals into an executive-level decision presentation.

```markdown
Scan <LINK_TO_DESIGN_DOC_OR_RFC> and shape a high-fidelity, executive engineering strategy deck. The objective is to secure leadership alignment and resolve core architectural decision gates.

Target Audience: Executive Engineering Leadership (Directors, VPs, Principal Engineers).

### Instructions:
1. **Introspection & Outline First:** Before authoring any slides, deeply introspect on the narrative and target audience. Provide a structured outline covering:
   - **Current State:** What exists today, proven capabilities, operational scale, and customer traction.
   - **Scale Bottlenecks:** Why the current tactical design hits a hard production ceiling (e.g., single identity, lack of tenant isolation, admission vs. authority conflation).
   - **Four Scale Principles:** Agnostic runtimes, pluggable toolchains, deep platform specialization, and right-to-left interventions.
   - **North Star Architecture:** Distributed Hub-and-Spoke topology, durable message fabric (e.g., NATS + A2A schemas), out-of-process credential brokering, and attenuating capability envelopes.
   - **Data Governance & Write Path:** Egress privacy lattice math, intelligent private fallbacks, propose-only GitOps PRs, and localized scheduling.
   - **Execution Roadmap & Decision Gates:** 3-month phased delivery schedule and blocking leadership product gates.

2. **Wait for Outline Confirmation:** Do not generate HTML until I confirm the outline.

3. **HTML Deck Generation:** 
   - Follow the repository's presentation contract (1600×900 px canvas per `<section>`).
   - Use Google brand design tokens (`assets/deck.css` or `templates/keynote/deck.css`).
   - Deliver exactly one core idea per slide.
   - Draw all diagrams, flowcharts, timelines, and terminals natively with styled divs (no image screenshots, no external charting libraries).
   - Include complete, speakable talking points in `<aside class="notes">` on every slide.

4. **Visual Quality & Verification Loop:**
   - Run `deckgen check deck.html` (or with `-lint keynote`) and ensure 0 warnings.
   - Run `deckgen screenshot deck.html -o shots/` and review every generated PNG.
   - Fix all text overflows, misalignments, or awkward line wraps.
   - Do not claim completion until visual fidelity is verified.

5. **Export:**
   - Use `deckgen build deck.html -o deck.pptx` or `deckgen push deck.html -title "<TITLE>"` to publish.
```

---

## Example 2: Go-To-Market (GTM) & Technical Product Deck

Use this prompt pattern when introducing a new open-source project, developer tool, or platform feature to both platform administrators and executive stakeholders.

```markdown
Scan https://github.com/gke-labs/workload-class and shape a high-fidelity, go-to-market (GTM) and engineering deck explaining key concepts on workload classes.

Start by creating an outline that leads with the value proposition and use cases, and continues with detailed technical insights.

### Instructions:
1. **Deep Introspection & Multi-Persona Outline:**
   - Before generating any deck, deeply introspect on how to deliver value to readers spanning from hands-on platform administrators to technology executives.
   - Provide an outline that balances business value (cost optimization, simplified developer experience, reliability guardrails) with deep technical mechanics (custom resource definitions, mutating webhooks, scheduler integration).
   - Only after I review and confirm the outline will you generate the HTML deck.

2. **Visual & Structural Contract:**
   - The HTML deck must be in Google brand style, high fidelity, with native architecture diagrams, flow diagrams, and comparisons.
   - Tastefully structured, delivering one key idea per slide with clear visual hierarchy.
   - No rasterized images or embedded external charts; build all cards, tables, and flows using CSS shapes.
   - Provide concise, professional speaker notes on every slide.

3. **Self-Verification & Iteration:**
   - Before offering the deck for review, screenshot each slide using `deckgen screenshot`.
   - Inspect and verify structure, layouts, branding, narrative pacing, and text wrapping.
   - Ensure zero warnings with `deckgen check`. You must not call the task done until the deck is verified at high quality.

4. **Native Rendering:**
   - Use `deckgen build deck.html -o deck.pptx` (via sidthe/google-slides-rendering-engine) to render the HTML into native PowerPoint / Google Slides format.
   - Do not use third-party PPTX generator libraries.
```

---

## Example 3: Minimal Speaker Keynote Prompt

Use this prompt pattern when preparing a keynote talk where the speaker is the focus and slides act as punchy backdrops.

```markdown
Author a keynote presentation based on <SOURCE_CONTENT> using the `keynote` template (`templates/keynote/`).

### Requirements:
- **Tone & Voice:** Strict adherence to `templates/keynote/VOICE.md` (no paired aphorisms, no antithesis slogans, ≤ 70 words per slide, hard 18px minimum font size).
- **Archetype Fidelity:** Map every slide to a catalog archetype from `templates/keynote/DESIGN.md` (e.g., A1 Title, A2 Statement, A5 Number Trio, A16 Terminal, A6 Chevron Flow, A8 Decision Flow, A14 Minimal Table, A23 Checklist Close).
- **Zero Chrome:** No kickers, footers, page numbers, or section labels.
- **Verification:** Validate with `deckgen check deck.html -lint keynote` to achieve 0 warnings and 0 lint findings, verify screenshots, and compile with `deckgen build`.
```
