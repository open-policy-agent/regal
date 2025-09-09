package cmd

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

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
			defer cancel()

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

			opts := &lsp.LanguageServerOptions{Logger: log.NewLogger(log.LevelMessage, os.Stderr)}
			ls := lsp.NewLanguageServer(ctx, opts)

			conf := connection.LoggingConfig{Logger: opts.Logger, LogInbound: verbose, LogOutbound: verbose}
			conn := connection.New(ctx, ls.Handle, &connection.Options{LoggingConfig: conf})
			defer conn.Close()

			ls.SetConn(conn)
			go ls.StartDiagnosticsWorker(ctx)
			go ls.StartHoverWorker(ctx)
			go ls.StartCommandWorker(ctx)
			go ls.StartConfigWorker(ctx)
			go ls.StartWorkspaceStateWorker(ctx)
			go ls.StartTemplateWorker(ctx)
			go ls.StartWebServer(ctx)

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			select {
			case <-conn.DisconnectNotify():
				fmt.Fprintln(os.Stderr, "Connection closed")
			case sig := <-sigChan:
				fmt.Fprintln(os.Stderr, "signal: ", sig.String())
			}

			return nil
		}),
	}

	languageServerCommand.Flags().BoolVarP(&verbose, "verbose", "v", verbose, "Enable verbose logging")

	addPprofFlag(languageServerCommand.Flags())

	RootCommand.AddCommand(languageServerCommand)
}
