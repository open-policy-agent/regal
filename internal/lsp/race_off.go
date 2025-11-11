//go:build !race

package lsp

func isRaceEnabled() bool {
	return false
}
