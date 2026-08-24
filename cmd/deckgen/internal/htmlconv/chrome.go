package htmlconv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

import _ "embed"

//go:embed extract.js
var extractJS string

// session starts a headless Chrome, navigates to the HTML file, waits for
// web fonts, and runs fn inside the page context.
func session(ctx context.Context, htmlPath string, fn func(ctx context.Context) error) error {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return err
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1600, 1000),
		chromedp.Flag("force-device-scale-factor", "1"),
		chromedp.Flag("hide-scrollbars", true),
	)
	if p := os.Getenv("DECKGEN_CHROME"); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	cctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	err = chromedp.Run(cctx,
		chromedp.Navigate("file://"+abs),
		chromedp.Evaluate(`document.fonts.ready.then(() => true)`, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
	)
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			return fmt.Errorf("headless Chrome/Chromium not found — install Google Chrome, or point DECKGEN_CHROME at a chrome binary: %w", err)
		}
		return err
	}
	return fn(cctx)
}

// Extract renders the HTML deck in headless Chrome and returns its scene
// graph.
func Extract(ctx context.Context, htmlPath string) (*Result, error) {
	var res Result
	err := session(ctx, htmlPath, func(cctx context.Context) error {
		return chromedp.Run(cctx, chromedp.Evaluate(extractJS, &res))
	})
	if err != nil {
		return nil, err
	}
	if len(res.Scenes) == 0 {
		return nil, fmt.Errorf("%s: no <section> elements found — each slide must be a 1600x900 <section>", htmlPath)
	}
	return &res, nil
}

// Screenshot writes one PNG per <section> into outDir and returns the
// paths — the visual self-review step for AI harnesses before pushing.
func Screenshot(ctx context.Context, htmlPath, outDir string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	var count int
	var paths []string
	err := session(ctx, htmlPath, func(cctx context.Context) error {
		if err := chromedp.Run(cctx, chromedp.Evaluate(`document.querySelectorAll("section").length`, &count)); err != nil {
			return err
		}
		for i := 1; i <= count; i++ {
			var buf []byte
			sel := fmt.Sprintf("section:nth-of-type(%d)", i)
			if err := chromedp.Run(cctx, chromedp.Screenshot(sel, &buf, chromedp.NodeVisible)); err != nil {
				return fmt.Errorf("screenshot %s: %w", sel, err)
			}
			p := filepath.Join(outDir, fmt.Sprintf("slide%02d.png", i))
			if err := os.WriteFile(p, buf, 0o644); err != nil {
				return err
			}
			paths = append(paths, p)
		}
		return nil
	})
	return paths, err
}
