package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-policy-agent/regal/internal/lsp/input"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
)

type (
	debug struct {
		workspace workspace.Workspace
		input     *input.Manager
		port      uint16
		args      types.CommandArgs
	}
	debugParams struct {
		Type        string `json:"type"`
		Name        string `json:"name"`
		Request     string `json:"request"`
		Command     string `json:"command"`
		Query       string `json:"query"`
		EnablePrint bool   `json:"enablePrint"`
		StopOnEntry bool   `json:"stopOnEntry"`
		InputPath   string `json:"inputPath,omitempty"`
		Port        uint16 `json:"port,omitempty"`
	}
)

// NewDebug returns a [Command] to start a debug session for the given target and query.
func NewDebug(ws workspace.Workspace, input *input.Manager, port uint16, args types.CommandArgs) Command {
	return &debug{workspace: ws, input: input, port: port, args: args}
}

// Run executes the debug command, which tells the client to start a debug session for the given target and query.
func (d *debug) Run(ctx context.Context) (err error) {
	client := d.workspace.Client()
	if !client.InitOptions.EnableDebugCodelens {
		return errors.New("regal.debug command called but client does not support debug functionality")
	}

	if d.args.Target == "" || d.args.Query == "" {
		return fmt.Errorf("expected query and optionally target, got target %q, query %q", d.args.Target, d.args.Query)
	}

	// FindForPath returns a workspace-relative path (or ""); the OPA debugger
	// resolves inputPath via os.Open against its own CWD, so pass an absolute path.
	var inputPath string
	if rel := d.input.FindForPath(d.args.Target); rel != "" {
		inputPath = d.workspace.Path(rel)
	}

	params := debugParams{
		Type:        "opa-debug",
		Request:     "launch",
		Command:     "eval",
		InputPath:   inputPath,
		Name:        d.args.Query,
		Query:       d.args.Query,
		EnablePrint: true,
		StopOnEntry: true,
		Port:        d.port,
	}

	rpcCtx, rpcCancel := context.WithTimeout(ctx, rpcTimeout)
	if err = client.Connection().Call(rpcCtx, "regal/startDebugging", params, nil); err != nil {
		err = fmt.Errorf("regal/startDebugging failed: %w", err)
	}

	rpcCancel()

	return err
}
