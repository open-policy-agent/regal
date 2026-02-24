package types

// ServerFeatureFlags defines which custom features are enabled
// in the language server instance. The client has its own set of matching
// flags.
type ServerFeatureFlags struct {
	// ExplorerProvider indicates whether the server supports the regal.explorer
	// command and regal/showExplorerResult notification.
	ExplorerProvider bool `json:"explorer_provider"`
	// InlineEvaluationProvider indicates whether the server supports the regal.eval
	// command response being sent rather than written to file.
	InlineEvaluationProvider bool `json:"inline_evaluation_provider"`
	// DebugProvider indicates whether the server supports the regal.debug
	// command and regal/startDebugging request.
	DebugProvider bool `json:"debug_provider"`
	// OPATestProvider indicates whether the server supports testing-related features
	// including running Rego tests via LSP command and test location notifications.
	OPATestProvider bool `json:"opa_test_provider"`
}
