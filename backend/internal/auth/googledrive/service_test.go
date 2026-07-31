package googledrive

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"cloud-saver/backend/internal/config"
)

func TestPKCEVerifierHasRequiredEntropyAndS256Challenge(t *testing.T) {
	verifier, err := newVerifier()
	if err != nil {
		t.Fatal(err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Fatalf("unexpected verifier length %d", len(verifier))
	}
	if strings.ContainsAny(verifier, "+/=") {
		t.Fatalf("verifier is not URL safe: %q", verifier)
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge == verifier || len(challenge) < 43 {
		t.Fatal("S256 challenge was not derived correctly")
	}
}

func TestCallbackReportsFailureWhenTokenExchangeFails(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
	}))
	defer tokenServer.Close()

	cfg, err := config.New(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	callbackResponse := make(chan struct {
		status int
		body   string
	}, 1)
	service := &Service{
		cfg:          cfg,
		clientID:     "public-test.apps.googleusercontent.com",
		clientSecret: "test-client-credential",
		clientSource: "test",
		endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.example.test/auth",
			TokenURL: tokenServer.URL,
		},
	}
	service.openBrowser = func(authURL string) error {
		parsed, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		redirect := parsed.Query().Get("redirect_uri")
		state := parsed.Query().Get("state")
		go func() {
			response, requestErr := http.Get(redirect + "?state=" + url.QueryEscape(state) + "&code=test-code")
			if requestErr != nil {
				callbackResponse <- struct {
					status int
					body   string
				}{status: 0, body: requestErr.Error()}
				return
			}
			defer response.Body.Close()
			body, _ := io.ReadAll(response.Body)
			callbackResponse <- struct {
				status int
				body   string
			}{status: response.StatusCode, body: string(body)}
		}()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, connectErr := service.Connect(ctx)
	if connectErr == nil || !strings.Contains(connectErr.Error(), "oauth_exchange_failed") {
		t.Fatalf("unexpected connect error: %v", connectErr)
	}
	select {
	case result := <-callbackResponse:
		if result.status != http.StatusInternalServerError {
			t.Fatalf("callback reported status %d with body %q", result.status, result.body)
		}
		if strings.Contains(result.body, "Google Drive is connected") {
			t.Fatalf("callback incorrectly reported success: %q", result.body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("callback response did not complete")
	}
}

func TestLoadsDeveloperOAuthJSONWithoutCopyingItIntoSettings(t *testing.T) {
	jsonPath := filepath.Join(t.TempDir(), "oauth.json")
	if err := os.WriteFile(jsonPath, []byte(`{"installed":{"client_id":"json-client","client_secret":"json-credential"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLOUD_SAVER_GOOGLE_CLIENT_ID", "")
	t.Setenv("CLOUD_SAVER_GOOGLE_CLIENT_SECRET", "")
	t.Setenv("CLOUD_SAVER_GOOGLE_OAUTH_JSON", jsonPath)
	clientID, credential, clientSource, credentialSource, err := loadClientConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if clientID != "json-client" || credential != "json-credential" {
		t.Fatal("OAuth JSON values were not loaded")
	}
	if clientSource != "oauth_json" || credentialSource != "oauth_json" {
		t.Fatalf("unexpected sources: %s / %s", clientSource, credentialSource)
	}
}
