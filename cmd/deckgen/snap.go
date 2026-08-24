package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sidthe/google-slides-rendering-engine/cmd/deckgen/internal/gauth"
	"github.com/sidthe/google-slides-rendering-engine/gslides"
)

// runSnap downloads Google's own PNG renders of a pushed presentation —
// the ground truth to compare against `deckgen screenshot` output when
// calibrating fidelity.
func runSnap(ctx context.Context, presentationID, outDir, oauthClient string) error {
	hc, err := gauth.Client(ctx, oauthClient)
	if err != nil {
		return err
	}
	thumbs, err := gslides.Thumbnails(ctx, hc, presentationID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for i, t := range thumbs {
		p := filepath.Join(outDir, fmt.Sprintf("slide%02d.png", i+1))
		if err := os.WriteFile(p, t.PNG, 0o644); err != nil {
			return err
		}
		fmt.Println(p)
	}
	return nil
}
