package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kcenon/web_crawler/pkg/crawler"
)

var crawlCmd = &cobra.Command{
	Use:   "crawl <url>",
	Short: "Crawl a single URL and output the result",
	Long:  "Fetch a URL and output its content and extracted data as JSON.",
	Args:  cobra.ExactArgs(1),
	RunE:  runCrawl,
}

var (
	crawlHeaders   []string
	crawlTimeout   time.Duration
	crawlUserAgent string
	crawlOutput    string
	crawlExtract   []string
	crawlMaxDepth  int
)

func init() {
	rootCmd.AddCommand(crawlCmd)

	crawlCmd.Flags().StringSliceVarP(&crawlHeaders, "headers", "H", nil, "HTTP headers (key:value)")
	crawlCmd.Flags().DurationVarP(&crawlTimeout, "timeout", "t", 30*time.Second, "request timeout")
	crawlCmd.Flags().StringVar(&crawlUserAgent, "user-agent", "web_crawler/0.1", "User-Agent header")
	crawlCmd.Flags().StringVarP(&crawlOutput, "output", "o", "", "output file (default: stdout)")
	crawlCmd.Flags().StringSliceVarP(&crawlExtract, "extract", "e", nil, "CSS selectors to extract (name=selector)")
	crawlCmd.Flags().IntVar(&crawlMaxDepth, "max-depth", 1, "maximum crawl depth")
}

type crawlResult struct {
	URL         string            `json:"url"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type,omitempty"`
	BodyLength  int               `json:"body_length"`
	Headers     map[string]string `json:"headers,omitempty"`
	Duration    string            `json:"duration"`
	Error       string            `json:"error,omitempty"`
}

func runCrawl(_ *cobra.Command, args []string) error {
	targetURL := args[0]

	headers := parseHeaders(crawlHeaders)

	cfg := crawler.Config{
		MaxDepth:  crawlMaxDepth,
		MaxPages:  1,
		UserAgent: crawlUserAgent,
	}

	c := crawler.NewEngine(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), crawlTimeout)
	defer cancel()

	var result crawlResult
	result.URL = targetURL
	start := time.Now()

	c.OnResponse(func(resp *crawler.CrawlResponse) {
		result.StatusCode = resp.StatusCode
		result.ContentType = resp.ContentType
		result.BodyLength = len(resp.Body)

		respHeaders := make(map[string]string)
		for k, v := range resp.Headers {
			if len(v) > 0 {
				respHeaders[k] = v[0]
			}
		}
		result.Headers = respHeaders
	})

	c.OnError(func(_ *crawler.CrawlRequest, err error) {
		result.Error = err.Error()
	})

	opts := make([]crawler.RequestOption, 0)
	if len(headers) > 0 {
		opts = append(opts, crawler.WithHeaders(headers))
	}

	if err := c.AddURL(targetURL, opts...); err != nil {
		return fmt.Errorf("add URL: %w", err)
	}

	if err := c.Start(ctx); err != nil {
		slog.Debug("crawler start", "error", err)
	}

	if err := c.Wait(); err != nil {
		slog.Debug("crawler wait", "error", err)
	}

	result.Duration = time.Since(start).Round(time.Millisecond).String()

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	if crawlOutput != "" {
		return os.WriteFile(crawlOutput, data, 0o644) //nolint:gosec // CLI output file
	}

	fmt.Println(string(data))
	return nil
}

func parseHeaders(headers []string) map[string]string {
	m := make(map[string]string, len(headers))
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m
}
