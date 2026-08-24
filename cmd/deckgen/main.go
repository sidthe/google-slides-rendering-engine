// deckgen converts an HTML deck (one 1600x900 <section> per slide) into
// fully native, editable presentations: a .pptx file or a Google Slides
// presentation created through the Slides API. Headless Chrome does the
// HTML layout; every element becomes a real shape, text box or table — no
// screenshots embedded in the output.
//
// Usage:
//
//	deckgen check      deck.html                 extract and report conversion warnings
//	deckgen build      deck.html [-o out.pptx]   write a .pptx
//	deckgen requests   deck.html                 print the Slides batchUpdate JSON
//	deckgen screenshot deck.html [-o shots/]     per-slide PNGs for visual self-review
//	deckgen push       deck.html [-title T]      create a Google Slides presentation
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sidthe/google-slides-rendering-engine/cmd/deckgen/internal/htmlconv"
	"github.com/sidthe/google-slides-rendering-engine/gslides"
	"github.com/sidthe/google-slides-rendering-engine/ir"
	"github.com/sidthe/google-slides-rendering-engine/pptx"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}
	cmd, htmlPath := os.Args[1], os.Args[2]
	args := os.Args[3:]
	ctx := context.Background()

	var err error
	switch cmd {
	case "check":
		err = runCheck(ctx, htmlPath, flagValue(args, "-scene", ""))
	case "build":
		err = runBuild(ctx, htmlPath, flagValue(args, "-o", strings.TrimSuffix(htmlPath, ".html")+".pptx"))
	case "requests":
		err = runRequests(ctx, htmlPath)
	case "screenshot":
		err = runScreenshot(ctx, htmlPath, flagValue(args, "-o", "shots"))
	case "push":
		err = runPush(ctx, htmlPath, flagValue(args, "-title", ""), flagValue(args, "-oauth-client", ""))
	case "snap":
		// htmlPath is a presentation ID here
		err = runSnap(ctx, htmlPath, flagValue(args, "-o", "shots-slides"), flagValue(args, "-oauth-client", ""))
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `deckgen — HTML decks to native PPTX / Google Slides

usage:
  deckgen check      deck.html                 extract, report warnings
  deckgen build      deck.html [-o out.pptx]   write .pptx
  deckgen requests   deck.html                 print Slides batchUpdate JSON
  deckgen screenshot deck.html [-o shots/]     per-slide PNGs
  deckgen push       deck.html [-title T] [-oauth-client client_secret.json]
  deckgen snap       <presentationId> [-o shots-slides/]   download Google's renders of a pushed deck
`)
}

func flagValue(args []string, name, def string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return def
}

// extract runs Chrome, maps to IR, and prints warnings to stderr.
func extract(ctx context.Context, htmlPath string) (*ir.Deck, []string, error) {
	res, err := htmlconv.Extract(ctx, htmlPath)
	if err != nil {
		return nil, nil, err
	}
	d, warns := htmlconv.MapScenes(res)
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	return d, warns, nil
}

func runCheck(ctx context.Context, htmlPath, scenePath string) error {
	if scenePath != "" {
		res, err := htmlconv.Extract(ctx, htmlPath)
		if err != nil {
			return err
		}
		f, err := os.Create(scenePath)
		if err != nil {
			return err
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", " ")
		if err := enc.Encode(res); err != nil {
			return err
		}
	}
	d, warns, err := extract(ctx, htmlPath)
	if err != nil {
		return err
	}
	shapes := 0
	for _, s := range d.Slides {
		shapes += len(s.Shapes)
	}
	fmt.Printf("%s: %d slide(s), %d native element(s), %d warning(s)\n",
		htmlPath, len(d.Slides), shapes, len(warns))
	return nil
}

func runBuild(ctx context.Context, htmlPath, out string) error {
	d, _, err := extract(ctx, htmlPath)
	if err != nil {
		return err
	}
	if err := pptx.Write(out, d); err != nil {
		return err
	}
	st, _ := os.Stat(out)
	fmt.Printf("saved %s (%d bytes, %d slides)\n", out, st.Size(), len(d.Slides))
	return nil
}

func runRequests(ctx context.Context, htmlPath string) error {
	d, _, err := extract(ctx, htmlPath)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	return enc.Encode(gslides.Build(d).Requests())
}

func runScreenshot(ctx context.Context, htmlPath, outDir string) error {
	paths, err := htmlconv.Screenshot(ctx, htmlPath, outDir)
	if err != nil {
		return err
	}
	for _, p := range paths {
		fmt.Println(p)
	}
	return nil
}
