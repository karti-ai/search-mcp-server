# search-mcp

MCP server for web and code search using **SearXNG** (privacy-respecting metasearch) and **Grep.app** (GitHub code search).

**Version 2.0.0** - Now with auto-managed local SearXNG!

> 🎯 **Tavily Alternative**: Free, local, privacy-respecting search - no API keys, no external dependencies, no cost!

## What's New in v2.0.0

🎉 **Fully Self-Contained Search** - No external SearXNG instance needed!

- ✅ **Auto-starts SearXNG** Docker container on first search
- ✅ **Keeps SearXNG running** for fast subsequent searches
- ✅ **Local-only** - all searches stay on your machine
- ✅ **Privacy-first** - queries multiple engines without tracking
- ✅ **Zero configuration** - works out of the box

## Features

### Web Search (via SearXNG)
- `searxng_search` - General web search with multiple categories
- `searxng_image_search` - Image search
- `searxng_news_search` - News search
- `searxng_engines_info` - List available search engines

### Code Search (via Grep.app)
- `grep_search` - Search code across public GitHub repos
- `github_file` - Fetch single file from GitHub
- `github_batch_files` - Fetch multiple files in parallel

### Management Tools
- `search_status` - Check status of all search services
- `searxng_start` - Manually start SearXNG container
- `searxng_stop` - Stop SearXNG to free resources
- `searxng_config` - View default SearXNG configuration

## vs Tavily (and other paid search APIs)

| Feature | Tavily | search-mcp v2 |
|---------|--------|-----------------|
| **Cost** | Paid API ($$/month) | **Free forever** |
| **Privacy** | Sends queries to external service | **100% local** - searches stay on your machine |
| **API Keys** | Required | **None needed** |
| **Rate Limits** | Yes | **Unlimited** |
| **Offline** | No | **Works offline** (with cached results) |
| **Setup** | Sign up, get API key | **Install Docker, run binary** |

**Bottom line**: If you want privacy, zero cost, and zero configuration - use search-mcp v2.

## Quick Start

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) must be installed

### Download & Run

```bash
# Clone and build from source
git clone https://github.com/yourusername/search-mcp
cd search-mcp
make build

# Or download release (coming soon)
# curl -L https://github.com/yourusername/search-mcp/releases/download/v2.0.0/search-mcp-$(uname -s)-$(uname -m) -o search-mcp
# chmod +x search-mcp

# Run with stdio (for MCP clients like Claude Desktop, Cursor)
./go_mcp_server_searxng -t stdio
```

### MCP Client Configuration

**Claude Desktop (`claude_desktop_config.json`):**
```json
{
  "mcpServers": {
    "search": {
      "command": "/path/to/search-mcp",
      "args": ["-t", "stdio"]
    }
  }
}
```

**Cursor (Settings > MCP):**
```json
{
  "mcpServers": {
    "search": {
      "command": "/path/to/search-mcp",
      "args": ["-t", "stdio"]
    }
  }
}
```

## How It Works

```
User asks for search
        ↓
MCP Server checks if SearXNG running
        ↓
    ┌─── No ───┐
    ↓          │
Auto-start    │
Docker container
    ↓          │
Wait for ready│
    ↓          │
    └──── Yes ─┘
        ↓
  Execute search
        ↓
  Return results
```

**First search:** ~10-15 seconds (includes container startup)

**Subsequent searches:** ~1-2 seconds (instant response)

## Command-Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-t` | `stdio` | Transport type: `stdio` or `sse` |
| `-h` | `0.0.0.0` | Host for SSE server |
| `-p` | `8892` | Port for SSE server |
| `-searxng` | `http://127.0.0.1:8080` | SearXNG URL (for external instance) |
| `-searxng-autostart` | `true` | Auto-start SearXNG container |
| `-searxng-image` | `searxng/searxng:latest` | Docker image to use |
| `-searxng-port` | `8080` | Port for SearXNG container |
| `-searxng-stop-on-exit` | `false` | Stop container when MCP server exits |
| `-searxng-timeout` | `60` | Seconds to wait for SearXNG ready |

### Examples

```bash
# Default: auto-start SearXNG, keep running after exit
./search-mcp -t stdio

# Stop SearXNG when MCP exits (saves RAM)
./search-mcp -t stdio -searxng-stop-on-exit

# Use external SearXNG instance (skip auto-start)
./search-mcp -t stdio -searxng https://searx.example.com

# SSE mode with custom port
./search-mcp -t sse -p 3000
```

## Resource Usage

| Component | Memory | CPU | Notes |
|-----------|--------|-----|-------|
| MCP Server | ~10MB | Negligible | Very lightweight |
| SearXNG | ~200-300MB | Low idle | Containerized metasearch |
| **Total** | **~250MB** | **Bursty** | Reasonable for modern machines |

## Privacy & Security

- ✅ **All searches local** - Nothing leaves your machine except to search engines
- ✅ **No tracking** - SearXNG aggregates results without personalization
- ✅ **No API keys needed** - Uses public search engines
- ✅ **Docker isolation** - SearXNG runs in container with no host access

Default engines: DuckDuckGo, Brave, Bing, Wikipedia, GitHub, StackOverflow

## Troubleshooting

### "Docker is not available"
Install Docker: https://docs.docker.com/get-docker/

### "Port 8080 is already in use"
```bash
# Use a different port
./search-mcp -t stdio -searxng-port 8081
```

### SearXNG won't start
```bash
# Check Docker is running
docker ps

# Check logs
docker logs searxng-mcp-local

# Reset: remove container
docker rm -f searxng-mcp-local
```

### Search is slow
First search is slow (container startup). If consistently slow:
- Check internet connection
- Try different engines in search queries
- Check `search_status` tool output

## Building from Source

```bash
# Clone repo
git clone https://github.com/yourusername/search-mcp
cd search-mcp

# Build
go build -o go_mcp_server_searxng .

# Or cross-compile
make build-all
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    MCP Client                           │
│              (Claude Desktop, Cursor, etc.)            │
└─────────────────────────────────────────────────────────┘
                          │
                          │ stdio/SSE
                          ▼
┌─────────────────────────────────────────────────────────┐
│                 search-mcp (Go)                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Auto-start logic                                │  │
│  │  • Check if SearXNG running                      │  │
│  │  • Start Docker container if needed              │  │
│  │  • Wait for health check                        │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │
│  │   Search    │  │   Image   │  │    News     │      │
│  │   Tools     │  │   Search  │  │   Search    │      │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘      │
│         └─────────────────┼─────────────────┘           │
│                           │                           │
│  ┌─────────────┐         │         ┌─────────────┐     │
│  │   Grep.app  │◄────────┴────────►│  SearXNG    │     │
│  │  (GitHub)   │                   │  (Docker)   │     │
│  └─────────────┘                   └─────────────┘     │
└─────────────────────────────────────────────────────────┘
```

## License

MIT
