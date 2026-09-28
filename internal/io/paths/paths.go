package paths

import (
	"os"
	"path/filepath"
)

const Separator = string(os.PathSeparator)

// WorkingDir carries the working directory as read during the init
// phase of the program. This due to the fact that os.Getwd() can be
// surprisingly expensive (as seen in pprof CPU profiling) when called
// frequently (many syscalls with no caching). Needless to say this is
// not be used in any scenario where the working directory may change
// after init. Neither Regal nor OPA does this currently, but definitely
// something to keep in mind for future changes.
var WorkingDir, _ = os.Getwd()

// Abs is like [filepath.Abs], but using the working directory determined
// at init rather than repeated syscalls.
func Abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(WorkingDir, path)
}
