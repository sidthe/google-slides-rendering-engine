package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/deck-engine/deck-engine/cmd/deckgen/internal/gauth"
	"github.com/deck-engine/deck-engine/gslides"
)

func runPush(ctx context.Context, htmlPath, title, oauthClient string) error {
	d, _, err := extract(ctx, htmlPath)
	if err != nil {
		return err
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(htmlPath), ".html")
	}
	hc, err := gauth.Client(ctx, oauthClient)
	if err != nil {
		return err
	}
	res, err := gslides.Push(ctx, hc, d, title)
	if err != nil {
		var apiErr *gslides.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 403 &&
			(strings.Contains(apiErr.Body, "SCOPE") || strings.Contains(apiErr.Body, "insufficient")) {
			return fmt.Errorf("%w\n%s", err, gauth.ScopeHint)
		}
		if strings.Contains(err.Error(), "invalid_scope") {
			return fmt.Errorf("%w\n%s", err, gauth.ScopeHint)
		}
		return err
	}
	fmt.Println(res.URL)
	return nil
}
