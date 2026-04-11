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

type SearXNGClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewSearXNGClient(baseURL string) *SearXNGClient {
	return &SearXNGClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type SearchResult struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	Engine        string  `json:"engine"`
	Category      string  `json:"category"`
	Score         float64 `json:"score,omitempty"`
	PublishedDate string  `json:"publishedDate,omitempty"`
}

type SearchResponse struct {
	Query           string         `json:"query"`
	NumberOfResults int            `json:"number_of_results"`
	Results         []SearchResult `json:"results"`
	Answers         []string       `json:"answers,omitempty"`
	Corrections     []string       `json:"corrections,omitempty"`
	Infoboxes       []interface{}  `json:"infoboxes,omitempty"`
	Suggestions     []string       `json:"suggestions,omitempty"`
}

type SearchParams struct {
	Query      string
	Categories []string
	Engines    []string
	Language   string
	PageNo     int
	TimeRange  string
	SafeSearch int
}

func (c *SearXNGClient) Search(params SearchParams) (*SearchResponse, error) {
	searchURL := fmt.Sprintf("%s/search", c.BaseURL)

	values := url.Values{}
	values.Set("q", params.Query)
	values.Set("format", "json")

	if len(params.Categories) > 0 {
		values.Set("categories", strings.Join(params.Categories, ","))
	}

	if len(params.Engines) > 0 {
		values.Set("engines", strings.Join(params.Engines, ","))
	}

	if params.Language != "" {
		values.Set("language", params.Language)
	}

	if params.PageNo > 0 {
		values.Set("pageno", strconv.Itoa(params.PageNo))
	}

	if params.TimeRange != "" {
		values.Set("time_range", params.TimeRange)
	}

	if params.SafeSearch >= 0 && params.SafeSearch <= 2 {
		values.Set("safesearch", strconv.Itoa(params.SafeSearch))
	}

	// Use POST request with form data (required by SearXNG)
	req, err := http.NewRequest("POST", searchURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "MCP-SearXNG-Client/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")

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

	var searchResponse SearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return &searchResponse, nil
}

func (c *SearXNGClient) GetEngines() (map[string]interface{}, error) {
	enginesURL := fmt.Sprintf("%s/config", c.BaseURL)

	req, err := http.NewRequest("GET", enginesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "MCP-SearXNG-Client/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	return config, nil
}

// Ping checks if SearXNG is reachable and responding
func (c *SearXNGClient) Ping() error {
	enginesURL := fmt.Sprintf("%s/config", c.BaseURL)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(enginesURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// WaitForReady polls SearXNG until it's ready or timeout is reached
func (c *SearXNGClient) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	checkInterval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		if err := c.Ping(); err == nil {
			return nil
		}
		time.Sleep(checkInterval)
	}

	return fmt.Errorf("SearXNG failed to become ready within %v", timeout)
}
