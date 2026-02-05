# Developer Experience Guide

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Strategy**: Go Core + Python Bindings (Strategy C)
> **Purpose**: CLI tools, debugging, project templates, and IDE support for SDK users

## Overview

Developer Experience (DX) is critical for SDK adoption. This document covers the tools and features that make the crawler SDK easy to use, debug, and integrate.

---

## 1. CLI Tool

### 1.1 CLI Architecture

```
crawler
├── init          # Initialize new project
├── generate      # Generate code (spiders, pipelines)
├── run           # Run crawler
├── server        # Start gRPC server
├── job           # Manage crawl jobs
│   ├── start     # Start a job
│   ├── status    # Check job status
│   ├── stop      # Stop a job
│   └── list      # List all jobs
├── crawl         # Quick single-URL crawl
├── test          # Test spiders with mock data
├── benchmark     # Performance benchmarking
├── config        # Configuration management
└── version       # Show version info
```

### 1.2 CLI Implementation

```go
// cmd/crawler/main.go
package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/yourorg/crawler-sdk/cmd/crawler/commands"
)

var (
    Version   = "dev"
    BuildTime = "unknown"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "crawler",
        Short: "Web Crawler SDK CLI",
        Long:  `A powerful CLI tool for the Go-based web crawler SDK.`,
    }

    // Add subcommands
    rootCmd.AddCommand(
        commands.InitCmd(),
        commands.GenerateCmd(),
        commands.RunCmd(),
        commands.ServerCmd(),
        commands.JobCmd(),
        commands.CrawlCmd(),
        commands.TestCmd(),
        commands.BenchmarkCmd(),
        commands.ConfigCmd(),
        versionCmd(),
    )

    // Global flags
    rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func versionCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "version",
        Short: "Show version information",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("Crawler SDK v%s (built %s)\n", Version, BuildTime)
        },
    }
}
```

### 1.3 Init Command

```go
// cmd/crawler/commands/init.go
package commands

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
)

func InitCmd() *cobra.Command {
    var template string
    var language string

    cmd := &cobra.Command{
        Use:   "init [project-name]",
        Short: "Initialize a new crawler project",
        Long: `Initialize a new crawler project with the specified template.

Available templates:
  - basic       Basic single-spider project
  - ecommerce   E-commerce product scraping
  - news        News article extraction
  - api         API-based data collection

Examples:
  crawler init my-project
  crawler init my-project --template ecommerce
  crawler init my-project --language python`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            projectName := args[0]
            return initProject(projectName, template, language)
        },
    }

    cmd.Flags().StringVarP(&template, "template", "t", "basic", "project template")
    cmd.Flags().StringVarP(&language, "language", "l", "go", "primary language (go, python)")

    return cmd
}

func initProject(name, template, language string) error {
    // Create project directory
    if err := os.MkdirAll(name, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // Generate project structure based on template and language
    generator := &ProjectGenerator{
        Name:     name,
        Template: template,
        Language: language,
    }

    if err := generator.Generate(); err != nil {
        return fmt.Errorf("failed to generate project: %w", err)
    }

    fmt.Printf("✓ Created project '%s' with template '%s'\n", name, template)
    fmt.Printf("\nNext steps:\n")
    fmt.Printf("  cd %s\n", name)

    if language == "python" {
        fmt.Printf("  pip install -e .\n")
        fmt.Printf("  crawler run\n")
    } else {
        fmt.Printf("  go mod tidy\n")
        fmt.Printf("  crawler run\n")
    }

    return nil
}

type ProjectGenerator struct {
    Name     string
    Template string
    Language string
}

func (g *ProjectGenerator) Generate() error {
    // Create directory structure
    dirs := []string{
        "spiders",
        "pipelines",
        "items",
        "config",
        "output",
    }

    for _, dir := range dirs {
        path := filepath.Join(g.Name, dir)
        if err := os.MkdirAll(path, 0755); err != nil {
            return err
        }
    }

    // Generate files based on template
    return g.generateFiles()
}

func (g *ProjectGenerator) generateFiles() error {
    // Generate config file
    configContent := g.generateConfig()
    if err := os.WriteFile(filepath.Join(g.Name, "crawler.yaml"), []byte(configContent), 0644); err != nil {
        return err
    }

    // Generate main file based on language
    if g.Language == "python" {
        return g.generatePythonFiles()
    }
    return g.generateGoFiles()
}

func (g *ProjectGenerator) generateConfig() string {
    return `# Crawler Configuration
crawler:
  name: ` + g.Name + `
  max_concurrency: 10
  requests_per_second: 2.0
  respect_robots_txt: true
  user_agent: "CrawlerSDK/2.0 (+https://yourorg.com/crawler)"

http:
  timeout: 30s
  max_retries: 3

storage:
  type: json
  output_path: ./output

logging:
  level: info
  format: text
`
}

func (g *ProjectGenerator) generateGoFiles() error {
    mainGo := `package main

import (
    "context"
    "fmt"
    "log"

    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

func main() {
    c, err := crawler.New(crawler.WithConfigFile("crawler.yaml"))
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    c.OnHTML("a[href]", func(e *crawler.HTMLElement) {
        link := e.Attr("href")
        fmt.Printf("Found link: %s\n", link)
    })

    c.OnResponse(func(r *crawler.Response) {
        fmt.Printf("Crawled: %s [%d]\n", r.Request.URL, r.StatusCode)
    })

    if err := c.Visit("https://example.com"); err != nil {
        log.Fatal(err)
    }

    c.Wait()
}
`
    return os.WriteFile(filepath.Join(g.Name, "main.go"), []byte(mainGo), 0644)
}

func (g *ProjectGenerator) generatePythonFiles() error {
    mainPy := `"""` + g.Name + ` - Web Crawler Project"""

from crawler_sdk import CrawlerClient, CrawlOptions
from bs4 import BeautifulSoup


def main():
    with CrawlerClient() as client:
        result = client.crawl("https://example.com")

        if result.success:
            soup = BeautifulSoup(result.text, 'lxml')

            for link in soup.find_all('a', href=True):
                print(f"Found link: {link['href']}")
        else:
            print(f"Error: {result.error}")


if __name__ == "__main__":
    main()
`

    requirementsTxt := `crawler-sdk>=2.0.0
beautifulsoup4>=4.12.0
lxml>=5.0.0
`

    if err := os.WriteFile(filepath.Join(g.Name, "main.py"), []byte(mainPy), 0644); err != nil {
        return err
    }

    return os.WriteFile(filepath.Join(g.Name, "requirements.txt"), []byte(requirementsTxt), 0644)
}
```

### 1.4 Run Command

```go
// cmd/crawler/commands/run.go
package commands

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/spf13/cobra"
    "github.com/yourorg/crawler-sdk/internal/engine"
    "github.com/yourorg/crawler-sdk/pkg/config"
)

func RunCmd() *cobra.Command {
    var (
        configFile string
        seedURLs   []string
        maxPages   int
        debug      bool
    )

    cmd := &cobra.Command{
        Use:   "run [spider-name]",
        Short: "Run a crawler or spider",
        Long: `Run a crawler with the specified configuration.

Examples:
  crawler run                         # Run with default config
  crawler run --config custom.yaml    # Run with custom config
  crawler run --url https://example.com --debug`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Load configuration
            cfg, err := config.Load(configFile)
            if err != nil {
                return fmt.Errorf("failed to load config: %w", err)
            }

            // Override with command line args
            if len(seedURLs) > 0 {
                cfg.SeedURLs = seedURLs
            }
            if maxPages > 0 {
                cfg.Crawler.MaxPages = maxPages
            }
            if debug {
                cfg.Logging.Level = "debug"
            }

            // Create engine
            eng, err := engine.New(cfg)
            if err != nil {
                return fmt.Errorf("failed to create engine: %w", err)
            }
            defer eng.Close()

            // Handle signals
            ctx, cancel := context.WithCancel(context.Background())
            defer cancel()

            sigCh := make(chan os.Signal, 1)
            signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

            go func() {
                <-sigCh
                fmt.Println("\nShutting down gracefully...")
                cancel()
            }()

            // Run crawler
            fmt.Printf("Starting crawler with %d seed URLs...\n", len(cfg.SeedURLs))
            return eng.Run(ctx)
        },
    }

    cmd.Flags().StringVarP(&configFile, "config", "c", "crawler.yaml", "config file")
    cmd.Flags().StringSliceVarP(&seedURLs, "url", "u", nil, "seed URLs")
    cmd.Flags().IntVarP(&maxPages, "max-pages", "m", 0, "maximum pages to crawl")
    cmd.Flags().BoolVarP(&debug, "debug", "d", false, "enable debug mode")

    return cmd
}
```

### 1.5 Crawl Command (Quick Single URL)

```go
// cmd/crawler/commands/crawl.go
package commands

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/spf13/cobra"
    "github.com/yourorg/crawler-sdk/internal/engine"
)

func CrawlCmd() *cobra.Command {
    var (
        renderJS    bool
        timeout     time.Duration
        headers     []string
        proxy       string
        outputJSON  bool
        showHeaders bool
    )

    cmd := &cobra.Command{
        Use:   "crawl <url>",
        Short: "Quickly crawl a single URL",
        Long: `Fetch a single URL and display the results.

Examples:
  crawler crawl https://example.com
  crawler crawl https://spa-site.com --render-js
  crawler crawl https://api.example.com --json
  crawler crawl https://example.com -H "Authorization: Bearer token"`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            url := args[0]

            eng, err := engine.NewSimple()
            if err != nil {
                return err
            }
            defer eng.Close()

            ctx, cancel := context.WithTimeout(context.Background(), timeout)
            defer cancel()

            start := time.Now()
            result, err := eng.CrawlURL(ctx, url, &engine.CrawlOptions{
                RenderJS: renderJS,
                Headers:  parseHeaders(headers),
                Proxy:    proxy,
            })

            elapsed := time.Since(start)

            if err != nil {
                return fmt.Errorf("crawl failed: %w", err)
            }

            if outputJSON {
                return outputAsJSON(result, elapsed)
            }

            return outputAsText(result, elapsed, showHeaders)
        },
    }

    cmd.Flags().BoolVar(&renderJS, "render-js", false, "render JavaScript")
    cmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "request timeout")
    cmd.Flags().StringSliceVarP(&headers, "header", "H", nil, "custom headers")
    cmd.Flags().StringVar(&proxy, "proxy", "", "proxy URL")
    cmd.Flags().BoolVar(&outputJSON, "json", false, "output as JSON")
    cmd.Flags().BoolVar(&showHeaders, "show-headers", false, "show response headers")

    return cmd
}

func outputAsJSON(result *engine.CrawlResult, elapsed time.Duration) error {
    output := map[string]interface{}{
        "url":          result.URL,
        "status_code":  result.StatusCode,
        "content_type": result.ContentType,
        "content_size": len(result.Content),
        "fetch_time":   elapsed.String(),
        "headers":      result.Headers,
    }

    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil {
        return err
    }

    fmt.Println(string(data))
    return nil
}

func outputAsText(result *engine.CrawlResult, elapsed time.Duration, showHeaders bool) error {
    fmt.Printf("URL:          %s\n", result.URL)
    fmt.Printf("Status:       %d\n", result.StatusCode)
    fmt.Printf("Content-Type: %s\n", result.ContentType)
    fmt.Printf("Size:         %d bytes\n", len(result.Content))
    fmt.Printf("Fetch Time:   %s\n", elapsed)

    if showHeaders {
        fmt.Println("\nHeaders:")
        for key, values := range result.Headers {
            for _, value := range values {
                fmt.Printf("  %s: %s\n", key, value)
            }
        }
    }

    fmt.Println("\nContent Preview:")
    preview := string(result.Content)
    if len(preview) > 500 {
        preview = preview[:500] + "..."
    }
    fmt.Println(preview)

    return nil
}

func parseHeaders(headers []string) map[string]string {
    result := make(map[string]string)
    for _, h := range headers {
        // Parse "Key: Value" format
        for i, c := range h {
            if c == ':' {
                key := h[:i]
                value := h[i+1:]
                if len(value) > 0 && value[0] == ' ' {
                    value = value[1:]
                }
                result[key] = value
                break
            }
        }
    }
    return result
}
```

---

## 2. Project Templates

### 2.1 Basic Template (Go)

```
my-project/
├── main.go
├── crawler.yaml
├── spiders/
│   └── example_spider.go
├── items/
│   └── items.go
├── pipelines/
│   └── json_pipeline.go
├── go.mod
└── README.md
```

### 2.2 E-commerce Template (Python)

```
my-ecommerce-crawler/
├── main.py
├── crawler.yaml
├── spiders/
│   ├── __init__.py
│   └── product_spider.py
├── items/
│   ├── __init__.py
│   └── product.py
├── pipelines/
│   ├── __init__.py
│   ├── validation.py
│   └── database.py
├── tests/
│   ├── __init__.py
│   ├── test_spider.py
│   └── fixtures/
│       └── product_page.html
├── requirements.txt
├── setup.py
└── README.md
```

### 2.3 Template Files

```python
# templates/ecommerce/spiders/product_spider.py
"""Product spider for e-commerce sites."""

from crawler_sdk import CrawlerClient, CrawlOptions
from bs4 import BeautifulSoup
from items.product import Product
from pipelines.validation import validate_product


class ProductSpider:
    """Spider for extracting product data."""

    name = "product_spider"
    allowed_domains = ["example-shop.com"]
    start_urls = ["https://example-shop.com/products"]

    def __init__(self):
        self.client = CrawlerClient()
        self.options = CrawlOptions(
            render_js=True,  # Many e-commerce sites use JS
            timeout=30,
        )

    def crawl(self):
        """Main crawl method."""
        for url in self.start_urls:
            yield from self.parse_listing(url)

    def parse_listing(self, url: str):
        """Parse product listing page."""
        result = self.client.crawl(url, self.options)

        if not result.success:
            print(f"Failed to crawl {url}: {result.error}")
            return

        soup = BeautifulSoup(result.text, 'lxml')

        # Extract product links
        for link in soup.select('.product-card a'):
            product_url = link.get('href')
            if product_url:
                yield from self.parse_product(product_url)

        # Handle pagination
        next_page = soup.select_one('.pagination .next a')
        if next_page:
            yield from self.parse_listing(next_page['href'])

    def parse_product(self, url: str):
        """Parse individual product page."""
        result = self.client.crawl(url, self.options)

        if not result.success:
            print(f"Failed to crawl product {url}: {result.error}")
            return

        soup = BeautifulSoup(result.text, 'lxml')

        product = Product(
            url=url,
            name=self.extract_text(soup, '.product-title'),
            price=self.extract_price(soup, '.product-price'),
            description=self.extract_text(soup, '.product-description'),
            image_url=self.extract_attr(soup, '.product-image img', 'src'),
            sku=self.extract_text(soup, '.product-sku'),
            in_stock=self.check_stock(soup),
        )

        if validate_product(product):
            yield product

    def extract_text(self, soup, selector: str) -> str:
        """Extract text from selector."""
        elem = soup.select_one(selector)
        return elem.get_text(strip=True) if elem else ""

    def extract_attr(self, soup, selector: str, attr: str) -> str:
        """Extract attribute from selector."""
        elem = soup.select_one(selector)
        return elem.get(attr, "") if elem else ""

    def extract_price(self, soup, selector: str) -> float:
        """Extract and parse price."""
        text = self.extract_text(soup, selector)
        import re
        match = re.search(r'[\d,.]+', text.replace(',', ''))
        return float(match.group()) if match else 0.0

    def check_stock(self, soup) -> bool:
        """Check if product is in stock."""
        stock_elem = soup.select_one('.stock-status')
        if stock_elem:
            return 'in stock' in stock_elem.get_text().lower()
        return True  # Default to in stock

    def close(self):
        """Clean up resources."""
        self.client.close()
```

---

## 3. Debug Mode

### 3.1 Debug Features

```go
// pkg/debug/debug.go
package debug

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

type DebugMode struct {
    enabled      bool
    requestLog   []*RequestLog
    maxLogSize   int
    breakpoints  map[string]BreakpointFunc
}

type RequestLog struct {
    Timestamp   time.Time              `json:"timestamp"`
    URL         string                 `json:"url"`
    Method      string                 `json:"method"`
    StatusCode  int                    `json:"status_code"`
    Duration    time.Duration          `json:"duration"`
    RequestSize int                    `json:"request_size"`
    ResponseSize int                   `json:"response_size"`
    Error       string                 `json:"error,omitempty"`
    Headers     map[string]string      `json:"headers"`
    Metadata    map[string]interface{} `json:"metadata"`
}

type BreakpointFunc func(req *crawler.Request, resp *crawler.Response) bool

func NewDebugMode() *DebugMode {
    return &DebugMode{
        enabled:     true,
        requestLog:  make([]*RequestLog, 0),
        maxLogSize:  1000,
        breakpoints: make(map[string]BreakpointFunc),
    }
}

// LogRequest logs a request/response pair
func (d *DebugMode) LogRequest(req *crawler.Request, resp *crawler.Response, duration time.Duration, err error) {
    if !d.enabled {
        return
    }

    log := &RequestLog{
        Timestamp:    time.Now(),
        URL:          req.URL,
        Method:       req.Method,
        Duration:     duration,
        RequestSize:  len(req.Body),
        Metadata:     req.Metadata,
    }

    if resp != nil {
        log.StatusCode = resp.StatusCode
        log.ResponseSize = len(resp.Body)
        log.Headers = flattenHeaders(resp.Headers)
    }

    if err != nil {
        log.Error = err.Error()
    }

    d.requestLog = append(d.requestLog, log)

    // Trim log if too large
    if len(d.requestLog) > d.maxLogSize {
        d.requestLog = d.requestLog[len(d.requestLog)-d.maxLogSize:]
    }
}

// AddBreakpoint adds a conditional breakpoint
func (d *DebugMode) AddBreakpoint(name string, fn BreakpointFunc) {
    d.breakpoints[name] = fn
}

// CheckBreakpoints checks if any breakpoint is triggered
func (d *DebugMode) CheckBreakpoints(req *crawler.Request, resp *crawler.Response) []string {
    triggered := make([]string, 0)

    for name, fn := range d.breakpoints {
        if fn(req, resp) {
            triggered = append(triggered, name)
        }
    }

    return triggered
}

// GetLogs returns the request log
func (d *DebugMode) GetLogs() []*RequestLog {
    return d.requestLog
}

// GetStats returns debug statistics
func (d *DebugMode) GetStats() map[string]interface{} {
    var totalDuration time.Duration
    var totalSuccess, totalError int
    statusCodes := make(map[int]int)

    for _, log := range d.requestLog {
        totalDuration += log.Duration
        if log.Error != "" {
            totalError++
        } else {
            totalSuccess++
        }
        statusCodes[log.StatusCode]++
    }

    avgDuration := time.Duration(0)
    if len(d.requestLog) > 0 {
        avgDuration = totalDuration / time.Duration(len(d.requestLog))
    }

    return map[string]interface{}{
        "total_requests":    len(d.requestLog),
        "successful":        totalSuccess,
        "failed":            totalError,
        "avg_duration_ms":   avgDuration.Milliseconds(),
        "status_codes":      statusCodes,
    }
}

// ServeDebugUI starts a debug web UI
func (d *DebugMode) ServeDebugUI(port int) error {
    mux := http.NewServeMux()

    mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(d.GetLogs())
    })

    mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(d.GetStats())
    })

    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(debugUIHTML))
    })

    fmt.Printf("Debug UI available at http://localhost:%d\n", port)
    return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

var debugUIHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Crawler Debug UI</title>
    <style>
        body { font-family: -apple-system, sans-serif; margin: 20px; }
        .stats { display: flex; gap: 20px; margin-bottom: 20px; }
        .stat-card { padding: 15px; background: #f5f5f5; border-radius: 8px; }
        .log-table { width: 100%; border-collapse: collapse; }
        .log-table th, .log-table td { padding: 8px; border: 1px solid #ddd; text-align: left; }
        .log-table tr:nth-child(even) { background: #f9f9f9; }
        .status-2xx { color: green; }
        .status-4xx { color: orange; }
        .status-5xx { color: red; }
    </style>
</head>
<body>
    <h1>Crawler Debug UI</h1>
    <div class="stats" id="stats"></div>
    <h2>Request Log</h2>
    <table class="log-table" id="logs">
        <thead>
            <tr>
                <th>Time</th>
                <th>URL</th>
                <th>Status</th>
                <th>Duration</th>
                <th>Size</th>
                <th>Error</th>
            </tr>
        </thead>
        <tbody></tbody>
    </table>
    <script>
        async function refresh() {
            const [stats, logs] = await Promise.all([
                fetch('/api/stats').then(r => r.json()),
                fetch('/api/logs').then(r => r.json())
            ]);

            document.getElementById('stats').innerHTML =
                '<div class="stat-card">Total: ' + stats.total_requests + '</div>' +
                '<div class="stat-card">Success: ' + stats.successful + '</div>' +
                '<div class="stat-card">Failed: ' + stats.failed + '</div>' +
                '<div class="stat-card">Avg: ' + stats.avg_duration_ms + 'ms</div>';

            const tbody = document.querySelector('#logs tbody');
            tbody.innerHTML = logs.slice(-100).reverse().map(log =>
                '<tr>' +
                '<td>' + new Date(log.timestamp).toLocaleTimeString() + '</td>' +
                '<td>' + log.url.substring(0, 60) + '</td>' +
                '<td class="status-' + Math.floor(log.status_code/100) + 'xx">' + log.status_code + '</td>' +
                '<td>' + (log.duration/1000000).toFixed(0) + 'ms</td>' +
                '<td>' + log.response_size + '</td>' +
                '<td>' + (log.error || '') + '</td>' +
                '</tr>'
            ).join('');
        }

        refresh();
        setInterval(refresh, 2000);
    </script>
</body>
</html>`

func flattenHeaders(headers http.Header) map[string]string {
    result := make(map[string]string)
    for key, values := range headers {
        if len(values) > 0 {
            result[key] = values[0]
        }
    }
    return result
}
```

### 3.2 Python Debug Tools

```python
# bindings/python/crawler_sdk/debug.py
"""Debug utilities for the crawler SDK."""

from __future__ import annotations

import json
import logging
from datetime import datetime
from typing import List, Optional, Dict, Any
from dataclasses import dataclass, field, asdict
from contextlib import contextmanager

from .client import CrawlResult


@dataclass
class DebugLog:
    """Debug log entry."""
    timestamp: datetime
    url: str
    status_code: int
    duration_ms: float
    response_size: int
    error: Optional[str] = None
    metadata: Dict[str, Any] = field(default_factory=dict)


class DebugSession:
    """Debug session for tracking crawl operations."""

    def __init__(self, name: str = "default"):
        self.name = name
        self.logs: List[DebugLog] = []
        self.start_time = datetime.now()
        self._logger = logging.getLogger(f"crawler.debug.{name}")

    def log(self, result: CrawlResult, duration_ms: float):
        """Log a crawl result."""
        entry = DebugLog(
            timestamp=datetime.now(),
            url=result.url,
            status_code=result.status_code,
            duration_ms=duration_ms,
            response_size=len(result.content),
            error=result.error,
            metadata=result.metadata,
        )
        self.logs.append(entry)
        self._logger.debug(f"Crawled {result.url} [{result.status_code}] in {duration_ms:.0f}ms")

    def stats(self) -> Dict[str, Any]:
        """Get session statistics."""
        if not self.logs:
            return {"total": 0}

        total = len(self.logs)
        success = sum(1 for log in self.logs if log.error is None and 200 <= log.status_code < 400)
        failed = total - success
        avg_duration = sum(log.duration_ms for log in self.logs) / total
        total_bytes = sum(log.response_size for log in self.logs)

        status_codes = {}
        for log in self.logs:
            status_codes[log.status_code] = status_codes.get(log.status_code, 0) + 1

        return {
            "name": self.name,
            "total": total,
            "success": success,
            "failed": failed,
            "success_rate": f"{success/total*100:.1f}%",
            "avg_duration_ms": round(avg_duration, 2),
            "total_bytes": total_bytes,
            "status_codes": status_codes,
            "duration": str(datetime.now() - self.start_time),
        }

    def export(self, path: str, format: str = "json"):
        """Export debug logs to file."""
        if format == "json":
            data = [asdict(log) for log in self.logs]
            for item in data:
                item['timestamp'] = item['timestamp'].isoformat()
            with open(path, 'w') as f:
                json.dump(data, f, indent=2)
        elif format == "csv":
            import csv
            with open(path, 'w', newline='') as f:
                writer = csv.writer(f)
                writer.writerow(['timestamp', 'url', 'status_code', 'duration_ms', 'response_size', 'error'])
                for log in self.logs:
                    writer.writerow([
                        log.timestamp.isoformat(),
                        log.url,
                        log.status_code,
                        log.duration_ms,
                        log.response_size,
                        log.error or '',
                    ])

    def print_summary(self):
        """Print session summary."""
        stats = self.stats()
        print(f"\n{'='*50}")
        print(f"Debug Session: {stats['name']}")
        print(f"{'='*50}")
        print(f"Total requests:  {stats['total']}")
        print(f"Successful:      {stats['success']} ({stats['success_rate']})")
        print(f"Failed:          {stats['failed']}")
        print(f"Avg duration:    {stats['avg_duration_ms']}ms")
        print(f"Total bytes:     {stats['total_bytes']:,}")
        print(f"Duration:        {stats['duration']}")
        print(f"\nStatus codes:")
        for code, count in sorted(stats['status_codes'].items()):
            print(f"  {code}: {count}")
        print(f"{'='*50}\n")


@contextmanager
def debug_session(name: str = "default"):
    """Context manager for debug sessions.

    Example:
        >>> with debug_session("my-crawl") as session:
        ...     # Your crawling code here
        ...     session.log(result, duration_ms)
        >>> session.print_summary()
    """
    session = DebugSession(name)
    try:
        yield session
    finally:
        session.print_summary()


class DebugClient:
    """Wrapper client with debug capabilities."""

    def __init__(self, client, session: Optional[DebugSession] = None):
        self.client = client
        self.session = session or DebugSession()

    def crawl(self, url: str, **kwargs):
        """Crawl with timing and logging."""
        import time
        start = time.time()
        result = self.client.crawl(url, **kwargs)
        duration_ms = (time.time() - start) * 1000
        self.session.log(result, duration_ms)
        return result

    def crawl_batch(self, urls, **kwargs):
        """Crawl batch with timing and logging."""
        import time
        for result in self.client.crawl_batch(urls, **kwargs):
            # Note: Individual timing not available for batch
            self.session.log(result, 0)
            yield result
```

---

## 4. IDE Support

### 4.1 VS Code Extension Configuration

```json
// .vscode/settings.json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "go.testFlags": ["-v"],
    "go.coverOnSave": true,

    "python.linting.enabled": true,
    "python.linting.mypyEnabled": true,
    "python.formatting.provider": "black",

    "[go]": {
        "editor.formatOnSave": true,
        "editor.codeActionsOnSave": {
            "source.organizeImports": true
        }
    },

    "[python]": {
        "editor.formatOnSave": true
    },

    "files.associations": {
        "crawler.yaml": "yaml",
        "*.proto": "proto3"
    }
}
```

### 4.2 Code Snippets

```json
// .vscode/crawler.code-snippets
{
    "Go Spider": {
        "prefix": "spider",
        "body": [
            "package spiders",
            "",
            "import (",
            "\t\"github.com/yourorg/crawler-sdk/pkg/crawler\"",
            ")",
            "",
            "type ${1:Name}Spider struct {",
            "\tcrawler.BaseSpider",
            "}",
            "",
            "func New${1:Name}Spider() *${1:Name}Spider {",
            "\treturn &${1:Name}Spider{}",
            "}",
            "",
            "func (s *${1:Name}Spider) Parse(resp *crawler.Response) error {",
            "\t$0",
            "\treturn nil",
            "}"
        ],
        "description": "Create a new Go spider"
    },

    "Python Spider": {
        "prefix": "pyspider",
        "body": [
            "from crawler_sdk import CrawlerClient, CrawlOptions",
            "from bs4 import BeautifulSoup",
            "",
            "",
            "class ${1:Name}Spider:",
            "    \"\"\"Spider for ${2:description}.\"\"\"",
            "",
            "    name = \"${3:spider_name}\"",
            "    start_urls = [\"${4:https://example.com}\"]",
            "",
            "    def __init__(self):",
            "        self.client = CrawlerClient()",
            "",
            "    def crawl(self):",
            "        for url in self.start_urls:",
            "            result = self.client.crawl(url)",
            "            yield from self.parse(result)",
            "",
            "    def parse(self, result):",
            "        soup = BeautifulSoup(result.text, 'lxml')",
            "        $0",
            "",
            "    def close(self):",
            "        self.client.close()"
        ],
        "description": "Create a new Python spider"
    }
}
```

---

## 5. Testing Support

### 5.1 Test Command

```go
// cmd/crawler/commands/test.go
package commands

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

func TestCmd() *cobra.Command {
    var (
        mockServer bool
        coverage   bool
        verbose    bool
    )

    cmd := &cobra.Command{
        Use:   "test [spider-name]",
        Short: "Test spiders with mock data",
        Long: `Run tests for spiders using mock HTTP responses.

Examples:
  crawler test                    # Run all tests
  crawler test product_spider     # Test specific spider
  crawler test --mock-server      # Start mock server for manual testing
  crawler test --coverage         # Run with coverage`,
        RunE: func(cmd *cobra.Command, args []string) error {
            if mockServer {
                return startMockServer()
            }

            return runTests(args, coverage, verbose)
        },
    }

    cmd.Flags().BoolVar(&mockServer, "mock-server", false, "start mock server")
    cmd.Flags().BoolVar(&coverage, "coverage", false, "run with coverage")
    cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

    return cmd
}

func runTests(spiders []string, coverage, verbose bool) error {
    args := []string{"test"}

    if verbose {
        args = append(args, "-v")
    }
    if coverage {
        args = append(args, "-coverprofile=coverage.out")
    }

    args = append(args, "./...")

    // Run go test
    cmd := exec.Command("go", args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func startMockServer() error {
    fmt.Println("Starting mock server on http://localhost:8080")
    fmt.Println("Press Ctrl+C to stop")

    // Start mock HTTP server
    mux := http.NewServeMux()

    // Add mock routes
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte("<html><body><h1>Mock Server</h1></body></html>"))
    })

    mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(mockProductsHTML))
    })

    return http.ListenAndServe(":8080", mux)
}

var mockProductsHTML = `<!DOCTYPE html>
<html>
<body>
    <div class="product-card">
        <a href="/product/1">
            <h2 class="product-title">Test Product 1</h2>
            <span class="product-price">$99.99</span>
        </a>
    </div>
    <div class="product-card">
        <a href="/product/2">
            <h2 class="product-title">Test Product 2</h2>
            <span class="product-price">$149.99</span>
        </a>
    </div>
</body>
</html>`
```

---

## 6. Documentation Generation

### 6.1 Auto-Generate API Docs

```go
// scripts/gendocs.go
package main

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "os"
    "path/filepath"
    "strings"
)

func main() {
    // Generate Go API docs
    generateGoDocs("pkg/crawler", "docs/api/go")

    // Generate Python API docs
    generatePythonDocs("bindings/python/crawler_sdk", "docs/api/python")
}

func generateGoDocs(srcDir, outDir string) {
    fset := token.NewFileSet()

    packages, err := parser.ParseDir(fset, srcDir, nil, parser.ParseComments)
    if err != nil {
        fmt.Printf("Error parsing: %v\n", err)
        return
    }

    for name, pkg := range packages {
        output := fmt.Sprintf("# Package %s\n\n", name)

        for _, file := range pkg.Files {
            for _, decl := range file.Decls {
                switch d := decl.(type) {
                case *ast.GenDecl:
                    for _, spec := range d.Specs {
                        if ts, ok := spec.(*ast.TypeSpec); ok {
                            output += formatTypeDoc(ts, d.Doc)
                        }
                    }
                case *ast.FuncDecl:
                    output += formatFuncDoc(d)
                }
            }
        }

        outPath := filepath.Join(outDir, name+".md")
        os.MkdirAll(outDir, 0755)
        os.WriteFile(outPath, []byte(output), 0644)
    }
}

func formatTypeDoc(ts *ast.TypeSpec, doc *ast.CommentGroup) string {
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("## %s\n\n", ts.Name.Name))

    if doc != nil {
        sb.WriteString(doc.Text())
        sb.WriteString("\n\n")
    }

    return sb.String()
}

func formatFuncDoc(fd *ast.FuncDecl) string {
    var sb strings.Builder

    if fd.Recv != nil {
        sb.WriteString(fmt.Sprintf("### %s.%s\n\n", getReceiverType(fd.Recv), fd.Name.Name))
    } else {
        sb.WriteString(fmt.Sprintf("### %s\n\n", fd.Name.Name))
    }

    if fd.Doc != nil {
        sb.WriteString(fd.Doc.Text())
        sb.WriteString("\n\n")
    }

    return sb.String()
}
```

---

## References

- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [Viper Configuration](https://github.com/spf13/viper)
- [VS Code Go Extension](https://marketplace.visualstudio.com/items?itemName=golang.go)
- [VS Code Python Extension](https://marketplace.visualstudio.com/items?itemName=ms-python.python)

---

*Great developer experience is the difference between adoption and abandonment.*
