package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new crawler project",
	Long:  "Generate a project scaffold with configuration file, example spider, and directory structure.",
	Args:  cobra.ExactArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

const defaultConfigYAML = `# Web Crawler SDK configuration
urls:
  - https://example.com

max_depth: 3
max_pages: 100
workers: 10
user_agent: "web_crawler/0.1"
timeout: 5m

headers:
  Accept: "text/html,application/xhtml+xml"
`

func runInit(_ *cobra.Command, args []string) error {
	projectName := args[0]
	projectDir := filepath.Clean(projectName)

	dirs := []string{
		projectDir,
		filepath.Join(projectDir, "output"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	configPath := filepath.Join(projectDir, "crawler.yaml")
	if err := os.WriteFile(configPath, []byte(defaultConfigYAML), 0o644); err != nil { //nolint:gosec // scaffold file
		return fmt.Errorf("write config: %w", err)
	}

	gitignorePath := filepath.Join(projectDir, ".gitignore")
	gitignoreContent := "output/\n*.log\n"
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644); err != nil { //nolint:gosec // scaffold file
		return fmt.Errorf("write .gitignore: %w", err)
	}

	fmt.Printf("Project %q initialized successfully.\n", projectName)
	fmt.Printf("  %s\n", configPath)
	fmt.Printf("  %s\n", gitignorePath)
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", projectDir)
	fmt.Println("  crawler run crawler.yaml")

	return nil
}
