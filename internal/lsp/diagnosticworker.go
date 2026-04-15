package lsp

import (
	"context"
	"time"
)

// lintFileJob is sent to the lintFileJobs channel to trigger a
// diagnostic update for a file.
type lintFileJob struct {
	Reason string
	URI    string
}

// lintWorkspaceJob is sent to lintWorkspaceJobs when a full workspace
// diagnostic update is needed.
type lintWorkspaceJob struct {
	Reason string
	// IsInitialization is only set once, for the workspace lint that is
	// run on all files loaded  from disk at startup. This closes the
	// init gate on the server.
	IsInitialization bool
	// FullWorkspaceLint runs all rules (aggregate + non-aggregate) with input
	// modules, like initialization, but without storing aggregates or closing
	// the initialization gate. Used for config changes and at init.
	FullWorkspaceLint bool
}

// startFileLintWorker processes individual file linting jobs.
// It listens on l.lintFileJobs, parses files, sends test locations,
// runs file diagnostics, and queues workspace jobs for aggregate updates.
func startFileLintWorker(ctx context.Context, l *LanguageServer) {
	// wait for initial workspace lint to run before doing incremental lints.
	<-l.initializationGate

	for {
		select {
		case <-ctx.Done():
			return
		case job := <-l.lintFileJobs:
			l.log.Debug("linting file %s (%s)", job.URI, job.Reason)

			// updateParse will not return an error when the parsing failed,
			// but only when it was impossible to parse the file.
			parseSuccess, err := updateParse(ctx, l.parseOpts(job.URI, l.builtinsForCurrentCapabilities()))
			if err != nil {
				l.log.Message("failed to update module for %s: %s", job.URI, err)

				continue
			}

			// Send test locations update after parse completes (if client supports it)
			if l.getClient().SupportsOPATestProvider() {
				if parseSuccess {
					l.testLocationJobs <- lintFileJob{Reason: job.Reason, URI: job.URI}
				} else {
					// Parse failed, send empty test locations
					if err := l.sendTestLocations(ctx, job.URI, []any{}); err != nil {
						l.log.Message("failed to send empty test locations after parse failure: %s", err)
					}
				}
			}

			// lint the file and send the diagnostics
			if err := updateFileDiagnostics(ctx, diagnosticsRunOpts{
				Cache:            l.cache,
				Store:            l.regoStore,
				RegalConfig:      l.getLoadedConfig(),
				FileURI:          job.URI,
				WorkspaceRootURI: l.getWorkspaceRootURI(),
				// updateFileDiagnostics only ever updates the diagnostics
				// of non aggregate rules
				UpdateForRules:  l.getEnabledNonAggregateRules(),
				CustomRulesPath: l.getCustomRulesPath(),
			}); err != nil {
				l.log.Message("failed to update file diagnostics: %s", err)

				continue
			}

			l.sendFileDiagnostics(ctx, job.URI)

			l.lintWorkspaceJobs <- lintWorkspaceJob{
				Reason: "file " + job.URI + " " + job.Reason,
			}

			l.log.Debug("linting file %s done", job.URI)
		}
	}
}

// startWorkspaceLintJobRouter routes workspace linting jobs with rate limiting.
// It listens on l.lintWorkspaceJobs and forwards to workspaceLintRuns,
// implementing backpressure for aggregate-only reports to prevent performance degradation.
func startWorkspaceLintJobRouter(ctx context.Context, l *LanguageServer, workspaceLintRuns chan<- lintWorkspaceJob) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-l.lintWorkspaceJobs:
			if !job.IsInitialization && !job.FullWorkspaceLint &&
				len(workspaceLintRuns) > workspaceLintRunsRateLimitThreshold {
				l.log.Debug("dropped surplus workspace lint run")

				continue
			}

			workspaceLintRuns <- job
		}
	}
}

// startPollingTicker sends periodic workspace diagnostic jobs.
// Only runs if l.workspaceDiagnosticsPoll > 0.
func startPollingTicker(ctx context.Context, l *LanguageServer, workspaceLintRuns chan<- lintWorkspaceJob) {
	ticker := time.NewTicker(l.workspaceDiagnosticsPoll)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			workspaceLintRuns <- lintWorkspaceJob{Reason: "poll ticker"}
		}
	}
}

// startWorkspaceLintWorker processes workspace-level linting jobs.
// It listens on workspaceLintRuns, runs aggregate and/or non-aggregate rules,
// and sends diagnostics for all files in the workspace.
func startWorkspaceLintWorker(ctx context.Context, l *LanguageServer, workspaceLintRuns <-chan lintWorkspaceJob) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-workspaceLintRuns:
			l.log.Debug("linting workspace: %#v", job)

			targetRules := l.getEnabledAggregateRules()
			if job.FullWorkspaceLint {
				targetRules = append(targetRules, l.getEnabledNonAggregateRules()...)
			}

			err := updateWorkspaceDiagnostics(ctx, diagnosticsRunOpts{
				Cache:             l.cache,
				RegalConfig:       l.getLoadedConfig(),
				Store:             l.regoStore,
				WorkspaceRootURI:  l.getWorkspaceRootURI(),
				UpdateForRules:    targetRules,
				CustomRulesPath:   l.getCustomRulesPath(),
				IsInitialization:  job.IsInitialization,
				FullWorkspaceLint: job.FullWorkspaceLint,
			})
			if err != nil {
				l.log.Message("failed to update all diagnostics: %s", err)
			}

			for fileURI := range l.cache.GetAllFiles() {
				l.sendFileDiagnostics(ctx, fileURI)
			}

			if job.IsInitialization {
				l.initializationGateOnce.Do(func() { close(l.initializationGate) })

				l.log.Debug("closed initialization gate")
			}

			l.log.Debug("linting workspace done")
		}
	}
}
