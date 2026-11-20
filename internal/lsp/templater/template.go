package templater

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/io/paths"
	"github.com/open-policy-agent/regal/internal/lsp/cache"
	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/util/concurrent"
)

type Processor struct {
	log *log.Logger
	// files tracks files currently being templated to ensure
	// other updates are not processed while the file is being updated.
	files             *concurrent.Map[string, bool]
	cache             *cache.Cache
	workspace         workspace.Workspace
	lintJobs          chan types.LintJob
	regoVersionForURI func(fileURI string) ast.RegoVersion
	fixRenameChanges  func(fileURI string) ([]workspace.DocumentChange, error)
}

var (
	validPathComponentPattern = regexp.MustCompile(`^\w+[\w\-]*\w+$`)
	rpcTimeout                = 3 * time.Second
)

func New(l *log.Logger, w workspace.Workspace, c *cache.Cache) *Processor {
	return &Processor{
		files:     concurrent.MapOf(make(map[string]bool)),
		log:       l,
		workspace: w,
		cache:     c,
	}
}

func (p *Processor) WithLintJobs(j chan types.LintJob) *Processor {
	p.lintJobs = j

	return p
}

func (p *Processor) WithRegoVersionForURI(f func(fileURI string) ast.RegoVersion) *Processor {
	p.regoVersionForURI = f

	return p
}

func (p *Processor) WithFixRenameChanges(f func(fileURI string) ([]workspace.DocumentChange, error)) *Processor {
	p.fixRenameChanges = f

	return p
}

// Process handles the templating of a newly created Rego file.
func (p *Processor) Process(ctx context.Context, job types.FileJob) error {
	p.log.Debug("template worker received job: %s (reason: %s)", job.URI, job.Reason)

	// mark file as being templated to prevent race conditions
	p.files.Set(job.URI, true)
	defer p.files.Delete(job.URI)

	// disable the templating feature for files in the workspace root.
	if filepath.Dir(uri.ToPath(job.URI)) == p.workspace.Path() {
		return nil
	}

	// determine the new contents for the file, if permitted
	newContents, err := p.Contents(job.URI)
	if err != nil {
		return fmt.Errorf("failed to template new file: %w", err)
	}

	// set the contents of the new file in the cache immediately as
	// these must be update to date in order for fixRenameParams
	// to work
	p.cache.SetFileContents(job.URI, newContents)

	// determine if a rename is needed based on the new file package.
	// edits will be empty if no file rename is needed.
	additionalRenameEdits, err := p.fixRenameChanges(job.URI)
	if err != nil {
		return fmt.Errorf("failed to get rename params: %w", err)
	}

	// combine content changes with any additional rename changes
	params := workspace.NewApplyEditParams("Template new Rego file").
		WithTimeout(rpcTimeout).
		WithChanges(types.NewTextDocumentEdit(job.URI, types.ComputeEdits("", newContents))).
		WithChanges(additionalRenameEdits...)

	if err = p.workspace.ApplyEdit(ctx, params); err != nil {
		return fmt.Errorf("failed to apply workspace edit for templating new file: %w", err)
	}

	// finally, trigger a diagnostics run for the new contents
	p.lintJobs <- types.LintJob{Reason: "internal/templateNewFile"}

	return nil
}

func (p *Processor) Contents(fileURI string) (string, error) {
	path := uri.ToPath(fileURI)
	wsPath := p.workspace.Path()

	// this function should not be called with files in the root, but if it is,
	// then it is an error to prevent unwanted behavior.
	if filepath.Dir(path) == wsPath {
		return "", errors.New("this function does not template files in the workspace root")
	}

	content, ok := p.cache.GetFileContents(fileURI)
	if !ok {
		return "", fmt.Errorf("failed to get file contents for URI %q", fileURI)
	}

	if content != "" {
		return "", errors.New("file already has contents, templating not allowed")
	}

	if diskContent, err := os.ReadFile(path); err == nil && len(diskContent) > 0 {
		// then we found the file on disk
		return "", errors.New("file on disk already has contents, templating not allowed")
	}

	roots, err := config.GetPotentialRoots(path)
	if err != nil {
		return "", fmt.Errorf("failed to get potential roots during templating of new file: %w", err)
	}

	dir := filepath.Dir(path)

	// handle the case where the root is unknown by providing the server's root
	// dir as a defacto root. This allows templating of files when there is no
	// known root, but the package could be determined based on the file path
	// relative to the server's workspace root
	if len(roots) == 1 && roots[0] == dir {
		roots = []string{wsPath}
	} else {
		roots = append(roots, wsPath)
	}

	longestPrefixRoot := ""
	for _, root := range roots {
		if strings.HasPrefix(dir, root) && len(root) > len(longestPrefixRoot) {
			longestPrefixRoot = root
		}
	}

	if longestPrefixRoot == "" {
		return "", fmt.Errorf("failed to find longest prefix root for templating of new file: %s", path)
	}

	parts := slices.Compact(strings.Split(strings.TrimPrefix(dir, longestPrefixRoot), paths.Separator))

	var pkg string

	for _, part := range parts {
		switch {
		case part == "":
		case !validPathComponentPattern.MatchString(part):
			return "", fmt.Errorf("failed to template new file as package path contained invalid part: %s", part)
		case strings.Contains(part, "-"):
			pkg += fmt.Sprintf("[%q]", part)
		case pkg == "":
			pkg += part
		default:
			pkg += "." + part
		}
	}

	// if we are in the root, then we can use main as a default
	if pkg = cmp.Or(pkg, "main"); strings.HasSuffix(fileURI, "_test.rego") {
		pkg += "_test"
	}

	if p.regoVersionForURI(fileURI) == ast.RegoV0 {
		return fmt.Sprintf("package %s\n\nimport rego.v1\n", pkg), nil
	}

	return "package " + pkg + "\n\n", nil
}
