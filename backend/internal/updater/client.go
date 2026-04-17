package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const latestReleaseAPIURL = "https://api.github.com/repos/njnis/cloud-saver/releases/latest"

// stores latest release metadata returned by the remote update provider.
type Release struct {
	TagName string
	HTMLURL string
}

// wraps update lookups using a timeout-bound http client.
type Client struct {
	httpClient *http.Client
	apiURL     string
}

// builds a "Client" configured for the project's latest release endpoint.
func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		apiURL:     latestReleaseAPIURL,
	}
}

// fetches and validates the latest release payload from github.
func (c *Client) LatestRelease(ctx context.Context) (Release, error) {
	if c == nil || c.httpClient == nil {
		return Release{}, errors.New("update client is not initialized")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cloud-saver-daemon")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("update endpoint returned %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Release{}, err
	}

	release := Release{
		TagName: strings.TrimSpace(payload.TagName),
		HTMLURL: strings.TrimSpace(payload.HTMLURL),
	}
	if release.TagName == "" || release.HTMLURL == "" {
		return Release{}, errors.New("latest release payload is missing required fields")
	}
	return release, nil
}
