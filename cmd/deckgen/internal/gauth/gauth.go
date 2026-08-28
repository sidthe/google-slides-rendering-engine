// Package gauth obtains an authorized *http.Client for the Slides API.
// Order: Application Default Credentials first (GOOGLE_APPLICATION_CREDENTIALS,
// gcloud ADC, metadata server), then an OAuth client JSON with a loopback
// browser consent flow and a cached token. The Drive file scope is used only
// to upload a generated PPTX as a Google Slides presentation.
package gauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Scope is retained for callers that need the core Slides permission.
const Scope = "https://www.googleapis.com/auth/presentations"

// DriveFileScope grants access only to files deckgen creates or opens, which
// is enough for the PPTX import calibration loop.
const DriveFileScope = "https://www.googleapis.com/auth/drive.file"

var Scopes = []string{Scope, DriveFileScope}

// ScopeHint is printed when credentials lack the required Slides/Drive scopes.
const ScopeHint = "hint: your Application Default Credentials likely lack the Google Slides import scopes; re-run:\n" +
	"  gcloud auth application-default login --scopes=openid,https://www.googleapis.com/auth/cloud-platform," + Scope + "," + DriveFileScope

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "deckgen"), nil
}

// Client returns an authorized HTTP client. oauthClientPath, when set,
// forces the OAuth-client flow; otherwise ADC is tried first and the
// default client-secret location (~/.config/deckgen/client_secret.json) is
// the fallback.
func Client(ctx context.Context, oauthClientPath string) (*http.Client, error) {
	return client(ctx, oauthClientPath, false)
}

// Reauthorize always starts the browser consent flow. Use it when the cached
// credential predates an additional deckgen API scope.
func Reauthorize(ctx context.Context, oauthClientPath string) (*http.Client, error) {
	return client(ctx, oauthClientPath, true)
}

func client(ctx context.Context, oauthClientPath string, forceBrowser bool) (*http.Client, error) {
	if oauthClientPath != "" {
		return oauthClient(ctx, oauthClientPath, forceBrowser)
	}
	if !forceBrowser {
		if creds, err := google.FindDefaultCredentials(ctx, Scopes...); err == nil {
			return oauth2.NewClient(ctx, creds.TokenSource), nil
		}
	}
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	def := filepath.Join(dir, "client_secret.json")
	if _, err := os.Stat(def); err == nil {
		return oauthClient(ctx, def, forceBrowser)
	}
	return nil, errors.New(`no Google credentials found. Either:
  - run: gcloud auth application-default login --scopes=openid,https://www.googleapis.com/auth/cloud-platform,` + Scope + `,` + DriveFileScope + `
  - or create an OAuth client (Desktop app) in Google Cloud Console, download its JSON,
    and pass -oauth-client client_secret.json (or save it as ~/.config/deckgen/client_secret.json)`)
}

func oauthClient(ctx context.Context, clientPath string, forceBrowser bool) (*http.Client, error) {
	b, err := os.ReadFile(clientPath)
	if err != nil {
		return nil, err
	}
	conf, err := google.ConfigFromJSON(b, Scopes...)
	if err != nil {
		return nil, fmt.Errorf("%s: not a valid OAuth client JSON: %w", clientPath, err)
	}
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	cachePath := filepath.Join(dir, "token.json")

	if !forceBrowser {
		if tok := loadToken(cachePath); tok != nil {
			ts := conf.TokenSource(ctx, tok)
			if t, err := ts.Token(); err == nil {
				return oauth2.NewClient(ctx, &savingSource{ts: oauth2.ReuseTokenSource(t, ts), path: cachePath, last: t.AccessToken}), nil
			}
			// cached token unusable (revoked/expired refresh) — fall through
		}
	}

	tok, err := loopbackFlow(ctx, conf)
	if err != nil {
		return nil, err
	}
	saveToken(cachePath, tok)
	ts := conf.TokenSource(ctx, tok)
	return oauth2.NewClient(ctx, &savingSource{ts: ts, path: cachePath, last: tok.AccessToken}), nil
}

// loopbackFlow runs the standard installed-app consent: a listener on a
// random 127.0.0.1 port receives the authorization code, PKCE protects the
// exchange.
func loopbackFlow(ctx context.Context, conf *oauth2.Config) (*oauth2.Token, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer ln.Close()
	conf.RedirectURL = fmt.Sprintf("http://%s", ln.Addr().String())

	stateB := make([]byte, 16)
	rand.Read(stateB)
	state := hex.EncodeToString(stateB)
	verifier := oauth2.GenerateVerifier()
	authURL := conf.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))

	fmt.Fprintf(os.Stderr, "Opening browser for Google consent…\nIf it does not open, visit:\n  %s\n", authURL)
	openBrowser(authURL)

	type outcome struct {
		code string
		err  error
	}
	ch := make(chan outcome, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			ch <- outcome{err: errors.New("oauth state mismatch")}
			return
		}
		if e := q.Get("error"); e != "" {
			fmt.Fprintf(w, "Authorization failed: %s. You can close this tab.", e)
			ch <- outcome{err: fmt.Errorf("authorization denied: %s", e)}
			return
		}
		fmt.Fprint(w, "deckgen is authorized. You can close this tab.")
		ch <- outcome{code: q.Get("code")}
	})}
	go srv.Serve(ln)
	defer srv.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case o := <-ch:
		if o.err != nil {
			return nil, o.err
		}
		return conf.Exchange(ctx, o.code, oauth2.VerifierOption(verifier))
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func loadToken(path string) *oauth2.Token {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var t oauth2.Token
	if json.Unmarshal(b, &t) != nil {
		return nil
	}
	return &t
}

func saveToken(path string, t *oauth2.Token) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	b, err := json.Marshal(t)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}

// savingSource persists refreshed tokens so the next run skips the browser.
type savingSource struct {
	ts   oauth2.TokenSource
	path string
	last string
}

func (s *savingSource) Token() (*oauth2.Token, error) {
	t, err := s.ts.Token()
	if err == nil && t.AccessToken != s.last {
		s.last = t.AccessToken
		saveToken(s.path, t)
	}
	return t, err
}
