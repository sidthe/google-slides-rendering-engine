package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sidthe/google-slides-rendering-engine/cmd/deckgen/internal/gauth"
	"github.com/sidthe/google-slides-rendering-engine/gslides"
	"github.com/sidthe/google-slides-rendering-engine/pptx"
)

// runImport exercises the same path users take: HTML -> local PPTX -> Drive
// conversion -> native Google Slides. This is deliberately distinct from push,
// which creates every object directly through the Slides API.
func runImport(ctx context.Context, htmlPath, title, oauthClient, out string, reauth bool) error {
	d, _, err := extract(ctx, htmlPath)
	if err != nil {
		return err
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(htmlPath), ".html")
	}
	var buf bytes.Buffer
	if err := pptx.WriteGoogleSlidesTo(&buf, d); err != nil {
		return err
	}
	if out != "" {
		if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
			return err
		}
		fmt.Printf("saved %s (%d bytes, Google Slides import target)\n", out, buf.Len())
	}
	var hc *http.Client
	if reauth {
		hc, err = gauth.Reauthorize(ctx, oauthClient)
	} else {
		hc, err = gauth.Client(ctx, oauthClient)
	}
	if err != nil {
		return err
	}
	res, err := gslides.ImportPPTX(ctx, hc, buf.Bytes(), title)
	if err != nil {
		return err
	}
	fmt.Println(res.URL)
	return nil
}
