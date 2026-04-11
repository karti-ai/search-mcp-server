package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var searxngClient *SearXNGClient
var grepAppClient *GrepAppClient
var dockerManager *DockerManager

func main() {
	var transport string
	var host string
	var port string
	var searxngURL string
	var searxngPort string
	var searxngImage string
	var autostart bool
	var stopOnExit bool
	var searxngTimeout int

	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&host, "h", "0.0.0.0", "Host of sse server")
	flag.StringVar(&port, "p", "8892", "Port of sse server")
	flag.StringVar(&searxngURL, "searxng", "http://127.0.0.1:8080", "SearXNG instance URL (overrides -searxng-port)")
	flag.StringVar(&searxngPort, "searxng-port", "8080", "Port for SearXNG container")
	flag.StringVar(&searxngImage, "searxng-image", SearXNGImage, "Docker image for SearXNG")
	flag.BoolVar(&autostart, "searxng-autostart", true, "Auto-start SearXNG Docker container if not running")
	flag.BoolVar(&stopOnExit, "searxng-stop-on-exit", false, "Stop SearXNG container when MCP server exits")
	flag.IntVar(&searxngTimeout, "searxng-timeout", 60, "Timeout in seconds to wait for SearXNG to become ready")
	flag.Parse()

	// Initialize clients
	searxngClient = NewSearXNGClient(searxngURL)
	grepAppClient = NewGrepAppClient()
	dockerManager = NewDockerManager(searxngPort, searxngImage, autostart, stopOnExit)

	// Clean up on exit
	defer dockerManager.Close()

	// Version 2.0.0 - with auto-managed SearXNG
	mcpServer := server.NewMCPServer(
		"search-mcp",
		"2.0.0",
	)

	// SearXNG Search Tools
	searchTool := mcp.NewTool("searxng_search",
		mcp.WithDescription("Search information through SearXNG. Supports various categories and search engines. Auto-starts SearXNG if not running."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query"),
		),
		mcp.WithString("categories",
			mcp.Description("Search categories (general, images, videos, news, music, files, science, it). Multiple values separated by comma"),
		),
		mcp.WithString("engines",
			mcp.Description("Search engines (google, bing, duckduckgo, yandex, etc.). Multiple values separated by comma"),
		),
		mcp.WithString("language",
			mcp.Description("Search language (ru, en, de, fr, etc.)"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number of results (default 1)"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range (day, week, month, year)"),
		),
		mcp.WithNumber("safe_search",
			mcp.Description("Safe search (0 - disabled, 1 - moderate, 2 - strict)"),
		),
	)
	mcpServer.AddTool(searchTool, searxngSearchHandler)

	enginesInfoTool := mcp.NewTool("searxng_engines_info",
		mcp.WithDescription("Get information about available SearXNG search engines and categories"),
	)
	mcpServer.AddTool(enginesInfoTool, searxngEnginesInfoHandler)

	imageSearchTool := mcp.NewTool("searxng_image_search",
		mcp.WithDescription("Specialized image search through SearXNG. Auto-starts SearXNG if not running."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for images"),
		),
		mcp.WithString("engines",
			mcp.Description("Image search engines (google images, bing images, flickr, etc.)"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number of results"),
		),
	)
	mcpServer.AddTool(imageSearchTool, searxngImageSearchHandler)

	newsSearchTool := mcp.NewTool("searxng_news_search",
		mcp.WithDescription("Specialized news search through SearXNG. Auto-starts SearXNG if not running."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for news"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range for news (day, week, month, year)"),
		),
		mcp.WithString("language",
			mcp.Description("News language"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number of results"),
		),
	)
	mcpServer.AddTool(newsSearchTool, searxngNewsSearchHandler)

	// Code Search Tools (via Grep.app)
	grepSearchTool := mcp.NewTool("grep_search",
		mcp.WithDescription("Search code across public GitHub repositories using grep.app. Find real-world code examples, patterns, and implementations."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query string - code pattern to search for"),
		),
		mcp.WithString("language",
			mcp.Description("Programming language filter (e.g., Go, Python, JavaScript, TypeScript)"),
		),
		mcp.WithString("repo",
			mcp.Description("Repository filter in format 'owner/repo' (e.g., 'vercel/next.js')"),
		),
		mcp.WithString("path",
			mcp.Description("Path filter for specific directories (e.g., 'src/', 'lib/')"),
		),
		mcp.WithBoolean("regex",
			mcp.Description("Treat query as regex pattern"),
		),
		mcp.WithBoolean("caseSensitive",
			mcp.Description("Case-sensitive search"),
		),
		mcp.WithBoolean("wholeWords",
			mcp.Description("Search for whole words only"),
		),
	)
	mcpServer.AddTool(grepSearchTool, grepSearchHandler)

	githubFileTool := mcp.NewTool("github_file",
		mcp.WithDescription("Fetch a single file from a GitHub repository"),
		mcp.WithString("owner",
			mcp.Required(),
			mcp.Description("Repository owner (e.g., 'vercel', 'facebook')"),
		),
		mcp.WithString("repo",
			mcp.Required(),
			mcp.Description("Repository name (e.g., 'next.js', 'react')"),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("File path (e.g., 'package.json', 'src/index.ts')"),
		),
		mcp.WithString("ref",
			mcp.Description("Branch, tag, or commit SHA (default: default branch)"),
		),
	)
	mcpServer.AddTool(githubFileTool, githubFileHandler)

	githubBatchFilesTool := mcp.NewTool("github_batch_files",
		mcp.WithDescription("Fetch multiple files from GitHub repositories in parallel"),
		mcp.WithArray("files",
			mcp.Required(),
			mcp.Description("Array of file objects with owner, repo, path, and optional ref"),
		),
	)
	mcpServer.AddTool(githubBatchFilesTool, githubBatchFilesHandler)

	// New v2.0.0 Management Tools
	searchStatusTool := mcp.NewTool("search_status",
		mcp.WithDescription("Check the status of search services: SearXNG, Docker, and code search availability"),
	)
	mcpServer.AddTool(searchStatusTool, searchStatusHandler)

	searxngStopTool := mcp.NewTool("searxng_stop",
		mcp.WithDescription("Stop the SearXNG Docker container to free resources"),
	)
	mcpServer.AddTool(searxngStopTool, searxngStopHandler)

	searxngStartTool := mcp.NewTool("searxng_start",
		mcp.WithDescription("Start the SearXNG Docker container manually"),
	)
	mcpServer.AddTool(searxngStartTool, searxngStartHandler)

	searxngConfigTool := mcp.NewTool("searxng_config",
		mcp.WithDescription("Get the default SearXNG configuration settings used by this MCP server"),
	)
	mcpServer.AddTool(searxngConfigTool, searxngConfigHandler)

	// Start server
	if transport == "sse" {
		sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL(fmt.Sprintf("http://localhost:%s", port)))
		log.Printf("SSE server listening on %s:%s URL: http://127.0.0.1:%s/sse", host, port, port)
		log.Printf("SearXNG: auto-start=%v, stop-on-exit=%v", autostart, stopOnExit)
		if err := sseServer.Start(fmt.Sprintf("%s:%s", host, port)); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Printf("Stdio server started. SearXNG: auto-start=%v, stop-on-exit=%v", autostart, stopOnExit)
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// ensureSearXNG ensures SearXNG is running before performing a search
func ensureSearXNG() error {
	if dockerManager == nil {
		return fmt.Errorf("docker manager not initialized")
	}

	// Fast path: already running and healthy
	if dockerManager.IsSearXNGRunning() && dockerManager.GetSearXNGHealth() {
		return nil
	}

	// Try to ensure it's running
	if err := dockerManager.EnsureSearXNG(); err != nil {
		return err
	}

	// Wait for it to be ready
	return dockerManager.WaitForSearXNGReady(30 * time.Second)
}

func searxngSearchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Ensure SearXNG is running
	if err := ensureSearXNG(); err != nil {
		return nil, fmt.Errorf("SearXNG not available: %w", err)
	}

	query, ok := request.Params.Arguments["query"].(string)
	if !ok {
		return nil, errors.New("query must be a string")
	}

	params := SearchParams{
		Query:      query,
		Categories: []string{"general"},
		Engines:    []string{"duckduckgo", "brave"},
		Language:   "en",
	}

	if categories, ok := request.Params.Arguments["categories"].(string); ok && categories != "" {
		params.Categories = strings.Split(categories, ",")
		for i := range params.Categories {
			params.Categories[i] = strings.TrimSpace(params.Categories[i])
		}
	}

	if engines, ok := request.Params.Arguments["engines"].(string); ok && engines != "" {
		params.Engines = strings.Split(engines, ",")
		for i := range params.Engines {
			params.Engines[i] = strings.TrimSpace(params.Engines[i])
		}
	}

	if language, ok := request.Params.Arguments["language"].(string); ok && language != "" {
		params.Language = language
	}

	if pageFloat, ok := request.Params.Arguments["page"].(float64); ok {
		params.PageNo = int(pageFloat)
	}

	if timeRange, ok := request.Params.Arguments["time_range"].(string); ok {
		params.TimeRange = timeRange
	}

	if safeSearchFloat, ok := request.Params.Arguments["safe_search"].(float64); ok {
		params.SafeSearch = int(safeSearchFloat)
	}

	result, err := searxngClient.Search(params)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}

	response := map[string]interface{}{
		"query":             result.Query,
		"number_of_results": result.NumberOfResults,
		"results":           result.Results,
	}

	if len(result.Answers) > 0 {
		response["answers"] = result.Answers
	}
	if len(result.Suggestions) > 0 {
		response["suggestions"] = result.Suggestions
	}
	if len(result.Corrections) > 0 {
		response["corrections"] = result.Corrections
	}

	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

func searxngEnginesInfoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Ensure SearXNG is running
	if err := ensureSearXNG(); err != nil {
		return nil, fmt.Errorf("SearXNG not available: %w", err)
	}

	config, err := searxngClient.GetEngines()
	if err != nil {
		return nil, fmt.Errorf("error getting engines information: %w", err)
	}

	jsonResult, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

func searxngImageSearchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Ensure SearXNG is running
	if err := ensureSearXNG(); err != nil {
		return nil, fmt.Errorf("SearXNG not available: %w", err)
	}

	query, ok := request.Params.Arguments["query"].(string)
	if !ok {
		return nil, errors.New("query must be a string")
	}

	params := SearchParams{
		Query:      query,
		Categories: []string{"images"},
		Engines:    []string{"bing images"},
		Language:   "en",
	}

	if engines, ok := request.Params.Arguments["engines"].(string); ok && engines != "" {
		params.Engines = strings.Split(engines, ",")
		for i := range params.Engines {
			params.Engines[i] = strings.TrimSpace(params.Engines[i])
		}
	}

	if pageFloat, ok := request.Params.Arguments["page"].(float64); ok {
		params.PageNo = int(pageFloat)
	}

	result, err := searxngClient.Search(params)
	if err != nil {
		return nil, fmt.Errorf("image search error: %w", err)
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

func searxngNewsSearchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Ensure SearXNG is running
	if err := ensureSearXNG(); err != nil {
		return nil, fmt.Errorf("SearXNG not available: %w", err)
	}

	query, ok := request.Params.Arguments["query"].(string)
	if !ok {
		return nil, errors.New("query must be a string")
	}

	params := SearchParams{
		Query:      query,
		Categories: []string{"news"},
		Engines:    []string{"bing news"},
		Language:   "en",
	}

	if timeRange, ok := request.Params.Arguments["time_range"].(string); ok {
		params.TimeRange = timeRange
	}

	if language, ok := request.Params.Arguments["language"].(string); ok && language != "" {
		params.Language = language
	}

	if pageFloat, ok := request.Params.Arguments["page"].(float64); ok {
		params.PageNo = int(pageFloat)
	}

	result, err := searxngClient.Search(params)
	if err != nil {
		return nil, fmt.Errorf("news search error: %w", err)
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

func grepSearchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := request.Params.Arguments["query"].(string)
	if !ok {
		return nil, errors.New("query must be a string")
	}

	params := GrepSearchParams{
		Query: query,
	}

	if language, ok := request.Params.Arguments["language"].(string); ok && language != "" {
		params.Language = language
	}

	if repo, ok := request.Params.Arguments["repo"].(string); ok && repo != "" {
		params.Repo = repo
	}

	if path, ok := request.Params.Arguments["path"].(string); ok && path != "" {
		params.Path = path
	}

	if regex, ok := request.Params.Arguments["regex"].(bool); ok {
		params.Regex = regex
	}

	if caseSensitive, ok := request.Params.Arguments["caseSensitive"].(bool); ok {
		params.CaseSensitive = caseSensitive
	}

	if wholeWords, ok := request.Params.Arguments["wholeWords"].(bool); ok {
		params.WholeWords = wholeWords
	}

	result, err := grepAppClient.Search(params)
	if err != nil {
		return nil, fmt.Errorf("grep search error: %w", err)
	}

	response := map[string]interface{}{
		"query":      params.Query,
		"total_hits": result.Hits.Total,
	}

	var results []map[string]interface{}
	for _, hit := range result.Hits.Hits {
		r := map[string]interface{}{
			"repo":     hit.Source.Repo,
			"path":     hit.Source.Path,
			"filename": hit.Source.Filename,
			"language": hit.Source.Language,
			"url":      hit.Source.URL,
		}

		if len(hit.Source.Lines) > 0 {
			var lines []map[string]interface{}
			for _, line := range hit.Source.Lines {
				lines = append(lines, map[string]interface{}{
					"content":     line.Content,
					"line_number": line.LineNumber,
				})
			}
			r["lines"] = lines
		}

		results = append(results, r)
	}

	response["results"] = results

	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

func githubFileHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	owner, ok := request.Params.Arguments["owner"].(string)
	if !ok {
		return nil, errors.New("owner is required")
	}

	repo, ok := request.Params.Arguments["repo"].(string)
	if !ok {
		return nil, errors.New("repo is required")
	}

	path, ok := request.Params.Arguments["path"].(string)
	if !ok {
		return nil, errors.New("path is required")
	}

	ref := ""
	if refVal, ok := request.Params.Arguments["ref"].(string); ok {
		ref = refVal
	}

	content, err := grepAppClient.GetFile(owner, repo, path, ref)
	if err != nil {
		return nil, fmt.Errorf("error fetching file: %w", err)
	}

	response := map[string]interface{}{
		"owner":   owner,
		"repo":    repo,
		"path":    path,
		"ref":     ref,
		"content": content,
	}

	jsonResult, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

func githubBatchFilesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filesArg, ok := request.Params.Arguments["files"].([]interface{})
	if !ok {
		return nil, errors.New("files must be an array")
	}

	var files []GitHubFileParams
	for _, f := range filesArg {
		fileMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}

		file := GitHubFileParams{}
		if owner, ok := fileMap["owner"].(string); ok {
			file.Owner = owner
		}
		if repo, ok := fileMap["repo"].(string); ok {
			file.Repo = repo
		}
		if path, ok := fileMap["path"].(string); ok {
			file.Path = path
		}
		if ref, ok := fileMap["ref"].(string); ok {
			file.Ref = ref
		}

		if file.Owner != "" && file.Repo != "" && file.Path != "" {
			files = append(files, file)
		}
	}

	if len(files) == 0 {
		return nil, errors.New("no valid files provided")
	}

	results, err := grepAppClient.GetFilesBatch(files)
	if err != nil {
		return nil, fmt.Errorf("error fetching files: %w", err)
	}

	jsonResult, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

// v2.0.0 Management Handlers

func searchStatusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := map[string]interface{}{
		"version":   "2.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Docker status
	if dockerManager != nil {
		dockerAvailable := dockerManager.IsDockerAvailable()
		status["docker_available"] = dockerAvailable

		if dockerAvailable {
			status["searxng_running"] = dockerManager.IsSearXNGRunning()
			status["searxng_healthy"] = dockerManager.GetSearXNGHealth()
			status["searxng_port"] = dockerManager.searxngPort
			status["searxng_image"] = dockerManager.searxngImage
		}
	}

	// Test code search (lightweight check)
	status["code_search_available"] = grepAppClient != nil

	jsonResult, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}

func searxngStopHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if dockerManager == nil {
		return nil, errors.New("docker manager not initialized")
	}

	if !dockerManager.IsDockerAvailable() {
		return nil, errors.New("Docker is not available")
	}

	if !dockerManager.IsSearXNGRunning() {
		return mcp.NewToolResultText("SearXNG is not currently running"), nil
	}

	if err := dockerManager.StopSearXNG(); err != nil {
		return nil, fmt.Errorf("failed to stop SearXNG: %w", err)
	}

	return mcp.NewToolResultText("SearXNG stopped successfully"), nil
}

func searxngStartHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if dockerManager == nil {
		return nil, errors.New("docker manager not initialized")
	}

	if !dockerManager.IsDockerAvailable() {
		return nil, errors.New("Docker is not available. Please install Docker to use search.")
	}

	if dockerManager.IsSearXNGRunning() && dockerManager.GetSearXNGHealth() {
		return mcp.NewToolResultText("SearXNG is already running and healthy"), nil
	}

	if err := dockerManager.EnsureSearXNG(); err != nil {
		return nil, fmt.Errorf("failed to start SearXNG: %w", err)
	}

	if err := dockerManager.WaitForSearXNGReady(30 * time.Second); err != nil {
		return nil, fmt.Errorf("SearXNG started but failed to become ready: %w", err)
	}

	return mcp.NewToolResultText("SearXNG started successfully and is ready for searches"), nil
}

func searxngConfigHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	config := map[string]interface{}{
		"default_settings":     GetSearXNGSettingsYAML(),
		"container_name":       SearXNGContainerName,
		"default_image":        SearXNGImage,
		"default_port":         "8080",
		"privacy_note":         "SearXNG is a privacy-respecting metasearch engine that queries multiple search engines without tracking",
		"engine_documentation": "https://docs.searxng.org/",
	}

	jsonResult, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("result serialization error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonResult)), nil
}
