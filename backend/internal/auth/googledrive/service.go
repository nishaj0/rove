// Package googledrive owns direct Google Drive authorization. It never returns
// authorization codes or tokens to callers/UI code.
package googledrive

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"cloud-saver/backend/internal/config"
	"cloud-saver/backend/internal/models"
)

const scope = drive.DriveFileScope

// defaultClientID is an installed-app identifier, which is public by design.
// Never add the client secret from Google's downloaded JSON to this repository.
const defaultClientID = "470012579652-bh1fo958rfv2f6l8t7mkbflma4eha822.apps.googleusercontent.com"

type Service struct {
	cfg         *config.Manager
	clientID    string
	openBrowser func(string) error
	mu          sync.Mutex
}

func New(cfg *config.Manager) *Service {
	clientID := strings.TrimSpace(os.Getenv("CLOUD_SAVER_GOOGLE_CLIENT_ID"))
	if clientID == "" {
		clientID = defaultClientID
	}
	return &Service{cfg: cfg, clientID: clientID, openBrowser: openSystemBrowser}
}

func (s *Service) Status() models.GoogleDriveSettings {
	value := s.cfg.Get().GoogleDrive
	value.EncryptedRefreshToken = ""
	return value
}
func (s *Service) Available() bool { return s.clientID != "" }

func (s *Service) Connect(ctx context.Context) (models.GoogleDriveSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clientID == "" {
		return models.GoogleDriveSettings{}, errors.New("Google Drive connection is not configured for this build")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return models.GoogleDriveSettings{}, err
	}
	defer listener.Close()
	verifier, err := newVerifier()
	if err != nil {
		return models.GoogleDriveSettings{}, err
	}
	state, err := randomURLToken(32)
	if err != nil {
		return models.GoogleDriveSettings{}, err
	}
	redirect := "http://" + listener.Addr().String() + "/callback"
	conf := s.oauthConfig(redirect)
	callback := make(chan callbackResult, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" || r.URL.Query().Get("state") != state {
			http.Error(w, "Sign-in could not be verified. Return to Cloud Saver and try again.", http.StatusBadRequest)
			callback <- callbackResult{err: errors.New("Google sign-in callback was rejected")}
			return
		}
		if value := r.URL.Query().Get("error"); value != "" {
			http.Error(w, "Google sign-in was cancelled. You can close this tab and return to Cloud Saver.", http.StatusBadRequest)
			callback <- callbackResult{err: fmt.Errorf("Google sign-in cancelled: %s", value)}
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Google did not return a sign-in result.", http.StatusBadRequest)
			callback <- callbackResult{err: errors.New("Google sign-in did not return a code")}
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Google Drive is connected. You can close this tab and return to Cloud Saver."))
		callback <- callbackResult{code: code}
	})}
	go server.Serve(listener)
	defer server.Shutdown(context.Background())
	if err := s.openBrowser(conf.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce, oauth2.S256ChallengeOption(verifier))); err != nil {
		return models.GoogleDriveSettings{}, err
	}
	select {
	case <-ctx.Done():
		return models.GoogleDriveSettings{}, ctx.Err()
	case result := <-callback:
		if result.err != nil {
			return models.GoogleDriveSettings{}, result.err
		}
		token, err := conf.Exchange(ctx, result.code, oauth2.VerifierOption(verifier))
		if err != nil {
			return models.GoogleDriveSettings{}, errors.New("Google sign-in could not be completed")
		}
		if token.RefreshToken == "" {
			return models.GoogleDriveSettings{}, errors.New("Google did not provide ongoing access; reconnect and approve access")
		}
		email, err := accountEmail(ctx, conf.Client(ctx, token))
		if err != nil {
			return models.GoogleDriveSettings{}, err
		}
		encrypted, err := protect(token.RefreshToken)
		if err != nil {
			return models.GoogleDriveSettings{}, fmt.Errorf("securely store Google connection: %w", err)
		}
		if err := s.cfg.SetGoogleDriveConnection(email, encrypted); err != nil {
			return models.GoogleDriveSettings{}, err
		}
		return s.Status(), nil
	case <-time.After(5 * time.Minute):
		return models.GoogleDriveSettings{}, errors.New("Google sign-in timed out")
	}
}

func (s *Service) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.cfg.Get()
	refresh := settings.GoogleDrive.EncryptedRefreshToken
	if refresh != "" {
		if token, err := unprotect(refresh); err == nil {
			values := url.Values{"token": {token}}
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/revoke", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if response, err := http.DefaultClient.Do(req); err == nil {
				response.Body.Close()
			}
		}
	}
	return s.cfg.ClearGoogleDriveConnection()
}

func (s *Service) HTTPClient(ctx context.Context) (*http.Client, error) {
	settings := s.cfg.Get()
	if !settings.GoogleDrive.Connected || settings.GoogleDrive.EncryptedRefreshToken == "" {
		return nil, errors.New("Google Drive is not connected")
	}
	refresh, err := unprotect(settings.GoogleDrive.EncryptedRefreshToken)
	if err != nil {
		return nil, errors.New("Google Drive access needs to be reconnected")
	}
	conf := s.oauthConfig("")
	source := oauth2.ReuseTokenSource(nil, conf.TokenSource(ctx, &oauth2.Token{RefreshToken: refresh}))
	return &http.Client{Transport: &reconnectTransport{base: &oauth2.Transport{Source: source}, mark: s.cfg.MarkGoogleDriveReconnectRequired}}, nil
}

func (s *Service) oauthConfig(redirect string) *oauth2.Config {
	return &oauth2.Config{ClientID: s.clientID, Endpoint: oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token"}, RedirectURL: redirect, Scopes: []string{scope}}
}

type callbackResult struct {
	code string
	err  error
}

func newVerifier() (string, error) { return randomURLToken(64) }
func randomURLToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
func openSystemBrowser(url string) error {
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
}
func accountEmail(ctx context.Context, client *http.Client) (string, error) {
	service, err := drive.NewService(ctx, option.WithHTTPClient(client), option.WithScopes(scope))
	if err != nil {
		return "", err
	}
	about, err := service.About.Get().Fields("user(emailAddress)").Context(ctx).Do()
	if err != nil || about.User == nil || about.User.EmailAddress == "" {
		return "", errors.New("could not confirm the Google account")
	}
	return about.User.EmailAddress, nil
}

type reconnectTransport struct {
	base http.RoundTripper
	mark func() error
}

func (t *reconnectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(req)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "invalid_grant") {
		_ = t.mark()
		return nil, errors.New("Google Drive access needs to be reconnected")
	}
	return response, err
}
