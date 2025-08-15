package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/pkg/config"
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
		searchPath = args[0]
		if !filepath.IsAbs(searchPath) {
			searchPath, _ = filepath.Abs(args[0])
		}

		if !rio.Exists(searchPath) {
			searchPath = "" // This is handled elsewhere — we don't need to fail here
		}
	}

	searchPath = cmp.Or(searchPath, rio.Getwd())
	if searchPath == "" {
		log.Println("could not determine config search directory - won't search for custom config or rules")
	}

	return searchPath
}

func readUserConfig(params lintAndFixParams, searchPath string) (userConfig *os.File, err error) {
	if params.configFile != "" {
		userConfig, err = os.Open(params.configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file %w", err)
		}

		if params.debug {
			log.Printf("found user config file: %s", userConfig.Name())
		}

		return userConfig, nil
	}

	if params.debug {
		log.Println("no user-provided config file found, will use the default config")
	}

	userConfig, err = config.FindConfig(searchPath)
	if err != nil {
		// if no config was found, attempt to load the user's global config if it exists
		if globalConfigDir := config.GlobalConfigDir(false); globalConfigDir != "" {
			globalConfigFile := filepath.Join(globalConfigDir, "config.yaml")

			if userConfig, err = os.Open(globalConfigFile); err != nil {
				return nil, fmt.Errorf("failed to open global config file %w", err)
			}
		}
	}

	return userConfig, err
}

func loadUserConfig(params lintAndFixParams, root string) (cfg config.Config, path string, err error) {
	file, err := readUserConfig(params, root)
	if err != nil && params.configFile != "" {
		return cfg, "", fmt.Errorf("user-provided config file %s not found: %w", params.configFile, err)
	}

	if file == nil {
		return config.Config{}, "", nil // No user config provided, use default
	}

	defer rio.CloseFileIgnore(file)

	cfg, err = config.FromFile(file)
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			log.Printf("user config file %q is empty, will use the default config", file.Name())
		case params.configFile != "":
			return cfg, "", fmt.Errorf("failed to decode user config from %s: %w", params.configFile, err)
		default:
			return cfg, "", fmt.Errorf("failed to decode user config: %w", err)
		}
	}

	return cfg, file.Name(), nil
}

func getLinterContext(params lintAndFixParams) (context.Context, func()) {
	ctx := context.Background()
	if to := params.timeout; to != 0 {
		return context.WithTimeout(ctx, to)
	}

	return ctx, func() {}
}
