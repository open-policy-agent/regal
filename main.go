package main

import (
	"errors"
	"log"
	"os"

	"github.com/open-policy-agent/regal/cmd"

	_ "github.com/open-policy-agent/regal/pkg/roast/intern"
)

func main() {
	// Remove date and time from any `log.*` calls, as that doesn't add much of value here
	// Evaluate options for logging later
	log.SetFlags(0)

	if err := cmd.RootCommand.Execute(); err != nil {
		code := 1
		if e, ok := errors.AsType[cmd.ExitError](err); ok {
			code = e.Code()
		}

		os.Exit(code)
	}
}
