package cmd

import (
	"os"
	"path"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// RootCommand is the base CLI command that all subcommands are added to.
var RootCommand = &cobra.Command{
	Use:   path.Base(os.Args[0]),
	Short: "Regal",
	Long:  "Regal is a linter for Rego, with the goal of making your Rego magnificent!",
}

// relaxGC trades heap for speed in the commands that allocate in bulk and then
// exit. The language server is long-lived, and keeps the default.
func relaxGC() {
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(300)
	}
}
