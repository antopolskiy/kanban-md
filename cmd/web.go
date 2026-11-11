package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/antopolskiy/kanban-md/internal/web"
)

const defaultWebPort = 5741

func newWebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Open the board in a browser",
		Long: `Serves the current board on 127.0.0.1 and opens it in a browser.
The page live-reloads when task files change. Task files stay authoritative.

The default port is 5741. Use --port to choose another, including 5000.`,
		RunE: runWeb,
	}
	cmd.Flags().Int("port", defaultWebPort, "localhost port")
	cmd.Flags().Bool("no-browser", false, "print the URL without opening a browser")
	cmd.Flags().Bool("hide-empty-columns", false, "hide empty columns (overrides config)")
	cmd.Flags().Bool("show-empty-columns", false, "show empty columns (overrides config)")
	return cmd
}

func init() {
	rootCmd.AddCommand(newWebCommand())
}

func runWeb(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		if !isBoardNotFound(err) {
			return err
		}
		cfg, err = offerInitTUI()
		if err != nil {
			return err
		}
	}

	hideEmpty, err := resolveHideEmptyColumns(cmd, cfg)
	if err != nil {
		return err
	}
	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		return err
	}
	noBrowser, err := cmd.Flags().GetBool("no-browser")
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintln(os.Stderr, "Press Ctrl+C to stop.")
	return web.Serve(ctx, cfg, web.Options{
		Port:             port,
		OpenBrowser:      !noBrowser,
		HideEmptyColumns: hideEmpty,
		Stdout:           os.Stdout,
	})
}
