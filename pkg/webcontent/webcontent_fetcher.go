package webcontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/alantheprice/ledit/pkg/llm"
)

var (
	githubRepoRegex = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)$`)
	githubFileRegex = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$`)
	githubDirRegex  = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/tree/([^/]+)(?:/(.*))?$`)
)

// GithubContentItem represents a file or directory in a GitHub repository.
type GithubContentItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

// WebContentFetcher handles fetching content from URLs.
type WebContentFetcher struct{}

// NewWebContentFetcher creates a new WebContentFetcher instance.
func NewWebContentFetcher() *WebContentFetcher {
	return &WebContentFetcher{}
}

// FetchWebContent fetches content from a given URL, using a cache to avoid refetching.
// It uses Jina Reader for external URLs if available, otherwise falls back to a direct HTTP GET.
func (w *WebContentFetcher) FetchWebContent(url string) (string, error) {
	// Check cache first
	if cachedEntry, found := w.loadURLCache(url); found {
		return cachedEntry.Content, nil
	}

	content, err := w.fetchContent(url)
	if err != nil {
		return "", err
	}

	if err := w.saveURLCache(url, content); err != nil {
		// Log warning but don't fail the operation
		fmt.Printf("Warning: Failed to cache content for URL %s: %v\n", url, err)
	}

	return content, nil
}

// fetchContent determines the best method to fetch content and retrieves it.
func (w *WebContentFetcher) fetchContent(url string) (string, error) {
	if strings.HasPrefix(url, "https://github.com") {
		content, handled, err := w.fetchGitHubContent(url)
		if err != nil {
			return "", fmt.Errorf("error fetching github content for %s: %w", url, err)
		}
		if handled {
			return fmt.Sprintf("\n--- Content from URL: %s ---\n\n%s\n--- End of content from URL: %s ---\n", url, content, url), nil
		}
	}

	isLocalhost := strings.HasPrefix(url, "http://localhost") || strings.HasPrefix(url, "https://localhost")
	jinaAPIKey, err := llm.GetAPIKey("JinaAI")

	useJina := !isLocalhost && err == nil && jinaAPIKey != ""
	if useJina {
		content, err := w.fetchWithJinaReader(url, jinaAPIKey)
		if err != nil {
			return "", fmt.Errorf("failed to fetch with Jina Reader: %w", err)
		}
		return fmt.Sprintf("\n--- Content from URL: %s ---\n\n%s\n--- End of content from URL: %s ---\n", url, content, url), nil
	}

	// Fallback to direct fetch for localhost or if Jina is not configured.
	if !isLocalhost {
		// Get your Jina AI API key for free: https://jina.ai/?sui=apikey
		fmt.Printf("Warning: Jina AI API key not found or provided. Jina Reader will not be used for URL: %s. Falling back to direct HTTP GET. Error: %v\n", url, err)
	}
	return w.fetchDirectURL(url)
}

// fetchGitHubContent handles fetching content from GitHub URLs.
// It returns the content, a boolean indicating if the URL was handled, and an error.
func (w *WebContentFetcher) fetchGitHubContent(u string) (string, bool, error) {
	u = strings.TrimSuffix(u, "/")

	if matches := githubFileRegex.FindStringSubmatch(u); len(matches) > 0 {
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", matches[1], matches[2], matches[3], matches[4])
		content, err := w.fetchDirectURL(rawURL)
		return content, true, err
	}

	if matches := githubDirRegex.FindStringSubmatch(u); len(matches) > 0 {
		owner := matches[1]
		repo := matches[2]
		path := matches[4]
		content, err := w.fetchGitHubDirContent(owner, repo, path)
		return content, true, err
	}

	if matches := githubRepoRegex.FindStringSubmatch(u); len(matches) > 0 {
		owner := matches[1]
		repo := matches[2]
		content, err := w.fetchGitHubReadme(owner, repo)
		return content, true, err
	}

	return "", false, nil // Not a supported GitHub URL type, fall back to Jina/direct fetch.
}

func (w *WebContentFetcher) makeGitHubAPIRequest(url string) (*http.Response, error) {
	githubAPIKey, _ := llm.GetAPIKey("github")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create github api request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if githubAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+githubAPIKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func (w *WebContentFetcher) fetchGitHubReadme(owner, repo string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, repo)
	resp, err := w.makeGitHubAPIRequest(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "README not found.", nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github API returned status %d for readme: %s", resp.StatusCode, string(body))
	}

	var readmeInfo struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&readmeInfo); err != nil {
		return "", fmt.Errorf("failed to decode github readme response: %w", err)
	}

	if readmeInfo.DownloadURL == "" {
		return "", fmt.Errorf("no download_url in github readme response")
	}

	return w.fetchDirectURL(readmeInfo.DownloadURL)
}

func (w *WebContentFetcher) fetchGitHubDirContent(owner, repo, path string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	resp, err := w.makeGitHubAPIRequest(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github API returned status %d for dir contents: %s", resp.StatusCode, string(body))
	}

	var contents []GithubContentItem
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return "", fmt.Errorf("failed to decode github dir contents response: %w", err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Directory listing for %s:\n", path))
	for _, item := range contents {
		builder.WriteString(fmt.Sprintf("- %s (%s)\n", item.Name, item.Type))
	}
	return builder.String(), nil
}

// fetchWithJinaReader fetches content using the Jina Reader API.
func (w *WebContentFetcher) fetchWithJinaReader(url, apiKey string) (string, error) {
	req, err := createJinaRequest(url, apiKey)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request to Jina Reader: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Jina Reader API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return parseJinaResponse(resp.Body)
}

// createJinaRequest creates an HTTP request for the Jina Reader API.
func createJinaRequest(url, apiKey string) (*http.Request, error) {
	const jinaURL = "https://r.jina.ai/"

	requestBody := map[string]string{"url": url}
	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Jina request body: %w", err)
	}

	req, err := http.NewRequest("POST", jinaURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request for Jina Reader: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("X-Engine", "browser")
	req.Header.Set("X-Return-Format", "markdown")
	// req.Header.Set("X-Respond-With", "readerlm-v2") // Use readerlm-v2 for better quality content

	return req, nil
}

// parseJinaResponse decodes the Jina Reader API response and extracts the content.
func parseJinaResponse(body io.Reader) (string, error) {
	var jinaResponse struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.NewDecoder(body).Decode(&jinaResponse); err != nil {
		return "", fmt.Errorf("failed to decode Jina Reader response: %w", err)
	}

	if jinaResponse.Data.Content == "" {
		return "", fmt.Errorf("Jina Reader response did not contain expected 'data.content'")
	}

	return jinaResponse.Data.Content, nil
}

// fetchDirectURL performs a direct HTTP GET request to the given URL.
func (w *WebContentFetcher) fetchDirectURL(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}