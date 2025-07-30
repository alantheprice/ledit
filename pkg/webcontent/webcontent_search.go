package webcontent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/llm"
	"github.com/alantheprice/ledit/pkg/prompts"
	"github.com/alantheprice/ledit/pkg/utils"
)

const githubSearchURL = "https://api.github.com/search/code"

func FetchContextFromSearch(query string, cfg *config.Config) (string, error) {
	logger := utils.GetLogger(cfg.SkipPrompt)
	logger.LogProcessStep(fmt.Sprintf("Starting web content search for query: %s", query))
	defer logger.LogProcessStep("Completed web content search")

	if strings.TrimSpace(query) == "" {
		logger.Log("No relevant content found for the query")
		return "", nil
	}

	// Fetch search results and content using GitHub Search API
	results, err := fetchGithubSearchResults(query, cfg)
	if err != nil {
		logger.Logf("Error fetching GitHub search results: %v", err)
		return "", fmt.Errorf("failed to fetch GitHub search results: %w", err)
	}

	if len(results) == 0 {
		logger.Log("No relevant content found for the query")
		return "", nil
	}

	logger.Logf("Found %d relevant content items", len(results))
	var sb strings.Builder
	for url, content := range results {
		sb.WriteString(fmt.Sprintf("URL: %s\nContent:\n%s\n\n", url, content))
	}

	return sb.String(), nil
}

func getSearchResults(query string, cfg *config.Config) ([]JinaSearchResult, error) {
	fetcher := NewWebContentFetcher()
	logger := utils.GetLogger(cfg.SkipPrompt)
	startTime := time.Now()
	defer func() {
		logger.Logf("GitHub search results fetch completed in %v", time.Since(startTime))
	}()

	logger.LogProcessStep("Checking for cached search results")
	if cachedEntry, err := fetcher.loadReferenceCache(query); err == nil {
		logger.Logf("Using cached search results (age: %v)", time.Since(cachedEntry.Timestamp))
		return cachedEntry.SearchResults, nil
	} else {
		logger.Logf("Cache check result: %v", err)
	}

	// Get GitHub API Key.
	githubAPIKey, err := llm.GetAPIKey("github")
	if err != nil {
		logger.Logf("Could not get GitHub API key: %v. Proceeding without it, but may be rate limited.", err)
	} else {
		logger.Log("Using GitHub API key for search")
	}

	logger.LogProcessStep(fmt.Sprintf("Performing GitHub Code search for query: %s", query))
	req, err := http.NewRequest("GET", githubSearchURL, nil)
	if err != nil {
		logger.Logf("Failed to create GitHub request: %v", err)
		return nil, fmt.Errorf("failed to create github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.text-match+json")
	if githubAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+githubAPIKey)
	}
	q := req.URL.Query()
	q.Add("q", query)
	req.URL.RawQuery = q.Encode()

	// Increase the timeout for search grounding
	client := &http.Client{Timeout: 120 * time.Second}
	logger.Logf("Making HTTP request to GitHub API: %s", req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		logger.Logf("GitHub search request failed: %v", err)
		return nil, fmt.Errorf("failed to perform github search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Logf("Failed to read GitHub response body: %v", err)
		return nil, fmt.Errorf("failed to read github response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Logf("GitHub search API returned status %d. Body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("github search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResponse struct {
		Items []GithubSearchResult `json:"items"`
	}
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		logger.Logf("Failed to unmarshal GitHub response: %v", err)
		return nil, fmt.Errorf("failed to unmarshal github response: %w", err)
	}

	logger.Logf("Received %d search results from GitHub", len(searchResponse.Items))

	var jinaResults []JinaSearchResult
	for _, item := range searchResponse.Items {
		var description strings.Builder
		for _, match := range item.TextMatches {
			description.WriteString(match.Fragment)
			description.WriteString("\n")
		}
		jinaResults = append(jinaResults, JinaSearchResult{
			Title:       fmt.Sprintf("%s: %s", item.Repository.FullName, item.Path),
			URL:         item.HTMLURL,
			Description: description.String(),
		})
	}

	return jinaResults, nil
}

// fetchGithubSearchResults fetches search results using GitHub Search API,
// selects relevant URLs using LLM, and fetches full content of selected URLs.
// uses embeddings to find the most relevant parts of the text for a given query.
func fetchGithubSearchResults(query string, cfg *config.Config) (map[string]string, error) {
	fetcher := NewWebContentFetcher()
	logger := utils.GetLogger(cfg.SkipPrompt)
	searchResponse, err := getSearchResults(query, cfg)
	if err != nil {
		logger.Logf("Failed to get search results: %v", err)
		return nil, fmt.Errorf("failed to get search results: %w", err)
	}

	selectedURLs, err := selectRelevantURLsWithLLM(query, searchResponse, cfg)
	if err != nil {
		logger.Logf("URL selection failed: %v", err)
		return nil, fmt.Errorf("failed to select relevant URLs: %w", err)
	}

	if len(selectedURLs) == 0 {
		logger.Log("No relevant URLs selected by LLM")
		return make(map[string]string), nil
	}

	logger.Logf("LLM selected %d URLs: %v", len(selectedURLs), selectedURLs)
	var wg sync.WaitGroup
	var mu sync.Mutex
	fetchedContent := make(map[string]string)

	for _, url := range selectedURLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			logger.LogProcessStep(fmt.Sprintf("Fetching content from %s", url))
			content, err := fetcher.FetchWebContent(url)
			if err != nil {
				logger.Logf("Failed to fetch content from %s: %v", url, err)
				return
			}

			logger.LogProcessStep(fmt.Sprintf("Extracting relevant content from %s for query: %s", url, query))
			relevantContent, err := GetRelevantContentFromText(query, content, cfg)
			if err != nil {
				logger.Logf("Failed to get relevant content from %s: %v. Using full content as fallback.", url, err)
				mu.Lock()
				fetchedContent[url] = content
				mu.Unlock()
				return
			}

			logger.Logf("Extracted %d characters of relevant content from %s", len(relevantContent), url)
			mu.Lock()
			fetchedContent[url] = relevantContent
			mu.Unlock()
		}(url)
	}

	wg.Wait()
	logger.Logf("Successfully fetched content from %d/%d URLs", len(fetchedContent), len(selectedURLs))

	cacheEntry := &ReferenceCacheEntry{
		Query:          query,
		SearchResults:  searchResponse,
		SelectedURLs:   selectedURLs,
		FetchedContent: fetchedContent,
		Timestamp:      time.Now(),
	}
	if err := fetcher.saveReferenceCache(query, cacheEntry); err != nil {
		logger.Logf("Failed to save reference cache: %v", err)
	} else {
		logger.Log("Successfully cached search results")
	}

	return fetchedContent, nil
}

func selectRelevantURLsWithLLM(query string, results []JinaSearchResult, cfg *config.Config) ([]string, error) {
	logger := utils.GetLogger(cfg.SkipPrompt)
	logger.LogProcessStep("Selecting relevant URLs with LLM")
	defer logger.LogProcessStep("Completed URL selection")

	var sb strings.Builder
	sb.WriteString("Search Query: ")
	sb.WriteString(query)
	sb.WriteString("\n\nSearch Results:\n")
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. URL: %s\n   Title: %s\n   Description: %s\n", i+1, r.URL, r.Title, r.Description))
	}

	messages := prompts.BuildSearchResultsQueryMessages(sb.String(), query)

	logger.Log("Sending URL selection request to LLM")
	_, resp, err := llm.GetLLMResponse(cfg.WorkspaceModel, messages, "search_results_selector", cfg, 2*time.Minute)
	if err != nil {
		logger.Logf("LLM URL selection failed: %v", err)
		return nil, err
	}

	logger.Logf("Received LLM response for URL selection: %s", resp)
	if strings.ToLower(strings.TrimSpace(resp)) == "none" {
		logger.Log("LLM determined no relevant URLs")
		return []string{}, nil
	}

	var selectedURLs []string
	parts := strings.Split(resp, ",")
	for _, p := range parts {
		var num int
		_, err := fmt.Sscanf(strings.TrimSpace(p), "%d", &num)
		if err == nil && num > 0 && num <= len(results) {
			selectedURLs = append(selectedURLs, results[num-1].URL)
		}
	}

	return selectedURLs, nil
}