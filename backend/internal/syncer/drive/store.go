// Package drive implements syncer.RevisionStore on the Google Drive v3 API.
// It owns only files created by Cloud Saver and requests drive.file exclusively.
package drive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"cloud-saver/backend/internal/models"
	"cloud-saver/backend/internal/syncer"
)

const (
	folderMIMEType                 = "application/vnd.google-apps.folder"
	maxDriveRequestsPerSecond      = 8
	driveUploadChunkSize           = 8 * 1024 * 1024
	driveChunkTransferTimeout      = 5 * time.Minute
	driveChunkRetryDeadline        = 10 * time.Minute
)

var driveMediaOptions = []googleapi.MediaOption{
	googleapi.ChunkSize(driveUploadChunkSize),
	googleapi.ChunkTransferTimeout(driveChunkTransferTimeout),
	googleapi.ChunkRetryDeadline(driveChunkRetryDeadline),
}

// Store maps the immutable Cloud Saver revision layout to Drive folders/files.
type Store struct {
	service *drive.Service
	mu      sync.Mutex
	folders map[string]string
}

func New(ctx context.Context, client *http.Client) (*Store, error) {
	if client == nil {
		return nil, errors.New("Google Drive client is unavailable")
	}
	driveClient := *client
	driveClient.Transport = newPacedTransport(client.Transport, maxDriveRequestsPerSecond)
	svc, err := drive.NewService(ctx, option.WithHTTPClient(&driveClient), option.WithScopes(drive.DriveFileScope))
	if err != nil {
		return nil, err
	}
	return &Store{service: svc, folders: map[string]string{}}, nil
}

func (s *Store) ReadHead(ctx context.Context, syncID string) (string, error) {
	payload, err := s.readFile(ctx, syncID, "head.json")
	if errors.Is(err, errNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var value struct {
		RevisionID string `json:"revisionId"`
	}
	if err := json.Unmarshal(payload, &value); err != nil || value.RevisionID == "" {
		return "", errors.New("Google Drive revision is incomplete; retry shortly")
	}
	return value.RevisionID, nil
}
func (s *Store) WriteHead(ctx context.Context, syncID, revisionID string) error {
	payload, _ := json.Marshal(struct {
		RevisionID string `json:"revisionId"`
	}{revisionID})
	return s.writeNamedFile(ctx, syncID, "head.json", payload, false)
}
func (s *Store) WriteDeviceHead(ctx context.Context, syncID, deviceID, revisionID string) error {
	payload, _ := json.Marshal(struct {
		RevisionID string `json:"revisionId"`
	}{revisionID})
	return s.writeNamedFile(ctx, syncID, "heads/"+deviceID+".json", payload, false)
}
func (s *Store) WriteGameProfile(ctx context.Context, profile models.GameProfile) error {
	payload, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return s.writeNamedFile(ctx, profile.SyncID, "game.json", payload, false)
}
func (s *Store) ListGameProfiles(ctx context.Context) ([]models.GameProfile, error) {
	query := "appProperties has { key='cloudSaverType' and value='game_profile' } and trashed = false"
	pageToken := ""
	result := []models.GameProfile{}
	for {
		call := s.service.Files.List().Q(query).Spaces("drive").Fields("nextPageToken,files(id)").PageSize(100).Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		page, err := call.Do()
		if err != nil {
			return nil, wrapDriveError(err)
		}
		for _, file := range page.Files {
			response, err := s.service.Files.Get(file.Id).Context(ctx).Download()
			if err != nil {
				return nil, wrapDriveError(err)
			}
			var profile models.GameProfile
			decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&profile)
			_ = response.Body.Close()
			if decodeErr == nil && profile.SyncID != "" {
				result = append(result, profile)
			}
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	return result, nil
}
func (s *Store) WriteRecoveryPoint(ctx context.Context, syncID string, point models.RecoveryPoint) error {
	payload, err := json.Marshal(point)
	if err != nil {
		return err
	}
	return s.writeNamedFile(ctx, syncID, "recovery-points/"+point.ID+".json", payload, false)
}
func (s *Store) DeleteRecoveryPoint(ctx context.Context, syncID, pointID string) error {
	file, err := s.findNamedFile(ctx, syncID, "recovery-points/"+pointID+".json")
	if errors.Is(err, errNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return wrapDriveError(s.service.Files.Delete(file.Id).Context(ctx).Do())
}
func (s *Store) ReadManifest(ctx context.Context, syncID, revisionID string) (*syncer.Manifest, error) {
	payload, err := s.readFile(ctx, syncID, "revisions/"+revisionID+".json")
	if err != nil {
		return nil, err
	}
	var value syncer.Manifest
	if err := json.Unmarshal(payload, &value); err != nil || value.ID != revisionID || value.Files == nil {
		return nil, errors.New("Google Drive revision is invalid; retry shortly")
	}
	return &value, nil
}
func (s *Store) WriteManifest(ctx context.Context, syncID string, m *syncer.Manifest) error {
	payload, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.writeNamedFile(ctx, syncID, "revisions/"+m.ID+".json", payload, true)
}
func (s *Store) ReadObject(ctx context.Context, syncID, sha string) (io.ReadCloser, error) {
	return s.downloadNamedFile(ctx, syncID, "objects/"+sha)
}
func (s *Store) WriteObject(ctx context.Context, syncID, sha string, r io.Reader) error {
	if _, err := s.findNamedFile(ctx, syncID, "objects/"+sha); err == nil {
		return nil
	} else if !errors.Is(err, errNotFound) {
		return err
	}
	return s.writeNamedReader(ctx, syncID, "objects/"+sha, r, true)
}
func (s *Store) ArchiveDisplaced(ctx context.Context, syncID, deviceID, relPath string, r io.Reader) error {
	name := fmt.Sprintf("versions/%s/%s-%s", deviceID, time.Now().UTC().Format("20060102T150405.000000000Z"), strings.ReplaceAll(filepath.ToSlash(relPath), "/", "__"))
	return s.writeNamedReader(ctx, syncID, name, r, true)
}

var errNotFound = errors.New("Google Drive file not found")

func (s *Store) readFile(ctx context.Context, syncID, name string) ([]byte, error) {
	in, err := s.downloadNamedFile(ctx, syncID, name)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	return io.ReadAll(in)
}
func (s *Store) downloadNamedFile(ctx context.Context, syncID, name string) (io.ReadCloser, error) {
	file, err := s.findNamedFile(ctx, syncID, name)
	if err != nil {
		return nil, err
	}
	response, err := s.service.Files.Get(file.Id).Context(ctx).Download()
	if err != nil {
		return nil, wrapDriveError(err)
	}
	return response.Body, nil
}

func (s *Store) writeNamedFile(ctx context.Context, syncID, logicalName string, data []byte, immutable bool) error {
	return s.writeNamedReader(ctx, syncID, logicalName, bytes.NewReader(data), immutable)
}

func (s *Store) writeNamedReader(ctx context.Context, syncID, logicalName string, data io.Reader, immutable bool) error {
	parent, name, err := s.parentAndName(ctx, syncID, logicalName)
	if err != nil {
		return err
	}
	existing, findErr := s.findChild(ctx, parent, name)
	if findErr == nil {
		if immutable {
			return nil
		}
		_, err = s.service.Files.Update(existing.Id, &drive.File{
			AppProperties: map[string]string{"cloudSaverType": logicalType(logicalName), "syncId": syncID, "logicalName": logicalName},
		}).Media(data, driveMediaOptions...).Context(ctx).Do()
		return wrapDriveError(err)
	}
	if !errors.Is(findErr, errNotFound) {
		return findErr
	}
	_, err = s.service.Files.Create(&drive.File{
		Name:          name,
		Parents:       []string{parent},
		AppProperties: map[string]string{"cloudSaverType": logicalType(logicalName), "syncId": syncID, "logicalName": logicalName},
	}).Media(data, driveMediaOptions...).Context(ctx).Do()
	return wrapDriveError(err)
}

func logicalType(name string) string {
	switch {
	case name == "game.json":
		return "game_profile"
	case name == "head.json":
		return "head"
	case strings.HasPrefix(name, "revisions/"):
		return "revision"
	case strings.HasPrefix(name, "objects/"):
		return "object"
	case strings.HasPrefix(name, "versions/"):
		return "archive"
	case strings.HasPrefix(name, "recovery-points/"):
		return "recovery_point"
	default:
		return "metadata"
	}
}

func (s *Store) findNamedFile(ctx context.Context, syncID, logicalName string) (*drive.File, error) {
	parent, name, err := s.parentAndName(ctx, syncID, logicalName)
	if err != nil {
		return nil, err
	}
	return s.findChild(ctx, parent, name)
}
func (s *Store) parentAndName(ctx context.Context, syncID, logicalName string) (string, string, error) {
	parts := strings.Split(strings.Trim(logicalName, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", errors.New("invalid Google Drive path")
	}
	segments := []string{".cloud-saver", "v1", "games", syncID}
	segments = append(segments, parts[:len(parts)-1]...)
	parent := "root"
	var err error
	for _, segment := range segments {
		parent, err = s.ensureFolder(ctx, parent, segment)
		if err != nil {
			return "", "", err
		}
	}
	return parent, parts[len(parts)-1], nil
}
func (s *Store) ensureFolder(ctx context.Context, parent, name string) (string, error) {
	key := parent + "/" + name
	s.mu.Lock()
	if id := s.folders[key]; id != "" {
		s.mu.Unlock()
		return id, nil
	}
	s.mu.Unlock()
	file, err := s.findChild(ctx, parent, name)
	if errors.Is(err, errNotFound) {
		file, err = s.service.Files.Create(&drive.File{Name: name, MimeType: folderMIMEType, Parents: []string{parent}}).Context(ctx).Do()
		if err != nil {
			err = wrapDriveError(err)
		}
	}
	if err != nil {
		return "", err
	}
	if file.MimeType != folderMIMEType {
		return "", fmt.Errorf("Google Drive path component %q is not a folder", name)
	}
	s.mu.Lock()
	s.folders[key] = file.Id
	s.mu.Unlock()
	return file.Id, nil
}
func (s *Store) findChild(ctx context.Context, parent, name string) (*drive.File, error) {
	q := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", strings.ReplaceAll(name, "'", "\\'"), parent)
	list, err := s.service.Files.List().Q(q).Fields("files(id,name,mimeType)").PageSize(10).Context(ctx).Do()
	if err != nil {
		return nil, wrapDriveError(err)
	}
	if len(list.Files) == 0 {
		return nil, errNotFound
	}
	return list.Files[0], nil
}

var _ syncer.RevisionStore = (*Store)(nil)

type pacedTransport struct {
	base     http.RoundTripper
	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

func newPacedTransport(base http.RoundTripper, requestsPerSecond int) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if requestsPerSecond < 1 {
		requestsPerSecond = 1
	}
	return &pacedTransport{base: base, interval: time.Second / time.Duration(requestsPerSecond)}
}

func (transport *pacedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.mu.Lock()
	now := time.Now()
	start := now
	if transport.next.After(start) {
		start = transport.next
	}
	transport.next = start.Add(transport.interval)
	transport.mu.Unlock()

	if wait := time.Until(start); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-request.Context().Done():
			return nil, request.Context().Err()
		case <-timer.C:
		}
	}
	return transport.base.RoundTrip(request)
}

func wrapDriveError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return err
	}
	message := strings.ToLower(apiErr.Message + " " + apiErr.Error())
	reasons := make([]string, 0, len(apiErr.Errors))
	for _, item := range apiErr.Errors {
		reasons = append(reasons, strings.ToLower(item.Reason))
	}
	joinedReasons := strings.Join(reasons, " ")
	code, retryable := "provider_unavailable", false
	switch {
	case apiErr.Code == http.StatusUnauthorized || strings.Contains(joinedReasons, "autherror") || strings.Contains(joinedReasons, "invalidcredentials"):
		code = "authentication_required"
	case apiErr.Code == http.StatusTooManyRequests || strings.Contains(joinedReasons, "ratelimit") || strings.Contains(joinedReasons, "userratelimit") || strings.Contains(message, "rate limit"):
		code, retryable = "rate_limited", true
	case strings.Contains(joinedReasons, "storagequota") || strings.Contains(message, "storage quota") || strings.Contains(message, "storagequota"):
		code = "quota_full"
	case apiErr.Code == http.StatusRequestTimeout || apiErr.Code >= 500:
		code, retryable = "provider_unavailable", true
	case apiErr.Code == http.StatusForbidden:
		code = "permission_denied"
	}
	return &syncer.ProviderOperationError{
		Code:       code,
		Retryable:  retryable,
		RetryAfter: parseRetryAfter(apiErr.Header.Get("Retry-After"), time.Now()),
		Err:        err,
	}
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0
	}
	return when.Sub(now)
}
