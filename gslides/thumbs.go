package gslides

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Thumbnail is one rendered slide image, as produced by the Slides
// renderer — the ground truth for fidelity calibration against the HTML
// screenshots.
type Thumbnail struct {
	PageID string
	PNG    []byte
}

// Thumbnails downloads a PNG render of every slide in the presentation.
func Thumbnails(ctx context.Context, hc *http.Client, presentationID string) ([]Thumbnail, error) {
	return ThumbnailsFrom(ctx, hc, DefaultBaseURL, presentationID)
}

// ThumbnailsFrom is Thumbnails against an alternate endpoint (tests).
func ThumbnailsFrom(ctx context.Context, hc *http.Client, baseURL, presentationID string) ([]Thumbnail, error) {
	var pres struct {
		Slides []struct {
			ObjectID string `json:"objectId"`
		} `json:"slides"`
	}
	if err := get(ctx, hc, baseURL+"/v1/presentations/"+presentationID+"?fields=slides.objectId", &pres); err != nil {
		return nil, fmt.Errorf("get presentation: %w", err)
	}
	var out []Thumbnail
	for _, s := range pres.Slides {
		var thumb struct {
			ContentURL string `json:"contentUrl"`
		}
		u := fmt.Sprintf("%s/v1/presentations/%s/pages/%s/thumbnail?thumbnailProperties.mimeType=PNG&thumbnailProperties.thumbnailSize=LARGE",
			baseURL, presentationID, s.ObjectID)
		if err := get(ctx, hc, u, &thumb); err != nil {
			return nil, fmt.Errorf("thumbnail %s: %w", s.ObjectID, err)
		}
		png, err := fetch(ctx, hc, thumb.ContentURL)
		if err != nil {
			return nil, fmt.Errorf("download %s: %w", s.ObjectID, err)
		}
		out = append(out, Thumbnail{PageID: s.ObjectID, PNG: png})
	}
	return out, nil
}

func get(ctx context.Context, hc *http.Client, url string, out any) error {
	data, err := fetch(ctx, hc, url)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func fetch(ctx context.Context, hc *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	return data, nil
}
