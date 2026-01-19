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

	"github.com/open-policy-agent/regal/internal/ogre"
	"github.com/open-policy-agent/regal/internal/semver"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
	"github.com/open-policy-agent/regal/pkg/roast/rast"

	_ "embed"
)

const CheckVersionDisableEnvVar = "REGAL_DISABLE_VERSION_CHECK"

var (
	//go:embed update.rego
	updateModule string
	query        = ast.MustParseBody("result = data.update.check")
)

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

func CheckAndWarn(opts Options, w io.Writer) {
	// only perform the version check on binaries with production semvers set
	if _, err := semver.Parse(opts.CurrentVersion); err != nil {
		if opts.Debug {
			fmt.Fprintf(w, "Skipping version check: invalid semver %s: %v\n", opts.CurrentVersion, err)
		}

		return
	}

	latestVersion, cacheIsStale := getLatestCachedVersionAndCheckStale(opts)
	ctx := context.Background()
	mod := map[string]*ast.Module{"update.rego": ast.MustParseModule(updateModule)}

	q, err := ogre.New(query).WithModules(mod).Prepare(ctx)
	if err != nil {
		if opts.Debug {
			w.Write(fmt.Appendf(nil, "failed to prepare update query: %v", err))
		}

		return
	}

	input := ast.NewObject(
		rast.Item("current_version", ast.InternedTerm(opts.CurrentVersion)),
		rast.Item("latest_version", ast.InternedTerm(latestVersion)),
		rast.Item("cta_url_prefix", ast.InternedTerm(opts.CTAURLPrefix)),
		rast.Item("release_server_host", ast.InternedTerm(opts.ReleaseServerHost)),
		rast.Item("release_server_path", ast.InternedTerm(opts.ReleaseServerPath)),
	)

	var result decision

	err = q.Evaluator().WithInput(input).WithResultHandler(func(qr ast.Value) (err error) {
		if obj, ok := qr.(ast.Object); ok {
			result = decision{
				NeedsUpdate:   rast.GetBool(obj, "needs_update"),
				LatestVersion: rast.GetString(obj, "latest_version"),
				CTA:           rast.GetString(obj, "cta"),
			}

			return nil
		}

		return errors.New("no result set")
	}).Eval(ctx)
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
	} else if opts.Debug {
		w.Write([]byte("Regal is up to date"))
	}
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
