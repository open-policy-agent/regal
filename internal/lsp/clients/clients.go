package clients

import "strings"

// Identifier represent different supported clients and can be used to toggle or change
// server behavior based on the client.
type Identifier uint8

const (
	IdentifierGeneric Identifier = iota
	IdentifierVSCode
	IdentifierGoTest
	IdentifierZed
	IdentifierNeovim
	IdentifierIntelliJ
)

// DetermineIdentifier is used to determine the Regal client identifier
// based on the client name.
// Clients with identifiers here should be featured on the 'Editor Support'
// page in the documentation (https://www.openpolicyagent.org/projects/regal/editor-support),
// `docs/editor-support.md`.
func DetermineIdentifier(clientName string) Identifier {
	switch clientName {
	case "go test":
		return IdentifierGoTest
	case "Visual Studio Code":
		return IdentifierVSCode
	case "Zed":
		return IdentifierZed
	case "Neovim":
		// 'Neovim' is sent as the client identifier when using the
		// nvim-lspconfig plugin.
		return IdentifierNeovim
	}

	// check for IntelliJ using contains since version is also set in the name
	if strings.Contains(clientName, "IntelliJ") {
		return IdentifierIntelliJ
	}

	return IdentifierGeneric
}
