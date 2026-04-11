package main

// DefaultSearXNGSettings returns the default SearXNG configuration
// optimized for privacy and local use
const DefaultSearXNGSettings = `
# SearXNG Settings for MCP Search Server
# Auto-generated default configuration

general:
  debug: false
  instance_name: "MCP Local Search"
  contact_url: false
  enable_stats: false

search:
  safe_search: 0
  autocomplete: "duckduckgo"
  autocomplete_min: 4
  default_lang: ""
  favicon_resolver: ""
  default_open_results: false
  advanced_search: true

outgoing:
  request_timeout: 10.0
  max_request_timeout: 15.0
  pool_connections: 100
  pool_maxsize: 100
  enable_http2: true
  retries: 1
  # Uncomment and configure proxy if needed:
  # proxies:
  #   all://:
  #     - http://proxy.example.com:8080
  source_ips:
    - 0.0.0.0

# Enabled engines - privacy respecting
engines:
  - name: duckduckgo
    engine: duckduckgo
    shortcut: ddg
    disabled: false
    timeout: 5.0

  - name: brave
    engine: brave
    shortcut: br
    disabled: false
    timeout: 5.0

  - name: bing
    engine: bing
    shortcut: bi
    disabled: false
    timeout: 5.0

  - name: wikipedia
    engine: wikipedia
    shortcut: wp
    disabled: false
    timeout: 5.0

  - name: google
    engine: google
    shortcut: go
    disabled: false
    timeout: 5.0

  # Image search engines
  - name: bing images
    engine: bing_images
    shortcut: bii
    disabled: false
    timeout: 5.0

  # News search engines
  - name: bing news
    engine: bing_news
    shortcut: bin
    disabled: false
    timeout: 5.0

  # IT/technical search
  - name: github
    engine: github
    shortcut: gh
    disabled: false
    timeout: 5.0

  - name: stackoverflow
    engine: stackoverflow
    shortcut: so
    disabled: false
    timeout: 5.0

  - name: arch wiki
    engine: archlinux
    shortcut: aw
    disabled: false
    timeout: 5.0

# UI preferences
ui:
  static_use_hash: true
  templates_path: ""
  default_theme: simple
  default_locale: ""
  query_in_title: false
  infinite_scroll: false
  center_alignment: false
  results_on_new_tab: false
  theme_args:
    simple_style: auto

# Result preferences
results:
  formats:
    - html

# Server settings (for internal use)
server:
  port: 8080
  bind_address: 127.0.0.1
  # Secret key is auto-generated at runtime, not hardcoded
  secret_key: ""
  base_url: false
  image_proxy: false
  http_protocol_version: "1.1"
  method: "POST"
  default_http_headers:
    X-Content-Type-Options: nosniff
    X-Download-Options: noopen
    X-Robots-Tag: noindex, nofollow
    Referrer-Policy: no-referrer

# Brand settings
brand:
  issue_url: https://github.com/searxng/searxng/issues
  docs_url: https://docs.searxng.org/
  public_instances: https://searx.space/
  wiki_url: https://github.com/searxng/searxng/wiki
`

// GetSearXNGSettingsYAML returns the default configuration
func GetSearXNGSettingsYAML() string {
	return DefaultSearXNGSettings
}
