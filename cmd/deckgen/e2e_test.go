package main

import (
	"context"
	"os"
	"testing"
)

// TestPushE2E pushes the demo deck to a real Google Slides presentation.
// It needs Chrome plus Slides credentials, so it only runs when
// DECKGEN_E2E=1. The created presentation is left in place for inspection;
// its URL is printed to stdout.
func TestPushE2E(t *testing.T) {
	if os.Getenv("DECKGEN_E2E") != "1" {
		t.Skip("set DECKGEN_E2E=1 (with ADC or ~/.config/deckgen credentials) to run")
	}
	if err := runPush(context.Background(), "../../examples/demo.html", "deck-engine E2E", ""); err != nil {
		t.Fatal(err)
	}
}
