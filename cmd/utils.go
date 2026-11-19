package cmd

import (
	"cmp"
	"context"
	"log"
	"strings"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/io/paths"
)

type repeatedStringFlag struct {
	v     []string
	isSet bool
}

func (*repeatedStringFlag) Type() string {
	return "string"
}

func (f *repeatedStringFlag) String() string {
	return strings.Join(f.v, ",")
}

func (f *repeatedStringFlag) Set(s string) error {
	f.v = append(f.v, s)
	f.isSet = true

	return nil
}

// getSearchPath tries to determine which path from which we should start searching
// for a Regal directory, custom rules, and user config.
func getSearchPath(args []string) (searchPath string) {
	if len(args) > 0 {
		if searchPath = paths.Abs(args[0]); !rio.Exists(searchPath) {
			searchPath = "" // This is handled elsewhere — we don't need to fail here
		}
	}

	if searchPath = cmp.Or(searchPath, paths.WorkingDir); searchPath == "" {
		log.Println("could not determine config search directory - won't search for custom config or rules")
	}

	return searchPath
}

func getLinterContext(params lintAndFixParams) (context.Context, func()) {
	if to := params.timeout; to != 0 {
		return context.WithTimeout(context.Background(), to)
	}

	return context.Background(), func() {}
}
