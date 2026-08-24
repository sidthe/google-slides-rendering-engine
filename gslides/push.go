package gslides

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sidthe/google-slides-rendering-engine/ir"
)

// DefaultBaseURL is the Slides API endpoint.
const DefaultBaseURL = "https://slides.googleapis.com"

// maxRequestsPerBatch bounds each batchUpdate call, far under the API's
// payload limits. Chunk boundaries never split a Group, so every object's
// create request lands in the same call as (or an earlier call than) the
// requests that reference it.
const maxRequestsPerBatch = 400

// PushResult identifies the created presentation.
type PushResult struct {
	PresentationID string
	URL            string
}

// Push creates a Google Slides presentation titled title and applies the
// deck's request plan through hc, which must already carry OAuth
// credentials with the https://www.googleapis.com/auth/presentations scope.
func Push(ctx context.Context, hc *http.Client, d *ir.Deck, title string) (*PushResult, error) {
	return PushTo(ctx, hc, DefaultBaseURL, d, title)
}

// PushTo is Push against an alternate endpoint (tests).
func PushTo(ctx context.Context, hc *http.Client, baseURL string, d *ir.Deck, title string) (*PushResult, error) {
	created, err := createPresentation(ctx, hc, baseURL, title)
	if err != nil {
		return nil, err
	}

	// The new presentation arrives with one default slide; delete it so the
	// deck starts clean, then append the plan.
	groups := make([]Group, 0, 1)
	for _, s := range created.Slides {
		groups = append(groups, Group{Label: "delete-default", Requests: []Request{
			{DeleteObject: &DeleteObjectRequest{ObjectID: s.ObjectID}},
		}})
	}
	groups = append(groups, Build(d).Groups...)

	for _, chunk := range chunkGroups(groups, maxRequestsPerBatch) {
		if err := batchUpdate(ctx, hc, baseURL, created.PresentationID, chunk); err != nil {
			return nil, err
		}
	}
	if err := pushNotes(ctx, hc, baseURL, created.PresentationID, d); err != nil {
		return nil, err
	}
	return &PushResult{
		PresentationID: created.PresentationID,
		URL:            "https://docs.google.com/presentation/d/" + created.PresentationID + "/edit",
	}, nil
}

// pushNotes writes speaker notes. The notes text shapes' object IDs are
// only discoverable by reading the presentation back, so this runs as a
// follow-up GET + batchUpdate after the slides exist.
func pushNotes(ctx context.Context, hc *http.Client, baseURL, id string, d *ir.Deck) error {
	any := false
	for _, sl := range d.Slides {
		if sl.Notes != "" {
			any = true
		}
	}
	if !any {
		return nil
	}
	var pres struct {
		Slides []struct {
			SlideProperties struct {
				NotesPage struct {
					NotesProperties struct {
						SpeakerNotesObjectID string `json:"speakerNotesObjectId"`
					} `json:"notesProperties"`
				} `json:"notesPage"`
			} `json:"slideProperties"`
		} `json:"slides"`
	}
	if err := get(ctx, hc, baseURL+"/v1/presentations/"+id+"?fields=slides.slideProperties.notesPage.notesProperties.speakerNotesObjectId", &pres); err != nil {
		return fmt.Errorf("read notes shape ids: %w", err)
	}
	var reqs []Request
	for i, sl := range d.Slides {
		if sl.Notes == "" || i >= len(pres.Slides) {
			continue
		}
		notesID := pres.Slides[i].SlideProperties.NotesPage.NotesProperties.SpeakerNotesObjectID
		if notesID == "" {
			continue
		}
		reqs = append(reqs, Request{InsertText: &InsertTextRequest{
			ObjectID: notesID, Text: sl.Notes, InsertionIndex: 0,
		}})
	}
	if len(reqs) == 0 {
		return nil
	}
	return batchUpdate(ctx, hc, baseURL, id, reqs)
}

// chunkGroups flattens groups into request chunks of at most limit,
// keeping each group whole (a group larger than limit becomes its own
// oversized chunk rather than being split).
func chunkGroups(groups []Group, limit int) [][]Request {
	var chunks [][]Request
	var cur []Request
	for _, g := range groups {
		if len(cur) > 0 && len(cur)+len(g.Requests) > limit {
			chunks = append(chunks, cur)
			cur = nil
		}
		cur = append(cur, g.Requests...)
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	return chunks
}

type createdPresentation struct {
	PresentationID string `json:"presentationId"`
	Slides         []struct {
		ObjectID string `json:"objectId"`
	} `json:"slides"`
}

func createPresentation(ctx context.Context, hc *http.Client, baseURL, title string) (*createdPresentation, error) {
	body, _ := json.Marshal(map[string]string{"title": title})
	var out createdPresentation
	if err := post(ctx, hc, baseURL+"/v1/presentations", body, &out); err != nil {
		return nil, fmt.Errorf("create presentation: %w", err)
	}
	if out.PresentationID == "" {
		return nil, fmt.Errorf("create presentation: response missing presentationId")
	}
	return &out, nil
}

func batchUpdate(ctx context.Context, hc *http.Client, baseURL, id string, reqs []Request) error {
	body, err := json.Marshal(map[string]any{"requests": reqs})
	if err != nil {
		return err
	}
	if err := post(ctx, hc, baseURL+"/v1/presentations/"+id+":batchUpdate", body, nil); err != nil {
		return fmt.Errorf("batchUpdate (%d requests): %w", len(reqs), err)
	}
	return nil
}

func post(ctx context.Context, hc *http.Client, url string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// APIError is a non-2xx Slides API response.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("slides API: HTTP %d: %s", e.Status, e.Body)
}
