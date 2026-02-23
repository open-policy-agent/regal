package types

// RunTestsParams represents the parameters for the regal/runTests LSP request.
type RunTestsParams struct {
	URI     string `json:"uri"`
	Package string `json:"package"`
	Name    string `json:"name"`
}
