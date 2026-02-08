package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cadre-oss/cadre/internal/config"
	"github.com/cadre-oss/cadre/internal/event"
	"github.com/cadre-oss/cadre/internal/server"
	"github.com/cadre-oss/cadre/internal/state"
	"github.com/cadre-oss/cadre/internal/telemetry"
	"github.com/spf13/cobra"
)

var (
	servePort int
	serveHost string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the cadre web UI",
	Long:  `Start a web server serving the cadre dashboard, agent chat, and visual pipeline composer.`,
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "host", "localhost", "host to bind to")
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(".")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger := telemetry.NewLogger(verbose)
	registerCustomTools(logger)
	eventBus := event.NewBus(logger)

	stateMgr, err := state.NewManager(cfg.State.Driver, cfg.State.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}
	defer stateMgr.Close()

	srv := server.New(cfg, stateMgr, eventBus, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	addr := fmt.Sprintf("%s:%d", serveHost, servePort)
	return srv.Start(ctx, addr)
}
