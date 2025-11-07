package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-policy-agent/regal/internal/io"
)

func init() {
	capabilitiesCommand := &cobra.Command{
		Hidden: true,
		Use:    "capabilities",
		Short:  "Print the capabilities of Regal",
		Long:   "Show capabilities for Regal",
		RunE: func(*cobra.Command, []string) error {
			bs, err := json.MarshalIndent(io.Capabilities(), "", "  ")
			if err == nil {
				_, err = os.Stdout.Write(append(bs, '\n'))
			}

			return err
		},
	}

	RootCommand.AddCommand(capabilitiesCommand)
}
