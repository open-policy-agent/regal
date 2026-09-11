package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-policy-agent/opa/v1/debug"
	"github.com/open-policy-agent/opa/v1/logging"

	"github.com/open-policy-agent/regal/internal/dap"
	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/util"
)

func init() {
	verboseLogging, serverMode := false, false
	address := "localhost:4712"

	debug := &cobra.Command{
		Use:   "debug",
		Short: "Run the Regal OPA Debugger",
		Long:  `Start the Regal OPA debugger and listen on stdin/stdout for client editor messages.`,

		RunE: wrapProfiling(func([]string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			logger := dap.NewDebugLogger(logging.New(), logging.Debug)
			if verboseLogging {
				logger.Local.SetLevel(logging.Debug)
			}

			if serverMode {
				return dap.NewServer(address, logger).Start(ctx)
			}

			return startCmd(ctx, logger)
		}),
	}

	debug.Flags().BoolVarP(&verboseLogging, "verbose", "v", verboseLogging, "Enable verbose logging")
	debug.Flags().BoolVarP(&serverMode, "server", "s", serverMode, "Start the debugger in server mode")
	debug.Flags().StringVarP(&address, "address", "a", address, "Address to listen on. For use with --server flag.")

	RootCommand.AddCommand(debug)
}

func startCmd(ctx context.Context, logger *dap.DebugLogger) error {
	protoManager := dap.NewProtocolManager(logger.Local)
	logger.ProtocolManager = protoManager

	debugger := debug.NewDebugger(debug.SetEventHandler(dap.NewEventHandler(protoManager)), debug.SetLogger(logger))
	conn := io.NewReadWriteCloser(os.Stdin, os.Stdout)
	s := dap.NewState(protoManager, debugger, logger)

	return util.WrapErr(protoManager.Start(ctx, conn, s.HandleMessage), "failed to handle connection")
}
