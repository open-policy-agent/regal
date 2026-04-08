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
	// OverwriteAggregates for a workspace is only run once at start up. All
	// later updates to aggregate state is made as files are changed.
	OverwriteAggregates bool
	AggregateReportOnly bool
}

// startFileLintWorker processes individual file linting jobs.
// It listens on l.lintFileJobs, parses files, sends test locations,
// runs file diagnostics, and queues workspace jobs for aggregate updates.
func startFileLintWorker(ctx context.Context, l *LanguageServer) {
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
			if l.client.SupportsOPATestProvider() {
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
				// this run is expected to used the cached aggregate state
				// for other files.
				// The aggregate state for this file will still be updated.
				OverwriteAggregates: false,
				// when a file has changed, then there is no need to run
				// any other rules globally other than aggregate rules.
				AggregateReportOnly: true,
			}

			l.log.Debug("linting file %s done", job.URI)
		}
	}
}

// startWorkspaceJobRouter routes workspace linting jobs with rate limiting.
// It listens on l.lintWorkspaceJobs and forwards to workspaceLintRuns,
// implementing backpressure for aggregate-only reports to prevent performance degradation.
func startWorkspaceJobRouter(ctx context.Context, l *LanguageServer, workspaceLintRuns chan<- lintWorkspaceJob) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-l.lintWorkspaceJobs:
			// AggregateReportOnly is set when updating aggregate
			// violations on character changes. Since these happen so
			// frequently, we stop adding to the channel if there already
			// jobs set to preserve performance
			if job.AggregateReportOnly && len(workspaceLintRuns) > 10/2 {
				l.log.Debug("rate limiting aggregate reports")

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
			workspaceLintRuns <- lintWorkspaceJob{Reason: "poll ticker", OverwriteAggregates: true}
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

			// if there are no parsed modules in the cache, then there is
			// no need to run the aggregate report. This can happen if the
			// server is very slow to start up.
			if len(l.cache.GetAllModules()) == 0 {
				continue
			}

			targetRules := l.getEnabledAggregateRules()
			if !job.AggregateReportOnly {
				targetRules = append(targetRules, l.getEnabledNonAggregateRules()...)
			}

			err := updateWorkspaceDiagnostics(ctx, diagnosticsRunOpts{
				Cache:            l.cache,
				RegalConfig:      l.getLoadedConfig(),
				WorkspaceRootURI: l.getWorkspaceRootURI(),
				// this is intended to only be set to true once at start up,
				// on following runs, cached aggregate data is used.
				OverwriteAggregates: job.OverwriteAggregates,
				AggregateReportOnly: job.AggregateReportOnly,
				UpdateForRules:      targetRules,
				CustomRulesPath:     l.getCustomRulesPath(),
			})
			if err != nil {
				l.log.Message("failed to update all diagnostics: %s", err)
			}

			for fileURI := range l.cache.GetAllFiles() {
				l.sendFileDiagnostics(ctx, fileURI)
			}

			l.log.Debug("linting workspace done")
		}
	}
}
