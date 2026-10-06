package types

type TestTarget struct {
	URI     string `json:"uri,omitempty"`
	Package string `json:"package,omitempty"`
	Name    string `json:"name,omitempty"`
}

// RunTestsParams represents the parameters for the regal/runTests LSP request.
type RunTestsParams struct {
	Targets []TestTarget `json:"targets,omitempty"`
	Exclude []TestTarget `json:"exclude,omitempty"`
}
