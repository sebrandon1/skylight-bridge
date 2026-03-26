package googlephotos

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	tokenURL     = "https://oauth2.googleapis.com/token" //nolint:gosec
	mediaItemURL = "https://photoslibrary.googleapis.com/v1/mediaItems"
)

// MediaItem represents a single item from the Google Photos Library API.
type MediaItem struct {
	ID            string        `json:"id"`
	BaseURL       string        `json:"baseUrl"`
	MimeType      string        `json:"mimeType"`
	Filename      string        `json:"filename"`
	MediaMetadata MediaMetadata `json:"mediaMetadata"`
}

// MediaMetadata holds creation time and dimension info for a media item.
type MediaMetadata struct {
	CreationTime time.Time `json:"creationTime"`
}

// Client handles OAuth2 token refresh and Google Photos API calls.
type Client struct {
	clientID     string
	clientSecret string
	refreshToken string
	httpClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

// NewClient creates a Google Photos client from OAuth2 credentials.
func NewClient(clientID, clientSecret, refreshToken string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: refreshToken,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// ListRecentItems returns the most recent pageSize media items.
func (c *Client) ListRecentItems(pageSize int) ([]MediaItem, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", mediaItemURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	q := req.URL.Query()
	q.Set("pageSize", fmt.Sprintf("%d", pageSize))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing media items: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google photos API returned %d: %s", resp.StatusCode, body)
	}

	var result struct {
		MediaItems []MediaItem `json:"mediaItems"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return result.MediaItems, nil
}

// DownloadImage fetches the raw bytes of a media item at full resolution.
func (c *Client) DownloadImage(item MediaItem) ([]byte, error) {
	// Append =d to get the original download URL
	downloadURL := item.BaseURL + "=d"

	resp, err := c.httpClient.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", item.Filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned %d for %s", resp.StatusCode, item.Filename)
	}

	return io.ReadAll(resp.Body)
}

// ensureToken refreshes the access token if it is missing or expired.
func (c *Client) ensureToken() error {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("refresh_token", c.refreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := c.httpClient.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("refreshing token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh returned %d: %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decoding token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn)*time.Second - 30*time.Second)
	return nil
}
