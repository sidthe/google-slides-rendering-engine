package gslides

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

const driveUploadURL = "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&fields=id,webViewLink"

// ImportPPTX uploads a generated PowerPoint file and asks Drive to convert it
// into a native, editable Google Slides presentation. The caller needs the
// narrowly scoped drive.file OAuth permission in addition to presentations.
func ImportPPTX(ctx context.Context, hc *http.Client, pptx []byte, title string) (*PushResult, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	meta, _ := json.Marshal(map[string]string{
		"name":     title,
		"mimeType": "application/vnd.google-apps.presentation",
	})
	metaHeader := make(textproto.MIMEHeader)
	metaHeader.Set("Content-Type", "application/json; charset=UTF-8")
	metaPart, err := mw.CreatePart(metaHeader)
	if err != nil {
		return nil, err
	}
	if _, err := metaPart.Write(meta); err != nil {
		return nil, err
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	part, err := mw.CreatePart(h)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(pptx); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, driveUploadURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "multipart/related; boundary="+mw.Boundary())
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		var msg bytes.Buffer
		_, _ = msg.ReadFrom(resp.Body)
		return nil, fmt.Errorf("Drive PPTX import: HTTP %d: %s", resp.StatusCode, msg.String())
	}
	var out struct {
		ID          string `json:"id"`
		WebViewLink string `json:"webViewLink"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.ID == "" {
		return nil, fmt.Errorf("Drive PPTX import: response missing id")
	}
	url := out.WebViewLink
	if url == "" {
		url = "https://docs.google.com/presentation/d/" + out.ID + "/edit"
	}
	return &PushResult{PresentationID: out.ID, URL: url}, nil
}
