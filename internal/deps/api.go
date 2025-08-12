package deps

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/package-url/packageurl-go"
)

const (
	depsDevAPIBase = "https://api.deps.dev/v3"
	requestTimeout = 30 * time.Second
)

// Version represents a package version with publication date
type Version struct {
	Version     string `json:"version" bson:"version"`
	PublishedAt string `json:"publishedAt" bson:"publishedAt"`
}

// Time parses the PublishedAt field as RFC3339 time
func (v *Version) Time() (time.Time, error) {
	if v.PublishedAt == "" {
		return time.Time{}, fmt.Errorf("empty publication date")
	}
	return time.Parse(time.RFC3339, v.PublishedAt)
}

// APIResponse represents the response from the deps.dev API
type APIResponse struct {
	Versions []VersionsAPIResponse `json:"versions"`
}

// VersionsAPIResponse represents a version entry in the API response
type VersionsAPIResponse struct {
	Version     Version `json:"versionKey"`
	PublishedAt string  `json:"publishedAt" bson:"publishedAt"`
}

// Client provides access to the deps.dev API
type Client struct {
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new deps.dev API client
func NewClient(logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		logger: logger,
	}
}

// GetVersions retrieves all versions for a package identified by its PURL
func (c *Client) GetVersions(ctx context.Context, rawPURL string) (*APIResponse, error) {
	purl, err := packageurl.FromString(rawPURL)
	if err != nil {
		c.logger.Warn("Failed to parse PURL", "purl", rawPURL, "error", err)
		return nil, fmt.Errorf("invalid PURL %q: %w", rawPURL, err)
	}

	c.logger.Debug("Starting deps.dev API query", "purl", purl.String())

	name, system, err := c.getNameAndSystem(purl)
	if err != nil {
		return nil, fmt.Errorf("failed to extract name and system from PURL: %w", err)
	}

	apiURL := fmt.Sprintf("%s/systems/%s/packages/%s", depsDevAPIBase, system, name)
	c.logger.Debug("Constructed API URL", "url", apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("HTTP request failed", "url", apiURL, "error", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("Failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp, apiURL)
	}

	var depsResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&depsResp); err != nil {
		c.logger.Debug("Failed to decode response", "url", apiURL, "error", err)
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	c.logger.Debug("Successfully retrieved versions", "url", apiURL, "count", len(depsResp.Versions))
	return &depsResp, nil
}

// handleHTTPError processes non-200 HTTP responses and returns appropriate errors
func (c *Client) handleHTTPError(resp *http.Response, url string) error {
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		c.logger.Debug("Rate limited by API", "retry-after", retryAfter, "url", url)
		return fmt.Errorf("rate limited (retry after: %s)", retryAfter)
	case http.StatusNotFound:
		c.logger.Debug("Package not found", "url", url)
		return fmt.Errorf("package not found")
	case http.StatusBadRequest:
		c.logger.Debug("Bad request", "url", url)
		return fmt.Errorf("invalid request")
	default:
		c.logger.Debug("HTTP request failed", "url", url, "status", resp.StatusCode)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
}

// getNameAndSystem extracts the package name and system from a PURL
func (c *Client) getNameAndSystem(purl packageurl.PackageURL) (name, system string, err error) {
	name = purl.Name
	if purl.Namespace != "" {
		name = purl.Namespace + "/" + name
	}

	switch purl.Type {
	case packageurl.TypeNPM:
		system = "npm"
	case packageurl.TypeGolang:
		system = "go"
	case packageurl.TypeCargo:
		system = "cargo"
	case packageurl.TypeMaven, packageurl.TypeGradle:
		system = "maven"
		if purl.Namespace != "" {
			name = purl.Namespace + ":" + purl.Name
		} else {
			name = purl.Name
		}
	case packageurl.TypePyPi:
		system = "pypi"
	case packageurl.TypeNuget:
		system = "nuget"
	case packageurl.TypeGem:
		system = "rubygems"
	default:
		return "", "", fmt.Errorf("unsupported package type: %s", purl.Type)
	}

	// URL encode the name and system to handle special characters
	name = url.PathEscape(name)
	system = url.PathEscape(system)

	if name == "" || system == "" {
		return "", "", fmt.Errorf("failed to extract name or system from PURL: name=%q, system=%q", name, system)
	}

	return name, system, nil
}

// GetVersions provides a convenient function to get versions using the default client
func GetVersions(ctx context.Context, rawPURL string) (*APIResponse, error) {
	client := NewClient(slog.Default())
	return client.GetVersions(ctx, rawPURL)
}

// Deprecated: GetVersions without context - use GetVersions with context instead
func GetVersionsWithoutContext(rawPURL string) (*APIResponse, error) {
	return GetVersions(context.Background(), rawPURL)
}
