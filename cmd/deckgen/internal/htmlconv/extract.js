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
    letterSpacing: cs.letterSpacing === "normal" ? 0 : round(px(cs.letterSpacing)),
    textTransform: cs.textTransform || "none",
  });

  const sameStyle = (a, b) =>
    a.bold === b.bold && a.italic === b.italic && a.color === b.color &&
    a.underline === b.underline && a.strike === b.strike &&
    a.size === b.size && a.font === b.font && a.link === b.link &&
    a.letterSpacing === b.letterSpacing && a.textTransform === b.textTransform;

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

  // applyTextTransform mirrors CSS text-transform across a paragraph's runs so
  // exported characters match the glyphs Chrome measured and painted.
  const WORD_CHAR = /[\p{L}\p{N}'\u2019]/u;
  const applyTextTransform = (runs) => {
    let wordStart = true;
    for (const r of runs) {
      const mode = r.style.textTransform;
      if (mode === "uppercase") {
        r.text = r.text.toUpperCase();
        if (r.text.length) wordStart = !WORD_CHAR.test([...r.text].pop());
      } else if (mode === "lowercase") {
        r.text = r.text.toLowerCase();
        if (r.text.length) wordStart = !WORD_CHAR.test([...r.text].pop());
      } else if (mode === "capitalize") {
        let out = "";
        for (const ch of r.text) {
          if (WORD_CHAR.test(ch)) {
            out += wordStart ? ch.toUpperCase() : ch;
            wordStart = false;
          } else {
            out += ch;
            wordStart = true;
          }
        }
        r.text = out;
      } else if (r.text.length) {
        wordStart = !WORD_CHAR.test([...r.text].pop());
      }
    }
  };

  // collectParas gathers the inline content of a block element as
  // paragraphs of styled runs. Nested block elements are skipped unless
  // crossBlocks is true; skipLists omits nested <ul>/<ol> when collecting <li>.
  const collectParas = (el, crossBlocks, skipLists) => {
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
        if (skipLists && (c.tagName === "UL" || c.tagName === "OL")) continue;
        if (c.tagName === "BR") { flush(); continue; }
        if (MEDIA_TAGS.has(c.tagName)) continue;
        const ccs = getComputedStyle(c);
        if (ccs.display === "none" || ccs.visibility === "hidden") continue;
        if (!isInline(ccs)) {
          if (!crossBlocks) continue; // nested block: emitted separately
          const blockBreak = !ccs.display.startsWith("inline");
          if (blockBreak && runs.length) flush();
          const clink = c.tagName === "A" && c.getAttribute("href") ? c.href : link;
          rec(c, runStyle(ccs, clink), clink, wsOf(ccs));
          if (blockBreak && runs.length) flush();
          continue;
        }
        const clink = c.tagName === "A" && c.getAttribute("href") ? c.href : link;
        rec(c, runStyle(ccs, clink), clink, wsOf(ccs));
      }
    };
    rec(el, runStyle(elCs, ""), "", elWs);
    flush();
    for (const p of paras) {
      p.runs = collapseParagraph(p.runs);
      applyTextTransform(p.runs);
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
  const inlineTextRect = (el, crossNonListBlocks) => {
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
        if (crossNonListBlocks && (c.tagName === "UL" || c.tagName === "OL")) continue;
        const ccs = getComputedStyle(c);
        if (ccs.display === "none" || ccs.visibility === "hidden") continue;
        if (!crossNonListBlocks && !isInline(ccs)) continue;
        rec(c, wsOf(ccs));
      }
    };
    rec(el, wsOf(getComputedStyle(el)));
    return Number.isFinite(left) ? { left, top, right, bottom, width: right - left, height: bottom - top } : null;
  };

  const flatRuns = (paras) =>
    paras.map((p) => {
      const out = {
        bullet: p.bullet || "",
        level: p.level || 0,
        runs: p.runs.map((r) => {
          const run = {
            text: r.text,
            bold: r.style.bold,
            italic: r.style.italic,
            underline: r.style.underline,
            strike: r.style.strike,
            color: r.style.color,
            size: r.style.size,
            font: r.style.font,
            link: r.style.link,
          };
          if (r.style.letterSpacing) run.letterSpacing = r.style.letterSpacing;
          return run;
        }),
      };
      if (p.bulletIndent > 0) out.bulletIndent = p.bulletIndent;
      if (p.bulletMargin > 0) out.bulletMargin = p.bulletMargin;
      if (p.spaceAfter > 0) out.spaceAfter = p.spaceAfter;
      return out;
    });

  // mergeListItemParas joins intra-item line breaks (<br> or multiple <p>
  // blocks inside one <li>) with U+000B soft breaks so native lists keep a
  // single bullet or ordered-list number per <li>.
  const mergeListItemParas = (paras, fallbackStyle) => {
    if (paras.length <= 1) return paras;
    const merged = [];
    for (let i = 0; i < paras.length; i++) {
      if (i > 0) {
        const prev = merged[merged.length - 1];
        const st = prev ? prev.style : (paras[i].runs[0] ? paras[i].runs[0].style : fallbackStyle);
        if (prev && sameStyle(prev.style, st)) prev.text += "\v";
        else merged.push({ text: "\v", style: st });
      }
      for (const r of paras[i].runs) {
        const last = merged[merged.length - 1];
        if (last && sameStyle(last.style, r.style)) last.text += r.text;
        else merged.push({ text: r.text, style: r.style });
      }
    }
    return [{ runs: merged }];
  };

  // collectListItems flattens a ul/ol (with nesting) into visual-order items
  // carrying CSS indent, vertical margins, and painted bounds so the caller
  // can compute inter-item paragraph spacing and true top/bottom bounds.
  const collectListItems = (listEl, level, parentMarginPx, items) => {
    const bullet = listEl.tagName === "OL" ? "number" : "disc";
    const listCs = getComputedStyle(listEl);
    const padLeft = round(px(listCs.paddingLeft));
    const effStep = padLeft > 0 ? padLeft : 22.5;
    const cumMargin = round(parentMarginPx + effStep);
    const bulletMargin = level > 0 && Math.abs(cumMargin - effStep * (level + 1)) > 0.05 ? cumMargin : 0;
    const listMt = px(listCs.marginTop);
    let firstInList = true;
    for (const li of listEl.children) {
      if (li.tagName !== "LI") continue;
      const liCs = getComputedStyle(li);
      if (liCs.display === "none" || liCs.visibility === "hidden") continue;
      const hasSubList = [...li.children].some((c) => c.tagName === "UL" || c.tagName === "OL");
      let paras = collectParas(li, true, true);
      if (!paras.length && !hasSubList) {
        paras = [{ runs: [] }];
      } else if (paras.length > 1) {
        paras = mergeListItemParas(paras, runStyle(liCs, ""));
      }
      const mt = firstInList && level > 0 ? Math.max(listMt, px(liCs.marginTop)) : px(liCs.marginTop);
      firstInList = false;
      const mb = px(liCs.marginBottom);
      let idx = -1;
      if (paras.length) {
        let rect = inlineTextRect(li, true);
        if (!rect && !hasSubList) {
          const lr = li.getBoundingClientRect();
          if (lr.width > 0 && lr.height > 0) rect = lr;
        }
        idx = items.length;
        items.push({
          paras,
          bullet,
          level,
          bulletIndent: padLeft,
          bulletMargin,
          mt,
          mb,
          exitMb: 0,
          rect,
        });
      }
      const beforeSub = items.length;
      for (const sub of li.children) {
        if (sub.tagName === "UL" || sub.tagName === "OL") {
          const subStart = items.length;
          collectListItems(sub, level + 1, cumMargin, items);
          if (items.length > subStart) {
            const subMb = px(getComputedStyle(sub).marginBottom);
            items[items.length - 1].exitMb = Math.max(items[items.length - 1].exitMb, subMb);
          }
        }
      }
      if (items.length > beforeSub) {
        // When a nested list ends, the enclosing <li>'s bottom margin applies
        // after the last nested item rather than before the first nested item.
        if (idx >= 0) items[idx].mb = 0;
        items[items.length - 1].exitMb = Math.max(items[items.length - 1].exitMb, mb);
      }
    }
  };

  const extractListParas = (listEl) => {
    const items = [];
    collectListItems(listEl, 0, 0, items);
    const out = [];
    let firstRect = null, lastRect = null;
    for (let i = 0; i < items.length; i++) {
      const it = items[i];
      if (it.rect) {
        if (!firstRect) firstRect = it.rect;
        lastRect = it.rect;
      }
      let gap = 0;
      if (i + 1 < items.length) {
        gap = round(Math.max(0, it.mb, it.exitMb, items[i + 1].mt));
      }
      for (let j = 0; j < it.paras.length; j++) {
        const isLastPara = j === it.paras.length - 1;
        out.push({
          ...it.paras[j],
          bullet: it.bullet,
          level: it.level,
          bulletIndent: it.bulletIndent,
          bulletMargin: it.bulletMargin,
          spaceAfter: isLastPara ? gap : 0,
        });
      }
    }
    return { listParas: out, firstRect, lastRect };
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

    // textBoxRect combines painted vertical bounds from textRect with the
    // element's CSS content-box width when el is a block container (or a
    // centered flex/grid box), preventing premature native line wraps caused
    // by shrinking multi-line blocks to Chrome's ragged glyph union.
    const textBoxRect = (el, cs, textRect) => {
      if (!textRect) return null;
      const er = el.getBoundingClientRect();
      const align = alignOf(cs);
      const isFlexOrGrid = cs.display.includes("flex") || cs.display.includes("grid");
      let cLeft = er.left + px(cs.borderLeftWidth) + px(cs.paddingLeft);
      let cRight = er.right - px(cs.borderRightWidth) - px(cs.paddingRight);
      if (textRect.left >= sRect.left - 1 && textRect.right <= sRect.right + 1) {
        cLeft = Math.max(cLeft, sRect.left);
        cRight = Math.max(cLeft, Math.min(cRight, sRect.right));
      }
      const cW = cRight - cLeft;
      let left = textRect.left;
      let right = textRect.right;
      if (cW >= textRect.width - 0.5) {
        const textMid = (textRect.left + textRect.right) / 2;
        const boxMid = (cLeft + cRight) / 2;
        if (align === "ctr" && Math.abs(textMid - boxMid) <= 3) {
          left = cLeft;
          right = cRight;
        } else if (!isFlexOrGrid && align === "r" && Math.abs(cRight - textRect.right) <= 3) {
          left = cLeft;
          right = cRight;
        } else if (!isFlexOrGrid && align === "l" && textRect.left >= cLeft - 2) {
          left = textRect.left <= cLeft + 1.5 ? cLeft : textRect.left;
          right = Math.max(cRight, textRect.right);
        } else if (isFlexOrGrid && align === "l" && textRect.height > px(cs.fontSize) * 1.7 && cRight > textRect.right) {
          right = cRight;
        }
      }
      const width = right - left;
      return {
        rect: { left, top: textRect.top, right, bottom: textRect.bottom, width, height: textRect.height },
        noSlack: textRect.width > 0 && width >= textRect.width * 1.12,
      };
    };

    const extractTable = (tbl) => {
      const rows = [...tbl.rows];
      if (!rows.length || !rows[0].cells.length) return;
      const r = rel(tbl.getBoundingClientRect());
      const colW = [...rows[0].cells].map((c) => round(c.getBoundingClientRect().width));
      const rowHs = rows.map((tr) => round(tr.getBoundingClientRect().height));
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
        rowHs,
        border: px(getComputedStyle(first).borderBottomWidth) > 0 && bb.alpha > 0 ? bb.hex : "",
        defSize: round(px(fcs.fontSize)),
        rows: rows.map((tr) => [...tr.cells].map((td) => {
          const tdCs = getComputedStyle(td);
          const sp = spacingOf(tdCs);
          const cell = {
            fill: cellBg(td),
            align: alignOf(tdCs),
            paras: flatRuns(collectParas(td, true)),
          };
          if (sp > 0) cell.spacing = sp;
          return cell;
        })),
      });
    };

    // extractImage rasterizes an <img data-embed> at 2x layout resolution
    // into a PNG, honoring object-fit; SVG sources rasterize here too, so
    // the Go side only ever sees PNG bytes.
    const extractImage = (el, r) => {
      if (!el.complete || !el.naturalWidth) {
        warn(el, "img not loaded — embed skipped");
        return;
      }
      try {
        const scale = 2;
        const cw = Math.max(1, Math.round(r.w * scale));
        const ch = Math.max(1, Math.round(r.h * scale));
        const canvas = document.createElement("canvas");
        canvas.width = cw;
        canvas.height = ch;
        const ctx = canvas.getContext("2d");
        let dx = 0, dy = 0, dw = cw, dh = ch;
        const fit = getComputedStyle(el).objectFit;
        if (fit === "contain" || fit === "cover" || fit === "scale-down") {
          const rx = cw / el.naturalWidth, ry = ch / el.naturalHeight;
          let f = fit === "cover" ? Math.max(rx, ry) : Math.min(rx, ry);
          if (fit === "scale-down") f = Math.min(f, scale);
          dw = el.naturalWidth * f;
          dh = el.naturalHeight * f;
          dx = (cw - dw) / 2;
          dy = (ch - dh) / 2;
        }
        ctx.drawImage(el, dx, dy, dw, dh);
        const data = canvas.toDataURL("image/png");
        elements.push({ kind: "image", ...r, png: data.slice(data.indexOf(",") + 1) });
      } catch (e) {
        warn(el, "img embed failed (" + e.message + ") — skipped");
      }
    };

    const walk = (el) => {
      if (SKIP_TAGS.has(el.tagName)) return;
      if (el.tagName === "ASIDE" && (el.classList.contains("notes") || el.hasAttribute("data-notes"))) return; // handled per-section
      const cs = getComputedStyle(el);
      if (cs.display === "none" || cs.visibility === "hidden" || px(cs.opacity) === 0) return;
      if (el.tagName === "IMG" && el.hasAttribute("data-embed")) {
        extractImage(el, rel(el.getBoundingClientRect()));
        return;
      }
      if (MEDIA_TAGS.has(el.tagName)) {
        if (el.tagName === "IMG") {
          warn(el, "img skipped — add data-embed to embed it as a native picture");
        } else {
          warn(el, "media element skipped (output must stay natively editable)");
        }
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
        const { listParas, firstRect, lastRect } = extractListParas(el);
        if (listParas.length) {
          const li = el.querySelector("li");
          const lcs = li ? getComputedStyle(li) : cs;
          const er = el.getBoundingClientRect();
          const left = er.left + px(cs.borderLeftWidth);
          const right = er.right - px(cs.borderRightWidth) - px(cs.paddingRight);
          const top = firstRect ? firstRect.top : er.top;
          const bottom = lastRect ? lastRect.bottom : er.bottom;
          const listRect = rel({
            left,
            top,
            width: Math.max(0, right - left),
            height: Math.max(0, bottom - top),
          });
          elements.push({
            kind: "text",
            ...listRect,
            noSlack: true,
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
          const tb = textBoxRect(el, cs, inlineTextRect(el));
          if (paras.length) {
            const elem = {
              kind: "text",
              ...(tb ? rel(tb.rect) : r),
              align: alignOf(cs),
              spacing: spacingOf(cs),
              fontSize: round(px(cs.fontSize)),
              fontColor: parseColor(cs.color).hex,
              fontFam: cs.fontFamily,
              paras: flatRuns(paras),
            };
            if (tb && tb.noSlack) elem.noSlack = true;
            elements.push(elem);
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
    // speaker notes: hidden aside, preserving paragraphs, <br> line breaks, and <ul>/<ol>/<li> lists
    let notes = "";
    const aside = section.querySelector("aside.notes, aside[data-notes]");
    if (aside) {
      function inlineLines(el) {
        const clone = el.cloneNode(true);
        for (const nested of clone.querySelectorAll("ul, ol")) {
          nested.remove();
        }
        for (const br of clone.querySelectorAll("br")) {
          br.replaceWith("\n");
        }
        return clone.textContent
          .split("\n")
          .map(s => s.replace(/\s+/g, " ").trim());
      }

      function extractList(listEl, depth) {
        const out = [];
        let idx = 1;
        for (const li of listEl.children) {
          if (li.tagName !== "LI") continue;
          const prefix = listEl.tagName === "OL" ? `${idx++}. ` : (depth > 0 ? "  - " : "• ");
          const contIndent = depth > 0 ? "    " : "  ";
          const liLines = inlineLines(li);
          let pushedFirst = false;
          for (const l of liLines) {
            if (!l) continue;
            if (!pushedFirst) {
              out.push(prefix + l);
              pushedFirst = true;
            } else {
              out.push(contIndent + l);
            }
          }
          for (const sub of li.children) {
            if (sub.tagName === "UL" || sub.tagName === "OL") {
              out.push(...extractList(sub, depth + 1));
            }
          }
        }
        return out;
      }

      function extractNoteLines(node) {
        const out = [];
        if (node.children.length === 0) {
          for (const l of inlineLines(node)) {
            if (l) out.push(l);
          }
          return out;
        }
        for (const c of node.children) {
          const tag = c.tagName;
          if (tag === "BR") {
            out.push("");
            continue;
          }
          if (tag === "UL" || tag === "OL") {
            out.push(...extractList(c, 0));
            continue;
          }
          const blockLines = inlineLines(c);
          if (blockLines.length === 1 && blockLines[0] === "") {
            out.push("");
          } else {
            for (const l of blockLines) {
              out.push(l);
            }
          }
          for (const sub of c.children) {
            if (sub.tagName === "UL" || sub.tagName === "OL") {
              out.push(...extractList(sub, 0));
            }
          }
        }
        const cleaned = [];
        for (const l of out) {
          if (l === "" && (cleaned.length === 0 || cleaned[cleaned.length - 1] === "")) {
            continue;
          }
          cleaned.push(l);
        }
        while (cleaned.length > 0 && cleaned[cleaned.length - 1] === "") {
          cleaned.pop();
        }
        return cleaned;
      }

      const lines = extractNoteLines(aside);
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
