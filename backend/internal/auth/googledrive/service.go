// Package googledrive owns direct Google Drive authorization. It never returns
// authorization codes or tokens to callers/UI code.
package googledrive

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	"cloud-saver/backend/internal/logstore"
	"cloud-saver/backend/internal/models"
)

const scope = drive.DriveFileScope

// defaultClientID is an installed-app identifier, which is public by design.
// Never add the client secret from Google's downloaded JSON to this repository.
var defaultClientID = "470012579652-bh1fo958rfv2f6l8t7mkbflma4eha822.apps.googleusercontent.com"

// Populated only in release builds with -ldflags -X. A desktop OAuth client
// credential cannot remain confidential inside a distributed executable, but
// keeping it out of Git prevents accidental copying from the repository.
var defaultClientSecret string

type Service struct {
	cfg          *config.Manager
	logs         *logstore.Store
	clientID     string
	clientSecret string
	clientSource string
	secretSource string
	configErr    error
	debug        bool
	endpoint     oauth2.Endpoint
	openBrowser  func(string) error
	mu           sync.Mutex
}

func New(cfg *config.Manager, logs *logstore.Store, debug bool) *Service {
	clientID, clientSecret, clientSource, secretSource, configErr := loadClientConfiguration()
	return &Service{
		cfg:          cfg,
		logs:         logs,
		clientID:     clientID,
		clientSecret: clientSecret,
		clientSource: clientSource,
		secretSource: secretSource,
		configErr:    configErr,
		debug:        debug,
		endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		openBrowser: openSystemBrowser,
	}
}

func (s *Service) Status() models.GoogleDriveSettings {
	value := s.cfg.Get().GoogleDrive
	value.EncryptedRefreshToken = ""
	return value
}
func (s *Service) Available() bool {
	return s.clientID != "" && s.clientSecret != "" && s.configErr == nil
}
func (s *Service) ClientIDSource() string   { return s.clientSource }
func (s *Service) CredentialSource() string { return s.secretSource }

func (s *Service) Connect(ctx context.Context) (models.GoogleDriveSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.debugLog("connect_start", "starting OAuth connection with client source "+s.clientSource+" and credential source "+s.secretSource)
	if s.configErr != nil {
		s.errorLog("configuration_invalid", s.configErr)
		return models.GoogleDriveSettings{}, errors.New("Google OAuth developer configuration could not be loaded; run with --debug for details")
	}
	if s.clientID == "" {
		return models.GoogleDriveSettings{}, errors.New("Google Drive connection is not configured for this build")
	}
	if s.clientSecret == "" {
		return models.GoogleDriveSettings{}, errors.New("Google OAuth client credential is missing in this development build. Set CLOUD_SAVER_GOOGLE_OAUTH_JSON to the downloaded Desktop OAuth JSON, or inject CLOUD_SAVER_GOOGLE_CLIENT_SECRET when building")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		s.errorLog("callback_listener_failed", err)
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
	s.debugLog("callback_ready", "OAuth callback listener is ready on loopback")
	callback := make(chan callbackResult, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" || r.URL.Query().Get("state") != state {
			s.debugLog("callback_rejected", "callback path or state did not match")
			http.Error(w, "Sign-in could not be verified. Return to Cloud Saver and try again.", http.StatusBadRequest)
			return
		}
		if value := r.URL.Query().Get("error"); value != "" {
			s.debugLog("callback_cancelled", "Google returned an OAuth error")
			http.Error(w, "Google sign-in was cancelled. You can close this tab and return to Cloud Saver.", http.StatusBadRequest)
			select {
			case callback <- callbackResult{err: fmt.Errorf("Google sign-in cancelled: %s", value)}:
			default:
			}
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			s.debugLog("callback_missing_code", "verified callback did not include an authorization code")
			http.Error(w, "Google did not return a sign-in result.", http.StatusBadRequest)
			select {
			case callback <- callbackResult{err: errors.New("Google sign-in did not return a code")}:
			default:
			}
			return
		}
		s.debugLog("callback_received", "verified callback received; finishing token exchange")
		done := make(chan error, 1)
		select {
		case callback <- callbackResult{code: code, done: done}:
		default:
			http.Error(w, "This sign-in callback has already been handled.", http.StatusConflict)
			return
		}
		select {
		case finishErr := <-done:
			if finishErr != nil {
				http.Error(w, "Google authorization was received, but Cloud Saver could not finish the connection. Return to the app for details.", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Google Drive is connected. You can close this tab and return to Cloud Saver."))
		case <-r.Context().Done():
		case <-time.After(90 * time.Second):
			http.Error(w, "Cloud Saver timed out while finishing the connection. Return to the app and try again.", http.StatusGatewayTimeout)
		}
	})}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			s.errorLog("callback_server_failed", serveErr)
		}
	}()
	defer server.Shutdown(context.Background())
	if err := s.openBrowser(conf.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce, oauth2.S256ChallengeOption(verifier))); err != nil {
		s.errorLog("browser_open_failed", err)
		return models.GoogleDriveSettings{}, err
	}
	s.debugLog("browser_opened", "system browser opened for Google authorization")
	select {
	case <-ctx.Done():
		s.errorLog("connect_context_ended", ctx.Err())
		return models.GoogleDriveSettings{}, ctx.Err()
	case result := <-callback:
		if result.err != nil {
			s.errorLog("callback_failed", result.err)
			return models.GoogleDriveSettings{}, result.err
		}
		status, finishErr := s.finishConnection(ctx, conf, result.code, verifier)
		if result.done != nil {
			result.done <- finishErr
		}
		return status, finishErr
	case <-time.After(5 * time.Minute):
		err := errors.New("Google sign-in timed out")
		s.errorLog("connect_timeout", err)
		return models.GoogleDriveSettings{}, err
	}
}

func (s *Service) finishConnection(ctx context.Context, conf *oauth2.Config, code, verifier string) (models.GoogleDriveSettings, error) {
	s.debugLog("token_exchange_start", "exchanging the authorization code with PKCE")
	token, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		s.errorLog("token_exchange_failed", err)
		return models.GoogleDriveSettings{}, errors.New("Google token exchange failed (oauth_exchange_failed). Run the daemon with --debug for details")
	}
	s.debugLog("token_exchange_complete", "token exchange completed; refresh token present="+fmt.Sprint(token.RefreshToken != ""))
	if token.RefreshToken == "" {
		err := errors.New("Google did not provide ongoing access; reconnect and approve access")
		s.errorLog("refresh_token_missing", err)
		return models.GoogleDriveSettings{}, err
	}
	email, err := accountEmail(ctx, conf.Client(ctx, token))
	if err != nil {
		// Account display metadata is useful but not required for a valid
		// refresh-token connection. Keep the connection and let real Drive
		// operations report provider availability separately.
		s.warningLog("account_metadata_unavailable", err)
		email = ""
	} else {
		s.debugLog("account_metadata_complete", "Google account metadata was confirmed")
	}
	encrypted, err := protect(token.RefreshToken)
	if err != nil {
		s.errorLog("token_protection_failed", err)
		return models.GoogleDriveSettings{}, fmt.Errorf("securely store Google connection: %w", err)
	}
	s.debugLog("token_protected", "refresh token was protected with Windows DPAPI")
	if err := s.cfg.SetGoogleDriveConnection(email, encrypted); err != nil {
		s.errorLog("connection_persist_failed", err)
		return models.GoogleDriveSettings{}, err
	}
	s.debugLog("connect_complete", "Google Drive connection was persisted successfully")
	return s.Status(), nil
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
	endpoint := s.endpoint
	if endpoint.AuthURL == "" {
		endpoint.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if endpoint.TokenURL == "" {
		endpoint.TokenURL = "https://oauth2.googleapis.com/token"
	}
	return &oauth2.Config{ClientID: s.clientID, ClientSecret: s.clientSecret, Endpoint: endpoint, RedirectURL: redirect, Scopes: []string{scope}}
}

type callbackResult struct {
	code string
	err  error
	done chan error
}

func (s *Service) debugLog(stage, message string) {
	if !s.debug {
		return
	}
	log.Printf("[google-drive:%s] %s", stage, message)
	if s.logs != nil {
		_ = s.logs.Write("debug", "google_drive_oauth", stage+": "+message)
	}
}

func (s *Service) warningLog(stage string, err error) {
	message := stage + ": " + s.safeDiagnostic(err)
	if s.logs != nil {
		_ = s.logs.Write("warning", "google_drive_oauth", message)
	}
	if s.debug {
		log.Printf("[google-drive:%s] warning: %s", stage, s.safeDiagnostic(err))
	}
}

func (s *Service) errorLog(stage string, err error) {
	message := stage + ": " + s.safeDiagnostic(err)
	if s.logs != nil {
		_ = s.logs.Write("error", "google_drive_oauth", message)
	}
	if s.debug {
		log.Printf("[google-drive:%s] error: %s", stage, s.safeDiagnostic(err))
	}
}

func (s *Service) safeDiagnostic(err error) string {
	if err == nil {
		return ""
	}
	value := err.Error()
	if s.clientID != "" {
		value = strings.ReplaceAll(value, s.clientID, "[client-id]")
	}
	if s.clientSecret != "" {
		value = strings.ReplaceAll(value, s.clientSecret, "[client-credential]")
	}
	if len(value) > 800 {
		value = value[:800] + "..."
	}
	return value
}

type oauthJSON struct {
	Installed *oauthJSONClient `json:"installed"`
	Web       *oauthJSONClient `json:"web"`
}

type oauthJSONClient struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func loadClientConfiguration() (clientID, clientSecret, clientSource, secretSource string, configErr error) {
	clientID = strings.TrimSpace(os.Getenv("CLOUD_SAVER_GOOGLE_CLIENT_ID"))
	clientSecret = strings.TrimSpace(os.Getenv("CLOUD_SAVER_GOOGLE_CLIENT_SECRET"))
	if clientID != "" {
		clientSource = "environment"
	}
	if clientSecret != "" {
		secretSource = "environment"
	}

	jsonPath := strings.TrimSpace(os.Getenv("CLOUD_SAVER_GOOGLE_OAUTH_JSON"))
	if jsonPath != "" && (clientID == "" || clientSecret == "") {
		payload, err := os.ReadFile(jsonPath)
		if err != nil {
			return clientID, clientSecret, clientSource, secretSource, errors.New("OAuth JSON file could not be read")
		}
		var value oauthJSON
		if err := json.Unmarshal(payload, &value); err != nil {
			return clientID, clientSecret, clientSource, secretSource, fmt.Errorf("decode OAuth JSON: %w", err)
		}
		candidate := value.Installed
		if candidate == nil {
			candidate = value.Web
		}
		if candidate == nil {
			return clientID, clientSecret, clientSource, secretSource, errors.New("OAuth JSON does not contain an installed or web client")
		}
		if clientID == "" {
			clientID, clientSource = strings.TrimSpace(candidate.ClientID), "oauth_json"
		}
		if clientSecret == "" {
			clientSecret, secretSource = strings.TrimSpace(candidate.ClientSecret), "oauth_json"
		}
	}

	if clientID == "" {
		clientID, clientSource = strings.TrimSpace(defaultClientID), "built_in"
	}
	if clientSecret == "" && strings.TrimSpace(defaultClientSecret) != "" {
		clientSecret, secretSource = strings.TrimSpace(defaultClientSecret), "release_embedded"
	}
	if secretSource == "" {
		secretSource = "missing"
	}
	return clientID, clientSecret, clientSource, secretSource, nil
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
