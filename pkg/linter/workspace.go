package linter

import (
	"path/filepath"

	rio "github.com/open-policy-agent/regal/internal/io"
	regalmetrics "github.com/open-policy-agent/regal/internal/metrics"
	"github.com/open-policy-agent/regal/pkg/config"
)

// WorkspaceOptions are the options of WithWorkspace.
type WorkspaceOptions struct {
	// ConfigFile is the config file to use, instead of searching for one.
	ConfigFile string
	// SkipCustomRules leaves out the rules in the .regal directory, as when custom rules are given another way.
	SkipCustomRules bool
	Debug           bool
}

// WithWorkspace configures the linter for the workspace at searchPath. The path prefix is the directory with the
// .regal directory found searching upwards from there, and the custom rules are the ones in that .regal directory.
// The user config is the given one, or else the one found for searchPath. It is returned with its path, for callers
// that need them too. Both are empty if there is no user config.
func (l Linter) WithWorkspace(searchPath string, opts WorkspaceOptions) (Linter, *config.Config, string, error) {
	l.startTimer(regalmetrics.RegalConfigSearch)

	if searchPath != "" {
		if regalPath, err := config.FindRegalDirectoryPath(searchPath); err == nil {
			l = l.WithPathPrefix(filepath.Dir(regalPath))

			if rulesDir := filepath.Join(regalPath, "rules"); !opts.SkipCustomRules && rio.IsDir(rulesDir) {
				l = l.WithCustomRulesPaths(rulesDir)
			}
		}
	}

	l.stopTimer(regalmetrics.RegalConfigSearch)
	l.startTimer(regalmetrics.RegalConfigParse)

	userConfig, userConfigPath, err := config.LoadUser(opts.ConfigFile, searchPath, opts.Debug)

	l.stopTimer(regalmetrics.RegalConfigParse)

	if err != nil {
		return l, nil, userConfigPath, err
	}

	if userConfig != nil {
		l = l.WithUserConfig(userConfig)
	}

	return l, userConfig, userConfigPath, nil
}
