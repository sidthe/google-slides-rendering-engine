---
name: gslides-renderer
description: >-
  Turn-based presentation generation and rendering system using google-slides-rendering-engine.
  Enforces a strict 4-phase workflow: audience introspection & outline alignment,
  Google-branded HTML deck authoring (1600x900 native shapes), visual screenshot
  inspection quality gate, native PPTX compilation via deckgen for Google Slides import,
  and maintaining live workspace presentation mapping (gslides-map.json).
---

# Google Slides Rendering Engine (`gslides-renderer` / `deckgen`)

> [!IMPORTANT]
> **Core Non-Negotiable Directives:**
> 1. **Exclusive Rendering Tool**: You MUST use `/usr/local/google/home/vsz/Projects/tools/google-slides-rendering-engine/` (`deckgen build deck.html -o deck.pptx`) for PPTX compilation. NEVER use any third-party PPTX generation libraries (such as `python-pptx`, Pandoc, Marp, pptxgenjs, etc.). All presentations must flow through the HTML -> `deckgen` pipeline.
> 2. **Zero Image Rasterization**: Output presentations are 100% native vector objects. Every box becomes a shape, every text a text box, every connector a native vector line, and every table a native table. Media tags (`<img>`, `<svg>`, `<canvas>`, `<video>`) are strictly forbidden.
> 3. **Turn-Based Execution**: You must not generate HTML until the slide outline is confirmed by the user.
> 4. **Live Presentation Mapping**: Once a slide deck is published or imported to Google Slides, maintain a mapping configuration file (`gslides-map.json`) in the workspace root linking local files (`deck.html`, `deck.pptx`) to the live Google Slides presentation.
> 5. **Strict Code & Manifest Indentation**: All YAML, JSON, code, and XML blocks must preserve standard hierarchical indentation (e.g. 2 spaces per nested dictionary/list level in YAML). Never flatten, strip, or collapse indentation in code presentations.

---

## 1. Mandatory Turn-Based Workflow

Presentation generation proceeds through explicit phases:

```
┌──────────────────────────┐      ┌──────────────────────────┐
│ Phase 1: Audience & Plan │ ───▶ │ Phase 2: HTML Generation │
│ Introspect + Outline     │      │ Google Brand Style       │
│ [STOP for confirmation]  │      │ 1 Idea Per Slide         │
└──────────────────────────┘      └──────────────────────────┘
             │                                  │
             ▼                                  ▼
┌──────────────────────────┐      ┌──────────────────────────┐
│ Phase 4: Native Render   │ ◀─── │ Phase 3: Screenshot Gate │
│ deckgen build -> PPTX    │      │ deckgen screenshot       │
│ Import & Live Map Record │      │ Inspect & Fix Overflows  │
└──────────────────────────┘      └──────────────────────────┘
```

### Phase 1: Audience Introspection & Outline Alignment (Turn 1)
- **Deep Audience Introspection**: Before writing any slide content or code, analyze how the presentation delivers value across the complete reader spectrum:
  - **Platform Admins / Engineers**: Technical correctness, architecture precision, failure modes, metrics, interfaces, and concrete workflows.
  - **Engineering Leads / Architects**: System trade-offs, scalability boundaries, operational complexity, and integration surfaces.
  - **Executives / Leadership**: Strategic thesis, business impact, latency/cost ROI, milestone roadmap, and clear takeaways.
- **Slide-by-Slide Outline**: Propose a structured outline containing for each slide:
  - Slide number and kicker label (e.g., `SYSTEM ARCHITECTURE`, `PERFORMANCE & LATENCY`).
  - Clear slide title delivering one key thesis.
  - Visual archetype (e.g., Sequence Diagram, 3-Stat Metric Grid, 2-Column Comparison, Architecture Flow).
  - Core takeaways and talking points.
- **Gate**: Stop and present the outline to the user. Await explicit confirmation before proceeding to HTML generation.

### Phase 2: High-Fidelity HTML Authoring (Turn 2)
- Upon user confirmation, create the HTML deck file (`deck.html`).
- **Google Brand Styling**: Use clean typography, authentic Google colors, subtle cards, and structured layouts using `/usr/local/google/home/vsz/Projects/tools/google-slides-rendering-engine/assets/deck.css`.
- **One Key Idea Per Slide**: Maintain high visual discipline. Never overcrowd slides (≤3 metrics or ≤5 content blocks per slide).
- **Native Diagrams & Visuals**:
  - Architecture diagrams: Built from styled `.box` and `.card` divs, group boundaries (`border: 2px dashed`), and connected via `data-line` or `data-elbow`.
  - Flowcharts & Process flows: Built using preset `data-shape` geometry (`diamond`, `cylinder`/`db`, `pentagon`, `chevron`, `trapezoid`).
  - Charts: Drawn natively from positioned flex/grid div bars over gridlines.
  - Code & Manifests: Rendered with proper monospace font (`var(--mono)`), dark background (`.code` or `.dark`), and explicit 2-space hierarchical indentation.

### Phase 3: Visual Review & Screenshot Quality Gate (Turn 3)
- Validate syntax and native compatibility:
  ```bash
  deckgen check deck.html
  ```
  Fix all conversion warnings.
- Render per-slide PNG screenshots via headless Chrome:
  ```bash
  deckgen screenshot deck.html -o shots/
  ```
- **Mandatory Self-Review Checklist**: View and inspect every generated `shots/slideNN.png` file before presenting the output:
  1. **Narrative & Hierarchy**: Does the slide immediately communicate its core point?
  2. **Overflow & Clipping**: Are any text runs wrapping unexpectedly or overflowing containers? (Google Slides wraps slightly tighter than Chrome; maintain 10–15% width slack).
  3. **Code Indentation**: Are YAML, JSON, and code blocks strictly indented to reflect their nested syntax?
  4. **Alignment & Spacing**: Are cards, gridlines, connectors, and labels aligned on standard grid coordinates?
  5. **Branding & Palette**: Are all colors sourced from `assets/deck.css` variables? Are fonts strictly Roboto and Roboto Mono?
- If any visual defects, misalignments, or awkward line wraps are discovered, edit `deck.html` and re-run the screenshot loop. Do not declare the deck done until visual quality is verified.

### Phase 4: Native PPTX Compilation & Live Mapping (Turn 4)
- Compile the validated HTML into a native Google Slides-compatible PPTX using the local engine:
  ```bash
  deckgen build deck.html -o deck.pptx
  ```
- Upon publishing or importing the presentation into Google Slides, record and maintain the mapping configuration in `gslides-map.json` in the workspace root.

---

## 2. Live Presentation Mapping Configuration (`gslides-map.json`)

To keep local slide definitions synchronized with published cloud presentations, maintain `gslides-map.json` in the workspace root following the document mapping format:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "workspace": "/usr/local/google/home/vsz/Projects/decks",
  "description": "Mapping of local workspace deck presentations to published live Google Slides.",
  "presentations": [
    {
      "local_path": "CapacityQuota-deck/deck.pptx",
      "html_path": "CapacityQuota-deck/deck.html",
      "presentation_id": "1m9GSyxrzLoPSTEG-Zt8ndCOoaiCOexFPhLkPyMUZNeo",
      "url": "https://docs.google.com/presentation/d/1m9GSyxrzLoPSTEG-Zt8ndCOoaiCOexFPhLkPyMUZNeo/edit",
      "title": "GKE Cluster Autoscaler - CapacityQuota",
      "published_at": "2026-08-28T20:12:54Z",
      "sync_enabled": true
    }
  ]
}
```

---

## 3. CLI Quick Reference

The `deckgen` and `gslides-renderer` binaries are installed in `~/.local/bin/`:

```bash
# Validate HTML structure and check for unconvertible CSS/tags
deckgen check deck.html

# Render Chrome PNG screenshots into shots/ for visual quality review
deckgen screenshot deck.html -o shots/

# Compile HTML to Google Slides-tuned native PPTX (primary deliverable)
deckgen build deck.html -o deck.pptx

# Optional: Push directly to Google Slides via API (if credentials configured)
deckgen push deck.html -title "Deck Title"

# Optional: Upload & convert via Drive API for maximum fidelity
deckgen import deck.html -title "Deck Title"

# Optional: Verify Google Slides cloud rendering
deckgen snap <PRESENTATION_ID> -o shots-slides/
```

---

## 4. Code & Manifest Indentation Rules

> [!IMPORTANT]
> **Enforcing Code Indentation Quality:**
> 1. **YAML Hierarchy**: Every nested YAML key must indent by exactly 2 spaces (`&nbsp;&nbsp;` or standard preformatted whitespace). Root level keys (`apiVersion`, `kind`, `metadata`, `spec`) are at column 0. Child objects (`spec.selector`, `spec.limits`) at 2 spaces; sub-mappings (`matchLabels`, `resources`) at 4 spaces; leaf key-values at 6 spaces.
> 2. **Code & JSON Blocks**: Python, Go, Bash, and JSON blocks must preserve 2-space or 4-space indentation. Never allow code lines to collapse flush-left against the container margin.
> 3. **Syntax Coloring**: Use VS Code dark theme palette spans for syntax readability (`#569CD6` for keywords/keys, `#9CDCFE` for attributes/variables, `#CE9178` for strings, `#6A9955` for comments, `#B5CEA8` for numbers).

```html
<!-- Example: Correctly Indented YAML Code Block in deck.html -->
<div class="code" style="left:80px; top:180px; width:700px; height:580px; font-size:18px; line-height:1.5;">
  <span style="color:#6A9955;"># autoscaling.x-k8s.io/v1beta1</span><br>
  <span style="color:#569CD6;">apiVersion</span>: autoscaling.x-k8s.io/v1beta1<br>
  <span style="color:#569CD6;">kind</span>: CapacityQuota<br>
  <span style="color:#569CD6;">metadata</span>:<br>
  &nbsp;&nbsp;<span style="color:#9CDCFE;">name</span>: n4-commitment-cap<br>
  <span style="color:#569CD6;">spec</span>:<br>
  &nbsp;&nbsp;<span style="color:#9CDCFE;">selector</span>:<br>
  &nbsp;&nbsp;&nbsp;&nbsp;<span style="color:#9CDCFE;">matchLabels</span>:<br>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span style="color:#CE9178;">cloud.google.com/compute-class</span>: prefer-n4<br>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span style="color:#CE9178;">cloud.google.com/machine-family</span>: n4<br>
  &nbsp;&nbsp;<span style="color:#9CDCFE;">limits</span>:<br>
  &nbsp;&nbsp;&nbsp;&nbsp;<span style="color:#9CDCFE;">resources</span>:<br>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span style="color:#9CDCFE;">cpu</span>: 100<br>
  &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span style="color:#9CDCFE;">memory</span>: 400Gi
</div>
```

---

## 5. Slide Authoring Contract

### Slide Template Structure

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<link rel="stylesheet" href="/usr/local/google/home/vsz/Projects/tools/google-slides-rendering-engine/assets/deck.css">
</head>
<body>

<!-- Slide 1: Title Slide -->
<section>
  <div class="accent-dots" style="left:80px; top:96px;"></div>
  <div class="title-hero" style="left:80px; top:292px; width:1440px;">Direct Node Provisioning</div>
  <div class="subtitle" style="left:80px; top:420px; width:1440px; color:var(--ink-2);">
    Sub-Second GKE Node Scale for High-Frequency AI Inference
  </div>
  <div class="pill" style="left:80px; top:520px;">Architecture & Implementation Strategy</div>
  <div class="foot"><span>Google Cloud | Internal</span><span>1</span></div>
</section>

<!-- Slide 2: Architecture / Content Slide -->
<section>
  <div class="kicker">SYSTEM TOPOLOGY</div>
  <div class="title">Bypassing Managed Instance Group Queues</div>
  
  <!-- Content Area: y=180 to y=800, x=80 to x=1520 -->
  <div class="card" style="left:80px; top:180px; width:680px; height:580px;">
    <div class="card-title">Legacy MIG Path</div>
    <div class="stat" style="color:var(--red);">45–90s</div>
    <div class="stat-label">Provisioning delay under quota pressure</div>
    <ul>
      <li>Centralized instance group manager arbitration</li>
      <li>Serial health checking and registration handshake</li>
    </ul>
  </div>

  <div class="card" style="left:800px; top:180px; width:680px; height:580px;">
    <div class="card-title">Direct Provisioning Path</div>
    <div class="stat" style="color:var(--green);">4.2s</div>
    <div class="stat-label">p99 direct RPC to TPU hypervisor slice</div>
    <ul>
      <li>Pre-warmed slice descriptor injection</li>
      <li>Bypasses MIG task serialization loops</li>
    </ul>
  </div>

  <aside class="notes">
    <p>Highlight the 10x p99 latency reduction under peak multi-tenant load.</p>
  </aside>
  <div class="foot"><span>Google Cloud | Internal</span><span>2</span></div>
</section>

</body>
</html>
```

### Canvas Coordinate Grid
- **Slide Dimensions**: Exactly `1600 × 900 px` per `<section>`.
- **Horizontal Boundaries**: Left `x = 80px`, Right `x = 1520px` (Width: `1440px`).
- **Vertical Landmarks**:
  - Kicker: `top: 52px` (uppercase tracking, `font-size: 17px`).
  - Slide Title: `top: 82px` (`font-size: 52px`, `line-height: 1.15`).
  - Content Canvas: `top: 180px` through `top: 800px` (`height: 620px`).
  - Footer: `top: 862px` (`font-size: 16px`, color `var(--ink-3)`).

---

## 6. Native Element Construction Rules

| Element | HTML Syntax | Native Deck Output |
| :--- | :--- | :--- |
| **Card / Box** | `<div class="card" style="...">` | Native rectangle shape with background, border, and rounding. |
| **Pill / Badge** | `<div class="pill">` or `border-radius: 20px;` | Native rounded rectangle with centered text. |
| **Circle / Node** | `<div style="width:60px; height:60px; border-radius:50%;">` | Native ellipse shape. |
| **Preset Shapes** | `<div data-shape="diamond\|cylinder\|db\|hexagon\|chevron\|pentagon\|trapezoid\|triangle\|callout">` | Native preset presentation shape. |
| **Straight Arrow** | `<div data-line data-arrow style="left:..; top:..; width:..; height:2px;">` | Native vector line with arrowhead. |
| **Diagonal Connector** | `<div data-line data-arrow style="width:LEN; transform:rotate(Adeg); transform-origin:0 50%;">` | Native rotated vector connector (`LEN = √(dx²+dy²)`, `A = atan2(dy,dx)`). |
| **Elbow (L-Shape)** | `<div data-elbow data-arrow style="border-left:2px solid; border-bottom:2px solid;">` | Native L-shaped vector connector. |
| **Group Boundary** | `<div style="border: 2px dashed var(--line); ...">` | Native dashed grouping container. |
| **Lists** | `<ul><li>First</li><li>Second</li></ul>` | Native bulleted text box. |
| **Data Table** | `<table><thead>...</thead><tbody>...</tbody></table>` | Native table with per-cell styling and alignments. |
| **Speaker Notes** | `<aside class="notes"><p>Talking points</p></aside>` | Native slide notes. |

---

## 7. Google Brand Style & Typography Guidelines

1. **Color Palette (`assets/deck.css`)**:
   - **Neutrals**: Canvas background `#ffffff`, Primary text `--ink` (`#202124`), Secondary text `--ink-2` (`#5f6368`), Subdued/Muted `--ink-3` (`#80868b`), Borders `--line` (`#dadce0`), Card backgrounds `--surface` (`#f8f9fa` or `#ffffff`).
   - **Google Blue**: Primary accent `--blue` (`#1a73e8`), Dark accent `--blue-dark` (`#174ea6`), Soft tint `--blue-tint` (`#e8f0fe`).
   - **Semantic Trios**:
     - Green: `--green` (`#1e8e3e`), `--green-dark` (`#137333`), `--green-tint` (`#e6f4ea`).
     - Red: `--red` (`#d93025`), `--red-dark` (`#c5221f`), `--red-tint` (`#fce8e6`).
     - Yellow: `--yellow` (`#f9ab00`), `--yellow-dark` (`#ea8600`), `--yellow-tint` (`#fef7e0`).
2. **Typography Rules**:
   - Limit to standard Google fonts available natively in Google Slides: `var(--font)` (Roboto) and `var(--mono)` (Roboto Mono).
   - Never introduce unsupported external web fonts (e.g. Google Sans, Inter, Helvetica Neue) which silently fall back and break alignment.
   - Text sizes: Hero numbers `66–90px`, Titles `52px`, Subtitles `28–30px`, Card headers `23px`, Body `20–23px`, Small labels `17px`, Captions/Ticks `16px`. Never go below `15px`.
3. **Editable Diagrams & Charts**:
   - Build sequence and topology diagrams using native `div` containers and vector lines.
   - Build bar charts, stacked progress bars, and waterfall distributions using styled HTML blocks over hairline gridlines.
   - Label values directly above bars so they remain fully editable in Google Slides.
