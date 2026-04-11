package main

import (
	"testing"
)

func TestToolNames(t *testing.T) {
	tools := getAllTools()
	
	expectedTools := []string{
		"searxng_search",
		"searxng_image_search", 
		"searxng_news_search",
		"grep_search",
		"github_file",
		"github_batch_files",
		"search_status",
		"searxng_start",
		"searxng_stop",
		"searxng_config",
		"searxng_engines_info",
	}
	
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}
}

func TestSearXNGConfig(t *testing.T) {
	config := getDefaultSearXNGConfig()
	if config == "" {
		t.Error("Default SearXNG config should not be empty")
	}
}
