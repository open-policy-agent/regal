package cmd

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/lsp"
	"github.com/open-policy-agent/regal/internal/lsp/connection"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/pkg/version"
)

func init() {
	verbose := false

	languageServerCommand := &cobra.Command{
		Use:   "language-server",
		Short: "Run the Regal Language Server",
		Long:  `Start the Regal Language Server and listen on stdin/stdout for client editor messages.`,

		RunE: wrapProfiling(func([]string) error {
			ctx, cancel := context.WithCancel(context.Background())

			if exe, err := os.Executable(); err != nil {
				fmt.Fprintln(os.Stderr, "error getting executable:", err)
			} else {
				msg := "Regal Language Server (path: %s, version: %s)\n"
				fmt.Fprintf(os.Stderr, msg, exe, cmp.Or(version.Version, "Unknown"))
			}

			if os.Getenv("REGAL_DEBUG") != "" {
				fmt.Fprintln(os.Stderr, "Debug mode enabled")

				verbose = true
			}

			logLevel := log.LevelMessage
			if verbose {
				logLevel = log.LevelDebug
			}

			opts := &lsp.LanguageServerOptions{
				Logger:       log.NewLogger(logLevel, os.Stderr),
				FeatureFlags: lsp.DefaultServerFeatureFlags(),
			}

			ls := lsp.NewLanguageServer(ctx, opts)

			conf := connection.LoggingConfig{Logger: opts.Logger, LogInbound: verbose, LogOutbound: verbose}
			copt := &connection.Options{LoggingConfig: conf}

			conn := connection.NewWithOptions(ctx, rio.NewReadWriteCloser(os.Stdin, os.Stdout), ls.Handle, copt)
			defer conn.Close()

			ls.SetConn(conn)

			ls.StartDiagnosticsWorker(ctx)
			ls.StartTestLocationsWorker(ctx)
			ls.StartCommandWorker(ctx)
			ls.StartConfigWorker(ctx)
			ls.StartWorkspaceStateWorker(ctx)
			ls.StartTemplateWorker(ctx)
			ls.StartQueryCacheWorker(ctx)
			ls.StartWebServer(ctx)

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			select {
			case <-conn.DisconnectNotify():
				fmt.Fprintln(os.Stderr, "Connection closed")
			case sig := <-sigChan:
				fmt.Fprintln(os.Stderr, "signal: ", sig.String())
			}

			cancel()

			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()

			if err := ls.Shutdown(shutdownCtx); err != nil {
				fmt.Fprintln(os.Stderr, "shutdown error:", err)
			}

			return nil
		}),
	}

	languageServerCommand.Flags().BoolVarP(&verbose, "verbose", "v", verbose, "Enable verbose logging")

	addPprofFlag(languageServerCommand.Flags())

	RootCommand.AddCommand(languageServerCommand)
}
