package metrics

import (
	"github.com/open-policy-agent/opa/v1/profiler"

	"github.com/open-policy-agent/regal/pkg/report"
)

const (
	RegalConfigSearch         = "regal_config_search"
	RegalConfigParse          = "regal_config_parse"
	RegalFilterIgnoredFiles   = "regal_filter_ignored_files"
	RegalFilterIgnoredModules = "regal_filter_ignored_modules"
	RegalInputParse           = "regal_input_parse"
	RegalPrepare              = "regal_prepare"
	RegalLint                 = "regal_lint_total"
	RegalLintRego             = "regal_lint_rego"
	RegalLintRegoAggregate    = "regal_lint_rego_aggregate"
	RegalMergeReport          = "regal_assemble_report"
)

func FromExprStats(stats profiler.ExprStats) report.ProfileEntry {
	return report.ProfileEntry{
		Location:    stats.Location.String(),
		TotalTimeNs: stats.ExprTimeNs,
		NumEval:     stats.NumEval,
		NumRedo:     stats.NumRedo,
		NumGenExpr:  stats.NumGenExpr,
	}
}
