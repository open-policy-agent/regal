package workspace

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/client"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
)

type (
	Workspace struct {
		uri    string
		path   string
		fs     fs.FS
		client client.Client
	}
	DocumentChange interface {
		AppendJSON(bs []byte) []byte
	}
	ApplyEditParams struct {
		Label string `json:"label"`
		Edit  struct {
			DocumentChanges []DocumentChange `json:"documentChanges"`
		} `json:"edit"`
		timeout time.Duration
	}
	ApplyEditResult struct {
		Applied       bool   `json:"applied"`
		FailureReason string `json:"failureReason,omitempty"`
		FailedChange  *uint  `json:"failedChange,omitempty"`
	}
)

func New(workspaceURI string) Workspace {
	path := uri.ToPath(workspaceURI)

	return Workspace{
		uri:    workspaceURI,
		path:   path,
		fs:     os.DirFS(path),
		client: client.NewGeneric(),
	}
}

// URI returns the root URI of the workspace.
func (w Workspace) URI(join ...string) string {
	if len(join) == 0 {
		return w.uri
	}

	if filepath.IsAbs(join[0]) {
		return w.client.URIFromPath(filepath.Join(join...))
	}

	return w.client.URIFromRelativePath(filepath.Join(join...), w.uri)
}

// Path returns the root path of the workspace, optionally joined with additional path segments.
func (w Workspace) Path(join ...string) string {
	if len(join) == 0 {
		return w.path
	}

	return filepath.Join(slices.Insert(join, 0, w.path)...)
}

// RelativePath returns the path of the given file URI relative to the workspace root.
func (w Workspace) RelativePath(fileURI string) string {
	return uri.ToRelativePath(fileURI, w.uri)
}

// FS returns a file system rooted at the workspace path.
func (w Workspace) FS() fs.FS {
	return w.fs
}

// Client returns the client associated with the workspace.
func (w Workspace) Client() client.Client {
	return w.client
}

// WithClient returns a copy of the workspace with the provided client.
func (w Workspace) WithClient(client client.Client) Workspace {
	w.client = client

	return w
}

// WithFS returns a copy of the workspace with the provided file system.
// This method is mainly meant for testing purposes.
func (w Workspace) WithFS(fs fs.FS) Workspace {
	w.fs = fs

	return w
}

func (w Workspace) ApplyEdit(ctx context.Context, params ApplyEditParams) error {
	conn := w.Client().Connection()
	if conn == nil {
		return errors.New("attempted workspace/applyEdit without a client connection")
	}

	rpcCtx, rpcCancel := context.WithTimeout(ctx, cmp.Or(params.timeout, 30*time.Second))

	var res ApplyEditResult

	err := conn.Call(rpcCtx, "workspace/applyEdit", params, &res)
	if err == nil && !res.Applied {
		err = fmt.Errorf("workspace edit was not applied: %s", res.FailureReason)
	}

	rpcCancel()

	return err
}

func NewApplyEditParams(label string) ApplyEditParams {
	return ApplyEditParams{Label: label}
}

func (e ApplyEditParams) WithChanges(changes ...DocumentChange) ApplyEditParams {
	e.Edit.DocumentChanges = append(e.Edit.DocumentChanges, changes...)

	return e
}

func (e ApplyEditParams) WithTimeout(timeout time.Duration) ApplyEditParams {
	e.timeout = timeout

	return e
}

func (e ApplyEditParams) MarshalJSON() ([]byte, error) {
	buf := append(strconv.AppendQuote([]byte(`{"label":`), e.Label), `,"edit":{"documentChanges":[`...)
	for i, change := range e.Edit.DocumentChanges {
		buf = change.AppendJSON(buf)
		if i < len(e.Edit.DocumentChanges)-1 {
			buf = append(buf, ',')
		}
	}

	return append(buf, "]}}"...), nil
}
