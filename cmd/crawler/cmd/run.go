package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kcenon/web_crawler/pkg/crawler"
)

var runCmd = &cobra.Command{
	Use:   "run [config-file]",
	Short: "Execute a crawl job from a YAML configuration file",
	Long:  "Run a crawl job defined in a YAML configuration file. Uses crawler.yaml by default.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRun,
}

var (
	runConcurrent int
	runMaxDepth   int
	runOutput     string
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().IntVarP(&runConcurrent, "concurrent", "c", 0, "number of concurrent workers")
	runCmd.Flags().IntVar(&runMaxDepth, "max-depth", 0, "maximum crawl depth (overrides config)")
	runCmd.Flags().StringVarP(&runOutput, "output", "o", "", "output file (default: stdout)")
}

type runConfig struct {
	URLs      []string          `mapstructure:"urls"`
	MaxDepth  int               `mapstructure:"max_depth"`
	MaxPages  int               `mapstructure:"max_pages"`
	Workers   int               `mapstructure:"workers"`
	UserAgent string            `mapstructure:"user_agent"`
	Headers   map[string]string `mapstructure:"headers"`
	Timeout   time.Duration     `mapstructure:"timeout"`
}

type runResult struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	BodyLength int    `json:"body_length"`
	Error      string `json:"error,omitempty"`
}

func runRun(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		viper.SetConfigFile(args[0])
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("read config %s: %w", args[0], err)
		}
	}

	var rc runConfig
	if err := viper.Unmarshal(&rc); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	if len(rc.URLs) == 0 {
		return fmt.Errorf("no URLs specified in configuration")
	}

	// CLI flags override config values.
	if runMaxDepth > 0 {
		rc.MaxDepth = runMaxDepth
	}
	if runConcurrent > 0 {
		rc.Workers = runConcurrent
	}

	if rc.MaxDepth == 0 {
		rc.MaxDepth = 3
	}
	if rc.Workers == 0 {
		rc.Workers = 10
	}
	if rc.UserAgent == "" {
		rc.UserAgent = "web_crawler/0.1"
	}
	if rc.Timeout == 0 {
		rc.Timeout = 5 * time.Minute
	}

	cfg := crawler.Config{
		MaxDepth:    rc.MaxDepth,
		MaxPages:    rc.MaxPages,
		UserAgent:   rc.UserAgent,
		WorkerCount: rc.Workers,
	}

	c := crawler.NewEngine(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), rc.Timeout)
	defer cancel()

	var results []runResult

	c.OnResponse(func(resp *crawler.CrawlResponse) {
		r := runResult{
			URL:        resp.Request.URL,
			StatusCode: resp.StatusCode,
			BodyLength: len(resp.Body),
		}
		results = append(results, r)
	})

	c.OnError(func(req *crawler.CrawlRequest, err error) {
		r := runResult{
			URL:   req.URL,
			Error: err.Error(),
		}
		results = append(results, r)
	})

	opts := make([]crawler.RequestOption, 0)
	if len(rc.Headers) > 0 {
		opts = append(opts, crawler.WithHeaders(rc.Headers))
	}

	if err := c.AddURLs(rc.URLs, opts...); err != nil {
		return fmt.Errorf("add URLs: %w", err)
	}

	slog.Info("starting crawl", "urls", len(rc.URLs), "max_depth", rc.MaxDepth, "workers", rc.Workers)

	if err := c.Start(ctx); err != nil {
		slog.Debug("crawler start", "error", err)
	}

	if err := c.Wait(); err != nil {
		slog.Debug("crawler wait", "error", err)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	if runOutput != "" {
		return os.WriteFile(runOutput, data, 0o644) //nolint:gosec // CLI output file
	}

	fmt.Println(string(data))
	return nil
}
