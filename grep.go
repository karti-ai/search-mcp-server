package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type GrepAppClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewGrepAppClient() *GrepAppClient {
	return &GrepAppClient{
		BaseURL: "https://grep.app",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type GrepSearchResult struct {
	Repo     string `json:"repo"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
	Language string `json:"language,omitempty"`
	Lines    []struct {
		Content    string `json:"content"`
		LineNumber int    `json:"line_number"`
	} `json:"lines,omitempty"`
	URL      string `json:"url"`
	RepoURL  string `json:"repo_url"`
	BlobURL  string `json:"blob_url"`
}

type GrepSearchResponse struct {
	Hits struct {
		Total    int               `json:"total"`
		Hits     []GrepSearchHit   `json:"hits"`
		Metadata map[string]int    `json:"metadata,omitempty"`
	} `json:"hits"`
}

type GrepSearchHit struct {
	Source GrepSearchResult `json:"_source"`
}

type GrepSearchParams struct {
	Query        string
	Language     string
	Repo         string
	Path         string
	Regex        bool
	CaseSensitive bool
	WholeWords   bool
}

func (c *GrepAppClient) Search(params GrepSearchParams) (*GrepSearchResponse, error) {
	searchURL := fmt.Sprintf("%s/search", c.BaseURL)

	values := url.Values{}
	values.Set("q", params.Query)
	values.Set("limit", "25")

	if params.Language != "" {
		values.Set("lang", params.Language)
	}
	if params.Repo != "" {
		values.Set("repo", params.Repo)
	}
	if params.Path != "" {
		values.Set("path", params.Path)
	}
	if params.Regex {
		values.Set("regex", "true")
	}
	if params.CaseSensitive {
		values.Set("case", "true")
	}
	if params.WholeWords {
		values.Set("word", "true")
	}

	req, err := http.NewRequest("GET", searchURL+"?"+values.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Go-GrepApp-MCP/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var searchResponse GrepSearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return &searchResponse, nil
}

func (c *GrepAppClient) GetFile(owner, repo, path, ref string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", owner, path)
	if ref != "" {
		apiURL += "?ref=" + ref
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Go-GrepApp-MCP/1.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var fileData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fileData); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	content, ok := fileData["content"].(string)
	if !ok {
		return "", fmt.Errorf("no content found in response")
	}

	decoded, err := decodeBase64(content)
	if err != nil {
		return "", fmt.Errorf("error decoding content: %w", err)
	}

	return decoded, nil
}

func decodeBase64(encoded string) (string, error) {
	encoded = strings.ReplaceAll(encoded, "\n", "")
	decoded, err := strconv.Unquote(`"` + encoded + `"`)
	if err != nil {
		return encoded, nil
	}
	return decoded, nil
}

type GitHubFileParams struct {
	Owner string
	Repo  string
	Path  string
	Ref   string
}

func (c *GrepAppClient) GetFilesBatch(files []GitHubFileParams) ([]map[string]string, error) {
	results := make([]map[string]string, 0, len(files))

	for _, file := range files {
		content, err := c.GetFile(file.Owner, file.Repo, file.Path, file.Ref)
		if err != nil {
			results = append(results, map[string]string{
				"owner": file.Owner,
				"repo":  file.Repo,
				"path":  file.Path,
				"error": err.Error(),
			})
			continue
		}

		results = append(results, map[string]string{
			"owner":   file.Owner,
			"repo":    file.Repo	,
			"path":    file.Path,
			"content": content,
		})
	}

	return results, nil
}