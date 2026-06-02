package lsp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/explorer"
	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/lsp/clients"
	"github.com/open-policy-agent/regal/internal/lsp/testgen"
	"github.com/open-policy-agent/regal/internal/lsp/types"
	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/lsp/workspace"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/config/modify"
	"github.com/open-policy-agent/regal/pkg/fixer"
	"github.com/open-policy-agent/regal/pkg/fixer/fileprovider"
	"github.com/open-policy-agent/regal/pkg/fixer/fixes"
	"github.com/open-policy-agent/regal/pkg/report"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

func (l *LanguageServer) StartCommandWorker(ctx context.Context) {
	l.workersWg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			case params := <-l.commandRequest:
				if params.Command == "regal.explorer" {
					if err := l.handleExplorerCommand(ctx, params); err != nil {
						l.log.Message("failed to handle explorer command: %s", err)
					}

					continue
				}

				if params.Command == "regal.createTest" {
					if err := l.handleCreateTestCommand(ctx, params); err != nil {
						l.log.Message("failed to handle createTest command: %s", err)
					}

					continue
				}

				// Handle all other commands (they use string arguments)
				if len(params.Arguments) != 1 {
					l.log.Message("expected one argument, got %d", len(params.Arguments))

					continue
				}

				jsonData, ok := params.Arguments[0].(string)
				if !ok {
					l.log.Message("expected argument to be a json.RawMessage, got %T", params.Arguments[0])

					continue
				}

				var (
					editParams workspace.ApplyEditParams
					args       types.CommandArgs
					err        error
				)

				if err = encoding.JSON().Unmarshal(outil.StringToByteSlice(jsonData), &args); err != nil {
					l.log.Message("failed to unmarshal command arguments: %s", err)

					continue
				}

				switch params.Command {
				case "regal.fix.opa-fmt":
					editParams, err = l.fixEditParams("Format using opa fmt", fixFmt, args)
				case "regal.fix.use-rego-v1":
					editParams, err = l.fixEditParams("Format for Rego v1 using opa-fmt", fixUseRegoV1, args)
				case "regal.fix.use-assignment-operator":
					editParams, err = l.fixEditParams("Replace = with := in assignment", fixUseAssignmentOperator, args)
				case "regal.fix.no-whitespace-comment":
					editParams, err = l.fixEditParams("Format comment to have leading whitespace", fixNoWhitespaceComment, args)
				case "regal.fix.non-raw-regex-pattern":
					editParams, err = l.fixEditParams("Replace \" with ` in regex pattern", fixNonRawRegexPattern, args)
				case "regal.fix.prefer-equals-comparison":
					editParams, err = l.fixEditParams("Replace = with == in comparison", fixPreferEqualsComparison, args)
				case "regal.fix.constant-condition":
					editParams, err = l.fixEditParams("Remove constant condition", fixConstantCondition, args)
				case "regal.fix.redundant-existence-check":
					editParams, err = l.fixEditParams("Remove redundant existence check", fixRedundantExistence, args)
				case "regal.fix.directory-package-mismatch":
					changes, err := l.fixRenameChanges(args.Target)
					if err != nil {
						l.log.Message("failed to fix directory package mismatch: %s", err)

						break
					}

					edit := workspace.NewApplyEditParams("Rename file to match package path").
						WithTimeout(rpcTimeout).
						WithChanges(changes...)

					if err = l.Workspace().ApplyEdit(ctx, edit); err != nil {
						l.log.Message("failed workspace/applyEdit request: %s", err.Error())
					}

					// handle this ourselves as it's a rename and not a content edit
					continue
				case "regal.eval":
					err = l.handleEvalCommand(ctx, args)
				case "regal.debug":
					if !l.Workspace().Client().InitOptions.EnableDebugCodelens {
						l.log.Message("regal.debug command called but client does not support debug functionality")

						break
					}

					if !l.featureFlags.DebugProvider {
						l.log.Message("regal.debug command called but disabled in server")

						break
					}

					if args.Target == "" || args.Query == "" {
						l.log.Message("expected command target and query, got target %q, query %q", args.Target, args.Query)

						break
					}

					// FindForPath returns a workspace-relative path (or ""); the OPA debugger
					// resolves inputPath via os.Open against its own CWD, so pass an absolute path.
					var inputPath string
					if rel := l.input.FindForPath(args.Target); rel != "" {
						inputPath = l.Workspace().Path(rel)
					}

					responseParams := map[string]any{
						"type":        "opa-debug",
						"name":        args.Query,
						"request":     "launch",
						"command":     "eval",
						"query":       args.Query,
						"enablePrint": true,
						"stopOnEntry": true,
						"inputPath":   inputPath,
					}

					responseResult := map[string]any{}

					// Use a timeout context for RPC to ensure it completes even during shutdown
					rpcCtx, rpcCancel := context.WithTimeout(context.Background(), rpcTimeout)

					//nolint:contextcheck
					if err = l.conn.Call(rpcCtx, "regal/startDebugging", responseParams, &responseResult); err != nil {
						l.log.Message("regal/startDebugging failed: %s", err.Error())
					}

					rpcCancel()
				case "regal.config.disable-rule":
					if err = l.handleIgnoreRuleCommand(ctx, args); err != nil {
						l.log.Message("failed to ignore rule: %s", err)
					}

					continue // handle this ourselves as it's a config edit
				}

				if err != nil {
					l.log.Message("command failed: %s", err)
					l.window.ShowMessage(ctx, types.ErrorMessage, err.Error())
				} else if len(editParams.Edit.DocumentChanges) > 0 {
					if err := l.Workspace().ApplyEdit(ctx, editParams); err != nil {
						l.log.Message("failed workspace/applyEdit request: %s", err.Error())
					}
				}
			}
		}
	})
}

func (l *LanguageServer) fixEditParams(
	label string,
	fix fixes.Fix,
	args types.CommandArgs,
) (workspace.ApplyEditParams, error) {
	var editParams workspace.ApplyEditParams

	oldContent, ok := l.cache.GetFileContents(args.Target)
	if !ok {
		return editParams, fmt.Errorf("could not get file contents for uri %q", args.Target)
	}

	ws := l.Workspace()

	rto := &fixes.RuntimeOptions{BaseDir: ws.Path()}
	if args.Diagnostic != nil {
		rto.Locations = []report.Location{{
			Row:    util.SafeUintToInt(args.Diagnostic.Range.Start.Line + 1),
			Column: util.SafeUintToInt(args.Diagnostic.Range.Start.Character + 1),
			End: &report.Position{
				Row:    util.SafeUintToInt(args.Diagnostic.Range.End.Line + 1),
				Column: util.SafeUintToInt(args.Diagnostic.Range.End.Character + 1),
			},
		}}
	}

	res, err := fix.Fix(&fixes.FixCandidate{Filename: filepath.Base(uri.ToPath(args.Target)), Contents: oldContent}, rto)
	if err != nil {
		return editParams, fmt.Errorf("failed to fix: %w", err)
	} else if len(res) == 0 {
		return editParams, nil
	}

	var edits []types.TextEdit

	if ws.Client().Identifier == clients.IdentifierIntelliJ {
		// IntelliJ clients need a single edit that replaces the entire file
		numLines := util.NumLines(oldContent)
		line, _ := util.Line(oldContent, numLines)

		edits = []types.TextEdit{{Range: types.RangeBetween(0, 0, numLines-1, len(line)), NewText: res[0].Contents}}
	} else {
		// Other clients use the standard diff-based edits
		edits = ComputeEdits(oldContent, res[0].Contents)
	}

	editParams = workspace.NewApplyEditParams(label).WithChanges(types.NewTextDocumentEdit(args.Target, edits))

	return editParams, nil
}

func (l *LanguageServer) fixRenameChanges(fileURI string) ([]workspace.DocumentChange, error) {
	ws := l.Workspace()

	roots, err := config.GetPotentialRoots(ws.Path())
	if err != nil {
		return nil, fmt.Errorf("failed to get potential roots: %w", err)
	}

	fix := &fixes.DirectoryPackageMismatch{}

	// the default for the LSP is to rename on conflict
	f := fixer.NewFixer().RegisterRoots(roots...).RegisterFixes(fix).SetOnConflictOperation(fixer.OnConflictRename)

	violations := []report.Violation{{Title: fix.Name(), Location: report.Location{File: uri.ToPath(fileURI)}}}
	cfprovider := fileprovider.NewCacheFileProvider(l.cache, ws.Client().Identifier)

	fixReport, err := f.FixViolations(violations, cfprovider, l.getLoadedConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to fix violations: %w", err)
	}

	ff := fixReport.FixedFiles()
	if len(ff) == 0 {
		return nil, nil
	}

	// find the new file and the old location
	var fixedFile, oldFile string

	var found bool

	for _, f := range ff {
		if oldFile, found = fixReport.OldPathForFile(f); found {
			fixedFile = f

			break
		}
	}

	if !found {
		return nil, errors.New("failed to find fixed file's old location")
	}

	oldURI, newURI := ws.URI(oldFile), ws.URI(fixedFile)

	// is the newURI still in the root?
	if !strings.HasPrefix(newURI, ws.URI()) {
		return nil, errors.New(
			"cannot move file out of workspace root, consider using a workspace config or manually setting roots")
	}

	// are there old dirs?
	dirs, err := rio.DirCleanUpPaths(uri.ToPath(oldURI), []string{
		ws.Path(),          // stop at the root
		uri.ToPath(newURI), // also preserve any dirs needed for the new file
	})
	if err != nil {
		return nil, fmt.Errorf("failed to determine empty directories post rename: %w", err)
	}

	renopts := &types.RenameFileOptions{Overwrite: false, IgnoreIfExists: false}
	changes := append(make([]workspace.DocumentChange, 0, len(dirs)+1),
		types.RenameFile{Kind: "rename", OldURI: oldURI, NewURI: newURI, Options: renopts},
	)

	delopts := &types.DeleteFileOptions{Recursive: true, IgnoreIfNotExists: true}
	for _, dir := range dirs {
		changes = append(changes, types.DeleteFile{Kind: "delete", URI: ws.URI(dir), Options: delopts})
	}

	l.cache.Delete(oldURI)

	return changes, nil
}

func (l *LanguageServer) handleIgnoreRuleCommand(_ context.Context, args types.CommandArgs) error {
	if args.Diagnostic == nil {
		return errors.New("diagnostic is required to ignore rule")
	}

	// find or create config file
	var configPath string

	workspace := l.Workspace()
	if configFile, err := config.Find(workspace.Path()); err == nil {
		defer configFile.Close()

		configPath = configFile.Name()
	} else {
		regalDir := workspace.Path(".regal")
		if err := os.MkdirAll(regalDir, 0o755); err != nil {
			return fmt.Errorf("failed to create .regal directory: %w", err)
		}

		configPath = workspace.Path(".regal", "config.yaml")
	}

	var currentContent string
	if content, err := os.ReadFile(configPath); err == nil {
		currentContent = string(content)
	}

	// default to empty set of rules
	if strings.TrimSpace(currentContent) == "" {
		currentContent = "rules: {}\n"
	}

	category := strings.TrimPrefix(*args.Diagnostic.Source, "regal/")
	path := []string{"rules", category, args.Diagnostic.Code, "level"}

	newContent, err := modify.SetKey(currentContent, path, "ignore")
	if err != nil {
		return fmt.Errorf("failed to modify config: %w", err)
	}

	// TODO: we need to trigger a config reload so that the server starts using
	// the new config immediately. Currently, the server will pick up the config
	// change through file system watchers only.
	return util.WrapErr(os.WriteFile(configPath, []byte(newContent), 0o600), "failed to write config file")
}

func (l *LanguageServer) handleExplorerCommand(ctx context.Context, params types.ExecuteCommandParams) error {
	workspace := l.Workspace()
	if !workspace.Client().InitOptions.EnableExplorer {
		l.log.Message("regal.explorer command called but client does not support explorer functionality")

		return errors.New("client does not support explorer functionality")
	}

	var args types.ExplorerCommandArgs

	if len(params.Arguments) > 0 {
		arg, ok := params.Arguments[0].(map[string]any)
		if !ok {
			return fmt.Errorf(
				"failed to unmarshal regal.explorer command arguments, expected object, got %T",
				params.Arguments[0],
			)
		}

		args = types.ExplorerCommandArgs{
			Target:      util.MapGet[string](arg, "target"),
			Strict:      util.MapGet[bool](arg, "strict"),
			Annotations: util.MapGet[bool](arg, "annotations"),
			Print:       util.MapGet[bool](arg, "print"),
			Format:      util.MapGet[bool](arg, "format"),
		}
	}

	if args.Target == "" {
		return errors.New("explorer: target file URI is required")
	}

	contents, ok := l.cache.GetFileContents(args.Target)
	if !ok {
		return fmt.Errorf("could not get file contents for uri %q", args.Target)
	}

	path := workspace.RelativePath(args.Target)

	compileResults := explorer.CompilerStages(
		path,
		contents,
		args.Strict,
		args.Annotations,
		args.Print,
	)

	// For VSCode, use the notification approach
	if l.Workspace().Client().Identifier == clients.IdentifierVSCode {
		stages := make([]types.ExplorerStageResult, 0, len(compileResults))
		hasErrors := false

		for _, cs := range compileResults {
			stage := types.ExplorerStageResult{
				Name:  string(cs.Stage),
				Error: cs.Error != "",
			}

			if cs.Error != "" {
				hasErrors = true
				stage.Output = cs.Error
			} else {
				if args.Format {
					stage.Output = cs.FormattedResult()
				} else if cs.Result != nil {
					stage.Output = cs.Result.String()
				}
			}

			stages = append(stages, stage)
		}

		responseParams := types.ExplorerResult{Stages: stages}

		if !hasErrors {
			if plan, err := explorer.Plan(ctx, path, contents, args.Print); err == nil {
				responseParams.Plan = plan
			}
		}

		if err := l.conn.Notify(ctx, "regal/showExplorerResult", responseParams); err != nil {
			return fmt.Errorf("regal/showExplorerResult notification failed: %w", err)
		}

		return nil
	}

	// For other LSP clients, write stages to temp files and use window/showDocument
	tmpDir, err := os.MkdirTemp("", "regal-explorer-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	hasErrors := false
	baseName := filepath.Base(uri.ToPath(args.Target))
	baseName = strings.TrimSuffix(baseName, ".rego")

	var previousOutput string

	filesToOpen := make([]string, 0)

	for i, cs := range compileResults {
		var output string

		if cs.Error != "" {
			hasErrors = true
			output = cs.Error
		} else if cs.Result != nil {
			if args.Format {
				output = cs.FormattedResult()
			} else {
				output = cs.Result.String()
			}
		}

		if output == "" {
			continue
		}

		stageName := strings.ReplaceAll(string(cs.Stage), " ", "_")
		filename := filepath.Join(tmpDir, fmt.Sprintf("%02d_%s_%s.txt", i, baseName, stageName))

		if err := os.WriteFile(filename, []byte(output), 0o600); err != nil {
			l.log.Message("failed to write stage file %s: %s", filename, err)

			continue
		}

		// Only open stages where output differs from previous stage
		if output != previousOutput {
			filesToOpen = append(filesToOpen, filename)
			previousOutput = output
		}
	}

	for _, filename := range filesToOpen {
		// Use a timeout context for RPC to ensure it completes during graceful shutdown
		rpcCtx, rpcCancel := context.WithTimeout(context.Background(), rpcTimeout)

		//nolint:contextcheck
		l.window.ShowDocument(rpcCtx, workspace.URI(filename), false)
		rpcCancel()
	}

	if !hasErrors {
		if plan, err := explorer.Plan(ctx, path, contents, args.Print); err == nil && plan != "" {
			planFile := filepath.Join(tmpDir, fmt.Sprintf("%02d_%s_Plan.txt", len(compileResults), baseName))
			if err := os.WriteFile(planFile, []byte(plan), 0o600); err != nil {
				l.log.Message("failed to write plan file: %s", err)
			} else {
				// Use a timeout context for RPC to ensure it completes during graceful shutdown
				rpcCtx, rpcCancel := context.WithTimeout(context.Background(), rpcTimeout)

				//nolint:contextcheck
				l.window.ShowDocument(rpcCtx, workspace.URI(planFile), false)
				rpcCancel()
			}
		}
	}

	return nil
}

func (l *LanguageServer) handleWorkspaceExecuteCommand(params types.ExecuteCommandParams) (any, error) {
	// this must not block, so we send the request to the worker on a buffered channel.
	// the response to the workspace/executeCommand request must be sent before the command is executed
	// so that the client can complete the request and be ready to receive the follow-on request for
	// workspace/applyEdit.
	l.commandRequest <- params

	// however, the contents of the response is not important
	return emptyStruct, nil
}

func (l *LanguageServer) handleCreateTestCommand(ctx context.Context, params types.ExecuteCommandParams) error {
	var args types.CommandArgs

	if len(params.Arguments) > 0 {
		arg, ok := params.Arguments[0].(map[string]any)
		if !ok {
			l.log.Message(
				"failed to unmarshal regal.createTest command arguments, expected object, got %T",
				params.Arguments[0],
			)

			return errors.New("failed to parse createTest arguments")
		}

		args = types.CommandArgs{
			Target: util.MapGet[string](arg, "target"),
		}
	}

	if args.Target == "" {
		l.log.Message("expected command target, got empty string")

		return errors.New("target file URI is required")
	}

	inputPath := l.input.FindForPath(args.Target)
	inputValue := l.input.Get(ctx, inputPath)

	inputEmpty := inputValue == nil
	if obj, ok := inputValue.(ast.Object); ok {
		inputEmpty = obj.Len() == 0
	}

	if inputPath == "" || inputEmpty {
		l.window.ShowMessage(
			ctx,
			types.InfoMessage,
			"No input.json or input.yaml file found. Create one to provide test input data.",
		)

		return nil
	}

	_, module, ok := l.cache.GetContentAndModule(args.Target)
	if !ok {
		return fmt.Errorf("could not get file contents for uri %q", args.Target)
	}

	combinedTest, err := testgen.CreateTestModule(ctx, testgen.TestModuleOptions{
		Module:        module,
		AllModules:    l.cache.GetAllModules(),
		WorkspacePath: l.Workspace().Path(),
		FileURI:       args.Target,
		InputManager:  l.input,
	})
	if err != nil {
		l.log.Message("errors creating tests: %v", err)
	}

	if combinedTest == "" {
		return err
	}

	//nolint:contextcheck
	if err := l.displayTestResult(combinedTest, args.Target); err != nil {
		return fmt.Errorf("failed to display test result: %w", err)
	}

	return nil
}

func (l *LanguageServer) displayTestResult(testCode, sourceURI string) error {
	sourceFile := uri.ToPath(sourceURI)
	baseName := strings.TrimSuffix(filepath.Base(sourceFile), ".rego")
	testFileName := l.Workspace().Path(baseName + "_test.rego")

	if err := os.WriteFile(testFileName, []byte(testCode), 0o600); err != nil {
		return fmt.Errorf("failed to write test file: %w", err)
	}

	// Use a timeout context for RPC to ensure it completes during graceful shutdown
	rpcCtx, rpcCancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer rpcCancel()

	l.window.ShowDocument(rpcCtx, l.Workspace().URI(testFileName), true)

	return nil
}
