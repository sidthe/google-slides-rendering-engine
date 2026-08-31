// deck-engine HTML extractor. Runs inside the page; returns one scene per
// <section> — a flat, paint-ordered list of boxes, lines, text blocks and
// tables with computed geometry (CSS px, relative to the section origin)
// and computed styles. The Go side (mapscene.go) converts scenes to IR.
(() => {
  const warnings = [];

  const parseColor = (str) => {
    // computed colors are rgb(r, g, b) or rgba(r, g, b, a)
    const m = /rgba?\(([\d.]+),\s*([\d.]+),\s*([\d.]+)(?:,\s*([\d.]+))?\)/.exec(str || "");
    if (!m) return { hex: "", alpha: 0 };
    const hex = [m[1], m[2], m[3]]
      .map((v) => Math.round(+v).toString(16).padStart(2, "0").toUpperCase())
      .join("");
    return { hex, alpha: m[4] === undefined ? 1 : +m[4] };
  };

  const px = (v) => {
    const n = parseFloat(v);
    return isNaN(n) ? 0 : n;
  };

  const round = (v) => Math.round(v * 100) / 100;

  const SKIP_TAGS = new Set(["SCRIPT", "STYLE", "META", "LINK", "TITLE", "NOSCRIPT", "TEMPLATE", "HEAD", "BR"]);
  const MEDIA_TAGS = new Set(["IMG", "CANVAS", "VIDEO", "IFRAME", "SVG", "OBJECT", "EMBED", "AUDIO", "PICTURE"]);

  const isInline = (cs) => cs.display === "inline";

  const alignOf = (cs) => {
    switch (cs.textAlign) {
      case "center": return "ctr";
      case "right": case "end": return "r";
      case "justify": return "l";
      default: return "l";
    }
  };

  const spacingOf = (cs) => {
    if (cs.lineHeight === "normal") return 0;
    const lh = px(cs.lineHeight), fs = px(cs.fontSize);
    if (lh <= 0 || fs <= 0) return 0;
    const mult = lh / fs;
    return mult > 0.5 && mult < 4 ? round(mult) : 0;
  };

  const runStyle = (cs, link) => ({
    bold: parseInt(cs.fontWeight, 10) >= 600,
    italic: cs.fontStyle === "italic" || cs.fontStyle === "oblique",
    underline: cs.textDecorationLine.includes("underline"),
    strike: cs.textDecorationLine.includes("line-through"),
    color: parseColor(cs.color).hex,
    size: round(px(cs.fontSize)),
    font: cs.fontFamily,
    link: link || "",
  });

  const sameStyle = (a, b) =>
    a.bold === b.bold && a.italic === b.italic && a.color === b.color &&
    a.underline === b.underline && a.strike === b.strike &&
    a.size === b.size && a.font === b.font && a.link === b.link;

  // ---- CSS white-space processing model ----
  //
  // The extractor mirrors what Chrome paints, so white space is handled the
  // way CSS defines it rather than by collapsing everything. CSS splits the
  // behaviour into two switches, which Chrome exposes on the
  // white-space-collapse longhand; the shorthand is the fallback for older
  // builds. white-space inherits, so a <span> inside a <pre> reports
  // "preserve" with no special case.
  const WS_MODES = {
    "collapse":        { spaces: false, breaks: false },
    "preserve":        { spaces: true,  breaks: true  },
    "preserve-breaks": { spaces: false, breaks: true  },
    "preserve-spaces": { spaces: true,  breaks: false },
    "break-spaces":    { spaces: true,  breaks: true  },
  };
  const WS_SHORTHAND = {
    "normal": "collapse", "nowrap": "collapse",
    "pre": "preserve", "pre-wrap": "preserve",
    "pre-line": "preserve-breaks", "break-spaces": "break-spaces",
  };
  // Collapsible white space per CSS Text: space, tab, LF, CR, FF. U+00A0 is
  // deliberately absent — CSS never collapses it, though JS \s and trim() do.
  const WS_RUN = /[ \t\r\n\f]+/g;
  const wsOf = (cs) =>
    WS_MODES[cs.whiteSpaceCollapse] ||
    WS_MODES[WS_SHORTHAND[cs.whiteSpace]] ||
    WS_MODES.collapse;

  // hasContent: does a text node carry anything Chrome would paint? Under a
  // space-preserving parent every character counts; otherwise a character
  // outside the collapsible set is needed. NBSP counts either way.
  const hasContent = (text, ws) =>
    ws.spaces ? text.length > 0 : text.replace(WS_RUN, "").length > 0;

  // collapseParagraph applies the collapsing that cannot be done per text
  // node: Chrome collapses a space ending one run against a space starting
  // the next, and drops white space at both ends of the line. Runs that came
  // from a preserving context are left alone.
  const collapseParagraph = (runs) => {
    let prevSpace = true; // start of line: leading white space is dropped
    let seen = false;     // any non-space content yet?
    for (const r of runs) {
      if (!r.collapsible) { prevSpace = false; seen = true; continue; }
      if (prevSpace) r.text = r.text.replace(/^ +/, "");
      if (r.text.length) {
        prevSpace = r.text.endsWith(" ");
        if (r.text.replace(WS_RUN, "").length) seen = true;
      }
      if (!seen) prevSpace = true;
    }
    // trailing white space at the end of the line
    for (let i = runs.length - 1; i >= 0; i--) {
      const r = runs[i];
      if (!r.collapsible) break;
      r.text = r.text.replace(/ +$/, "");
      if (r.text.length) break;
    }
    return runs.filter((r) => r.text.length);
  };

  // collectParas gathers the inline content of a block element as
  // paragraphs of styled runs. Nested block elements are skipped (the main
  // walk emits them separately); <br> starts a new paragraph.
  const collectParas = (el, crossBlocks) => {
    const elCs = getComputedStyle(el);
    const elWs = wsOf(elCs);
    const paras = [];
    let runs = [];
    const flush = () => { paras.push({ runs }); runs = []; };
    const push = (text, style, ws) => {
      if (!text.length) return;
      const last = runs[runs.length - 1];
      if (last && last.collapsible === !ws.spaces && sameStyle(last.style, style)) last.text += text;
      else runs.push({ text, style, collapsible: !ws.spaces });
    };
    const add = (text, style, ws) => {
      if (!ws.breaks) {
        // segment breaks collapse into spaces alongside tabs and space runs
        push(text.replace(WS_RUN, " "), style, ws);
        return;
      }
      let t = text.replace(/\r\n?/g, "\n");
      if (!ws.spaces) {
        t = t.replace(/[ \t\f]+/g, " ").replace(/ *\n */g, "\n");
      }
      const lines = t.split("\n");
      for (let i = 0; i < lines.length; i++) {
        if (i > 0) flush();
        if (lines[i].length) push(lines[i], style, ws);
      }
    };
    const rec = (node, style, link, ws) => {
      for (const c of node.childNodes) {
        if (c.nodeType === Node.TEXT_NODE) { add(c.textContent, style, ws); continue; }
        if (c.nodeType !== Node.ELEMENT_NODE || SKIP_TAGS.has(c.tagName) && c.tagName !== "BR") continue;
        if (c.tagName === "BR") { flush(); continue; }
        if (MEDIA_TAGS.has(c.tagName)) continue;
        const ccs = getComputedStyle(c);
        if (ccs.display === "none" || ccs.visibility === "hidden") continue;
        if (!crossBlocks && !isInline(ccs)) continue; // nested block: emitted separately
        const clink = c.tagName === "A" && c.getAttribute("href") ? c.href : link;
        rec(c, runStyle(ccs, clink), clink, wsOf(ccs));
      }
    };
    rec(el, runStyle(elCs, ""), "", elWs);
    flush();
    for (const p of paras) {
      p.runs = collapseParagraph(p.runs);
      for (const r of p.runs) delete r.collapsible;
    }
    if (!elWs.breaks) return paras.filter((p) => p.runs.length);
    // Preserved breaks: blank lines are content and must survive. Chrome
    // removes a single segment break at the end of a block, so a pre block
    // whose text ends in a newline is not one line taller.
    if (paras.length > 1 && !paras[paras.length - 1].runs.length) paras.pop();
    return paras;
  };

  // hasDirectInlineText: does el own text that no nested block claims?
  const hasDirectInlineText = (el) => {
    const rec = (node, ws) => {
      for (const c of node.childNodes) {
        if (c.nodeType === Node.TEXT_NODE && hasContent(c.textContent, ws)) return true;
        if (c.nodeType !== Node.ELEMENT_NODE || SKIP_TAGS.has(c.tagName) || MEDIA_TAGS.has(c.tagName)) continue;
        const ccs = getComputedStyle(c);
        if (ccs.display === "none" || ccs.visibility === "hidden") continue;
        if (!isInline(ccs)) continue;
        if (rec(c, wsOf(ccs))) return true;
      }
      return false;
    };
    return rec(el, wsOf(getComputedStyle(el)));
  };

  // inlineTextRect returns the painted bounds of text owned by el: direct
  // text nodes plus inline descendants. A block child is emitted separately
  // by walk(), so it must not enlarge this frame. Using el's full box here
  // loses the text's CSS position when a block label precedes direct text
  // inside a card (for example, <b>Label</b>body), which makes native Slides
  // overlay the two text boxes.
  const inlineTextRect = (el) => {
    let left = Infinity, top = Infinity, right = -Infinity, bottom = -Infinity;
    const add = (rect) => {
      if (!rect || rect.width <= 0 || rect.height <= 0) return;
      left = Math.min(left, rect.left); top = Math.min(top, rect.top);
      right = Math.max(right, rect.right); bottom = Math.max(bottom, rect.bottom);
    };
    const addText = (node, ws) => {
      if (!hasContent(node.textContent, ws)) return;
      const range = document.createRange();
      range.selectNode(node);
      for (const rect of range.getClientRects()) add(rect);
    };
    const rec = (node, ws) => {
      for (const c of node.childNodes) {
        if (c.nodeType === Node.TEXT_NODE) { addText(c, ws); continue; }
        if (c.nodeType !== Node.ELEMENT_NODE || SKIP_TAGS.has(c.tagName) || MEDIA_TAGS.has(c.tagName)) continue;
        const ccs = getComputedStyle(c);
        if (ccs.display === "none" || ccs.visibility === "hidden" || !isInline(ccs)) continue;
        rec(c, wsOf(ccs));
      }
    };
    rec(el, wsOf(getComputedStyle(el)));
    return Number.isFinite(left) ? { left, top, right, bottom, width: right - left, height: bottom - top } : null;
  };

  const flatRuns = (paras) =>
    paras.map((p) => ({
      bullet: p.bullet || "",
      level: p.level || 0,
      runs: p.runs.map((r) => ({
        text: r.text,
        bold: r.style.bold,
        italic: r.style.italic,
        underline: r.style.underline,
        strike: r.style.strike,
        color: r.style.color,
        size: r.style.size,
        font: r.style.font,
        link: r.style.link,
      })),
    }));

  // extractListParas flattens a ul/ol (with nesting) into bullet
  // paragraphs; li content must be inline (nested lists recurse).
  const extractListParas = (listEl, level, out) => {
    const bullet = listEl.tagName === "OL" ? "number" : "disc";
    for (const li of listEl.children) {
      if (li.tagName !== "LI") continue;
      for (const p of collectParas(li, false)) out.push({ ...p, bullet, level });
      for (const sub of li.children) {
        if (sub.tagName === "UL" || sub.tagName === "OL") extractListParas(sub, level + 1, out);
      }
    }
  };

  const extract = (section) => {
    const sRect = section.getBoundingClientRect();
    const sceneWarn = [];
    const elements = [];

    const rel = (r) => ({
      x: round(r.left - sRect.left),
      y: round(r.top - sRect.top),
      w: round(r.width),
      h: round(r.height),
    });

    const describe = (el) => {
      let d = el.tagName.toLowerCase();
      if (el.id) d += "#" + el.id;
      else if (el.classList.length) d += "." + el.classList[0];
      return d;
    };

    const warn = (el, msg) => sceneWarn.push(describe(el) + ": " + msg);

    const boxOf = (el, cs) => {
      const r = rel(el.getBoundingClientRect());
      const bg = parseColor(cs.backgroundColor);
      const bw = px(cs.borderTopWidth);
      const bc = parseColor(cs.borderTopColor);
      if (cs.backgroundImage !== "none") warn(el, "background-image/gradient approximated as solid fill");
      if (cs.boxShadow !== "none") warn(el, "box-shadow dropped");
      if (cs.transform !== "none") warn(el, "transform: using transformed bounding box; rotation is only preserved on data-line and pure-text elements");
      // radius: px or % of the smaller side
      const minSide = Math.max(1, Math.min(r.w, r.h));
      let radius = 0;
      const rv = cs.borderTopLeftRadius.split(" ")[0];
      radius = rv.endsWith("%") ? px(rv) / 100 : px(rv) / minSide;
      radius = Math.min(0.5, Math.max(0, round(radius)));
      return {
        kind: "box",
        ...r,
        shape: el.getAttribute("data-shape") || "",
        fill: bg.alpha > 0 ? bg.hex : "",
        fillAlpha: bg.alpha > 0 && bg.alpha < 1 ? round(bg.alpha) : 0,
        border: bw > 0 && bc.alpha > 0 ? bc.hex : "",
        borderW: round(bw),
        dashed: cs.borderTopStyle === "dashed" || cs.borderTopStyle === "dotted",
        radius,
        oval: radius >= 0.49 && Math.abs(r.w - r.h) <= Math.max(r.w, r.h) * 0.25,
      };
    };

    // extractElbow: a data-elbow div draws an L with exactly one vertical
    // border (left/right) and one horizontal border (top/bottom). The path
    // runs from the free tip of the vertical segment, through the shared
    // corner, to the free tip of the horizontal segment.
    const extractElbow = (el, cs, r) => {
      const sides = {
        left: px(cs.borderLeftWidth), right: px(cs.borderRightWidth),
        top: px(cs.borderTopWidth), bottom: px(cs.borderBottomWidth),
      };
      const v = sides.left > 0 ? "left" : (sides.right > 0 ? "right" : "");
      const hz = sides.top > 0 ? "top" : (sides.bottom > 0 ? "bottom" : "");
      if (!v || !hz) {
        warn(el, "data-elbow needs one vertical (left/right) and one horizontal (top/bottom) border");
        return;
      }
      const vx = v === "left" ? r.x : r.x + r.w;
      const hy = hz === "top" ? r.y : r.y + r.h;
      const startY = hz === "top" ? r.y + r.h : r.y;
      const endX = v === "left" ? r.x + r.w : r.x;
      const bcol = parseColor(v === "left" ? cs.borderLeftColor : cs.borderRightColor);
      const bstyle = v === "left" ? cs.borderLeftStyle : cs.borderRightStyle;
      const arrowAttr = el.hasAttribute("data-arrow") ? (el.getAttribute("data-arrow") || "end") : "";
      elements.push({
        kind: "elbow",
        x1: vx, y1: startY, x2: endX, y2: hy,
        color: bcol.hex,
        widthPx: round(Math.max(sides[v], 1)),
        dashed: el.hasAttribute("data-dashed") || bstyle === "dashed" || bstyle === "dotted",
        arrow: arrowAttr === "end" || arrowAttr === "both",
        arrowStart: arrowAttr === "start" || arrowAttr === "both",
      });
    };

    // pureRotation returns the clockwise angle in degrees when the
    // element's transform is rotation-only (no skew/scale), else null.
    const pureRotation = (cs) => {
      if (cs.transform === "none") return null;
      const m = /matrix\(([-\d.e]+),\s*([-\d.e]+),\s*([-\d.e]+),\s*([-\d.e]+),\s*([-\d.e]+),\s*([-\d.e]+)\)/.exec(cs.transform);
      if (!m) return null;
      const [a, b, c, d] = m.slice(1, 5).map(Number);
      if (Math.abs(a - d) > 0.001 || Math.abs(b + c) > 0.001 || Math.abs(a * d - b * c - 1) > 0.01) return null;
      return round(Math.atan2(b, a) * 180 / Math.PI);
    };

    // lineEndpoints returns the element's midline in viewport coords,
    // honoring CSS transforms so rotated divs become true diagonal lines:
    // measure untransformed, then map both endpoints through the computed
    // matrix around the transform origin.
    const lineEndpoints = (el, cs) => {
      const hasTransform = cs.transform !== "none";
      let base;
      if (hasTransform) {
        const saved = el.style.transform;
        el.style.transform = "none";
        base = el.getBoundingClientRect();
        el.style.transform = saved;
      } else {
        base = el.getBoundingClientRect();
      }
      const horizontal = base.width >= base.height;
      const thick = Math.min(base.width, base.height);
      let p1 = horizontal
        ? { x: base.left, y: base.top + base.height / 2 }
        : { x: base.left + base.width / 2, y: base.top };
      let p2 = horizontal
        ? { x: base.right, y: base.top + base.height / 2 }
        : { x: base.left + base.width / 2, y: base.bottom };
      if (hasTransform) {
        const m = /matrix\(([-\d.e]+),\s*([-\d.e]+),\s*([-\d.e]+),\s*([-\d.e]+),\s*([-\d.e]+),\s*([-\d.e]+)\)/.exec(cs.transform);
        if (m) {
          const [a, b, c, d, e, f] = m.slice(1).map(Number);
          const [ox, oy] = cs.transformOrigin.split(" ").map(px);
          const O = { x: base.left + ox, y: base.top + oy };
          const map = (p) => ({
            x: O.x + a * (p.x - O.x) + c * (p.y - O.y) + e,
            y: O.y + b * (p.x - O.x) + d * (p.y - O.y) + f,
          });
          p1 = map(p1);
          p2 = map(p2);
        } else {
          warn(el, "non-2D transform on line element; endpoints approximate");
        }
      }
      return { p1, p2, thick };
    };

    const emitLine = (el, cs) => {
      const bg = parseColor(cs.backgroundColor);
      const bc = parseColor(cs.borderTopColor);
      const color = bg.alpha > 0 ? bg.hex : (bc.alpha > 0 ? bc.hex : "");
      if (!color) return;
      const { p1, p2, thick } = lineEndpoints(el, cs);
      // data-arrow: "" or "end" puts the head at the geometric end
      // (right/bottom), "start" at the start, "both" at both ends.
      const arrowAttr = el.hasAttribute("data-arrow") ? (el.getAttribute("data-arrow") || "end") : "";
      elements.push({
        kind: "line",
        x1: round(p1.x - sRect.left), y1: round(p1.y - sRect.top),
        x2: round(p2.x - sRect.left), y2: round(p2.y - sRect.top),
        color,
        widthPx: round(Math.max(thick, 1)),
        dashed: el.hasAttribute("data-dashed") || cs.borderTopStyle === "dashed" || cs.borderTopStyle === "dotted",
        arrow: arrowAttr === "end" || arrowAttr === "both",
        arrowStart: arrowAttr === "start" || arrowAttr === "both",
      });
    };

    const extractTable = (tbl) => {
      const rows = [...tbl.rows];
      if (!rows.length || !rows[0].cells.length) return;
      const r = rel(tbl.getBoundingClientRect());
      const colW = [...rows[0].cells].map((c) => round(c.getBoundingClientRect().width));
      const rowH = round(r.h / rows.length);
      const cellBg = (td) => {
        for (let n = td; n && n !== tbl.parentElement; n = n.parentElement) {
          const c = parseColor(getComputedStyle(n).backgroundColor);
          if (c.alpha > 0) return c.hex;
        }
        return "";
      };
      const first = rows[rows.length - 1].cells[0];
      const bb = parseColor(getComputedStyle(first).borderBottomColor);
      const fcs = getComputedStyle(rows[0].cells[0]);
      elements.push({
        kind: "table",
        ...r,
        colW,
        rowH,
        border: px(getComputedStyle(first).borderBottomWidth) > 0 && bb.alpha > 0 ? bb.hex : "",
        defSize: round(px(fcs.fontSize)),
        rows: rows.map((tr) => [...tr.cells].map((td) => ({
          fill: cellBg(td),
          align: alignOf(getComputedStyle(td)),
          paras: flatRuns(collectParas(td, true)),
        }))),
      });
    };

    const walk = (el) => {
      if (SKIP_TAGS.has(el.tagName)) return;
      if (el.tagName === "ASIDE" && (el.classList.contains("notes") || el.hasAttribute("data-notes"))) return; // handled per-section
      const cs = getComputedStyle(el);
      if (cs.display === "none" || cs.visibility === "hidden" || px(cs.opacity) === 0) return;
      if (MEDIA_TAGS.has(el.tagName)) {
        warn(el, "media element skipped (output must stay natively editable)");
        return;
      }
      if (el.tagName === "TABLE") {
        extractTable(el);
        return;
      }
      const r = rel(el.getBoundingClientRect());
      if (el.hasAttribute("data-elbow")) {
        extractElbow(el, cs, r);
        return;
      }
      if (el.tagName === "UL" || el.tagName === "OL") {
        const listParas = [];
        extractListParas(el, 0, listParas);
        if (listParas.length) {
          const li = el.querySelector("li");
          const lcs = li ? getComputedStyle(li) : cs;
          elements.push({
            kind: "text",
            ...r,
            align: alignOf(lcs),
            spacing: spacingOf(lcs),
            fontSize: round(px(lcs.fontSize)),
            fontColor: parseColor(lcs.color).hex,
            fontFam: lcs.fontFamily,
            paras: flatRuns(listParas),
          });
        }
        return;
      }
      if (el.tagName === "HR" || el.hasAttribute("data-line") ||
          (cs.transform === "none" && r.w > 0 && r.h > 0 && Math.min(r.w, r.h) <= 3 &&
           !hasDirectInlineText(el) && (parseColor(cs.backgroundColor).alpha > 0))) {
        emitLine(el, cs);
        return;
      }
      // Rotated text (axis titles): pure-rotation transform on an element
      // that only carries text. Measured untransformed; both backends
      // rotate about the box center, matching the CSS default origin.
      const rot = pureRotation(cs);
      if (rot !== null && hasDirectInlineText(el)) {
        const saved = el.style.transform;
        el.style.transform = "none";
        const base = rel(el.getBoundingClientRect());
        el.style.transform = saved;
        if (parseColor(cs.backgroundColor).alpha > 0 || px(cs.borderTopWidth) > 0) {
          warn(el, "background/border on rotated element dropped; only the text rotates");
        }
        const paras = collectParas(el, false);
        if (paras.length) {
          elements.push({
            kind: "text",
            ...base,
            rot,
            align: alignOf(cs),
            spacing: spacingOf(cs),
            fontSize: round(px(cs.fontSize)),
            fontColor: parseColor(cs.color).hex,
            fontFam: cs.fontFamily,
            paras: flatRuns(paras),
          });
        }
        return;
      }
      if (el !== section && r.w > 0 && r.h > 0) {
        const b = boxOf(el, cs);
        if (b.fill || b.border) elements.push(b);
        if (hasDirectInlineText(el)) {
          const paras = collectParas(el, false);
          const textRect = inlineTextRect(el);
          if (paras.length) {
            elements.push({
              kind: "text",
              ...(textRect ? rel(textRect) : r),
              align: alignOf(cs),
              spacing: spacingOf(cs),
              fontSize: round(px(cs.fontSize)),
              fontColor: parseColor(cs.color).hex,
              fontFam: cs.fontFamily,
              paras: flatRuns(paras),
            });
          }
        }
      }
      // Inline descendants are already preserved as rich runs in their
      // parent's text box. Walking them again duplicates text in native
      // output; block descendants retain independent CSS geometry.
      for (const c of el.children) {
        if (!isInline(getComputedStyle(c))) walk(c);
      }
    };

    for (const c of section.children) walk(c);
    // speaker notes: hidden aside, one paragraph per block child
    let notes = "";
    const aside = section.querySelector("aside.notes, aside[data-notes]");
    if (aside) {
      const lines = [];
      for (const c of aside.children) {
        const t = c.textContent.replace(/\s+/g, " ").trim();
        if (t) lines.push(t);
      }
      notes = lines.length ? lines.join("\n") : aside.textContent.replace(/\s+/g, " ").trim();
    }
    // background of the section itself
    const scs = getComputedStyle(section);
    const sbg = parseColor(scs.backgroundColor);
    return {
      w: round(sRect.width),
      h: round(sRect.height),
      background: sbg.alpha > 0 ? sbg.hex : "",
      notes,
      elements,
      warnings: sceneWarn,
    };
  };

  const sections = [...document.querySelectorAll("section")];
  if (!sections.length) warnings.push("no <section> elements found — each slide must be a <section>");
  return {
    scenes: sections.map(extract),
    warnings,
  };
})()
