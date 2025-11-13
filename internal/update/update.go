//nolint:errcheck
package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/open-policy-agent/regal/internal/semver"
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
	// only perform the version check on binaries with production semvers set
	if _, err := semver.Parse(opts.CurrentVersion); err != nil {
		if opts.Debug {
			fmt.Fprintf(w, "Skipping version check: invalid semver %s: %v\n", opts.CurrentVersion, err)
		}

		return
	}

	latestVersion, cacheIsStale := getLatestCachedVersionAndCheckStale(opts)

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

	// update cache if it was stale and we made a remote fetch
	if cacheIsStale && opts.StateDir != "" && result.LatestVersion != "" {
		content := latestVersionFileContents{LatestVersion: result.LatestVersion, CheckedAt: opts.CurrentTime}
		if bs, err := encoding.JSON().MarshalIndent(content, "", "  "); err == nil {
			os.WriteFile(opts.StateDir+"/latest_version.json", bs, 0o600)
		}
	}

	if result.NeedsUpdate {
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

func getLatestCachedVersionAndCheckStale(opts Options) (string, bool) {
	if opts.StateDir == "" {
		return "", true // when missing it's 'stale', but we will also not update later.
	}

	latestVersionFilePath := filepath.Join(opts.StateDir, "latest_version.json")

	file, err := os.Open(latestVersionFilePath)
	if err != nil {
		return "", true // no file means stale
	}
	defer file.Close()

	var preExistingState latestVersionFileContents
	if err := encoding.JSON().NewDecoder(file).Decode(&preExistingState); err != nil {
		return "", true // can't decode means stale
	}

	isStale := opts.CurrentTime.Sub(preExistingState.CheckedAt) >= 3*24*time.Hour

	if isStale {
		return "", true // if stale, return empty to force remote fetch
	}

	return preExistingState.LatestVersion, false
}
