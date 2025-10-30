//nolint:errcheck
package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"

	_ "embed"
)

//go:embed update.rego
var updateModule string

const CheckVersionDisableEnvVar = "REGAL_DISABLE_VERSION_CHECK"

type Options struct {
	CurrentTime       time.Time
	CurrentVersion    string
	StateDir          string
	ReleaseServerHost string
	ReleaseServerPath string
	CTAURLPrefix      string
	Debug             bool
}

type latestVersionFileContents struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

type decision struct {
	NeedsUpdate   bool   `json:"needs_update"`
	LatestVersion string `json:"latest_version"`
	CTA           string `json:"cta"`
}

var query = ast.MustParseBody("result := data.update.check")

func CheckAndWarn(opts Options, w io.Writer) {
	// this is a shortcut heuristic to avoid version checking when in dev/test etc.
	if !strings.HasPrefix(opts.CurrentVersion, "v") {
		return
	}

	latestVersion, err := getLatestCachedVersion(opts)
	if err != nil {
		if opts.Debug {
			w.Write([]byte(err.Error()))
		}

		return
	}

	regoArgs := []func(*rego.Rego){
		rego.Module("update.rego", updateModule),
		rego.ParsedQuery(query),
		rego.ParsedInput(ast.NewObject(
			ast.Item(ast.StringTerm("current_version"), ast.StringTerm(opts.CurrentVersion)),
			ast.Item(ast.StringTerm("latest_version"), ast.StringTerm(latestVersion)),
			ast.Item(ast.StringTerm("cta_url_prefix"), ast.StringTerm(opts.CTAURLPrefix)),
			ast.Item(ast.StringTerm("release_server_host"), ast.StringTerm(opts.ReleaseServerHost)),
			ast.Item(ast.StringTerm("release_server_path"), ast.StringTerm(opts.ReleaseServerPath)),
		)),
	}

	rs, err := rego.New(regoArgs...).Eval(context.Background())
	if err != nil {
		if opts.Debug {
			w.Write([]byte(err.Error()))
		}

		return
	}

	result, err := resultSetToDecision(rs)
	if err != nil {
		if opts.Debug {
			w.Write([]byte(err.Error()))
		}

		return
	}

	if result.NeedsUpdate {
		if err = saveLatestCachedVersion(opts, result.LatestVersion); err != nil && opts.Debug {
			w.Write([]byte(err.Error()))
		}

		w.Write([]byte(result.CTA))

		return
	}

	if opts.Debug {
		w.Write([]byte("Regal is up to date"))
	}
}

func resultSetToDecision(rs rego.ResultSet) (decision, error) {
	if len(rs) == 0 || rs[0].Bindings["result"] == nil {
		return decision{}, errors.New("no result set")
	}

	return util.Wrap(encoding.JSONRoundTripTo[decision](rs[0].Bindings["result"]))("failed to decode result set")
}

func getLatestCachedVersion(opts Options) (string, error) {
	if opts.StateDir != "" {
		// first, attempt to get the file from previous invocations to save on remote calls
		latestVersionFilePath := filepath.Join(opts.StateDir, "latest_version.json")

		if file, err := os.Open(latestVersionFilePath); err == nil {
			defer file.Close()

			var preExistingState latestVersionFileContents
			if err := encoding.JSON().NewDecoder(file).Decode(&preExistingState); err != nil {
				return "", fmt.Errorf("failed to decode existing version state file: %w", err)
			}

			if opts.CurrentTime.Sub(preExistingState.CheckedAt) < 3*24*time.Hour {
				return preExistingState.LatestVersion, nil
			}
		}
	}

	return "", nil
}

func saveLatestCachedVersion(opts Options, latestVersion string) error {
	if opts.StateDir != "" {
		content := latestVersionFileContents{LatestVersion: latestVersion, CheckedAt: opts.CurrentTime}

		bs, err := encoding.JSON().MarshalIndent(content, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal state file: %w", err)
		}

		if err = os.WriteFile(filepath.Join(opts.StateDir, "latest_version.json"), bs, 0o600); err != nil {
			return fmt.Errorf("failed to write state file: %w", err)
		}
	}

	return nil
}
