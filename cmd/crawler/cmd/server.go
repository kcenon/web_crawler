package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/kcenon/web_crawler/pkg/crawler"
	"github.com/kcenon/web_crawler/pkg/server"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the gRPC server",
	Long:  "Start the Web Crawler gRPC server for remote client connections.",
	RunE:  runServer,
}

var (
	serverHost string
	serverPort int
)

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "server listen host")
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 50051, "server listen port")
}

func runServer(_ *cobra.Command, _ []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	svc := crawler.NewService()

	srv := server.New(svc, server.Config{
		Port: serverPort,
	}, server.WithLogger(slog.Default()))

	fmt.Fprintf(os.Stderr, "Starting gRPC server on %s:%d\n", serverHost, serverPort)

	if err := srv.Start(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
