package cmd

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

func init() {
	capabilitiesCommand := &cobra.Command{
		Hidden: true,
		Use:    "capabilities",
		Short:  "Print the capabilities of Regal",
		Long:   "Show capabilities for Regal",
		RunE: func(*cobra.Command, []string) (err error) {
			return encoding.MarshalWriteLn(os.Stdout, io.Capabilities(), json.JoinOptions(
				jsontext.WithIndent("  "),
				jsontext.EscapeForHTML(true),
				json.OmitZeroStructFields(true),
			))
		},
	}

	RootCommand.AddCommand(capabilitiesCommand)
}
