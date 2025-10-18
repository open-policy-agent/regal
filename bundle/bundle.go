package bundle

import (
	"embed"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/open-policy-agent/opa/v1/bundle"

	rio "github.com/open-policy-agent/regal/internal/io"
	"github.com/open-policy-agent/regal/internal/mode"
)

// Bundle FS will include the tests as well, but since that has negligible impact on the size of the binary,
// it's preferable to filter them out from the bundle than to e.g. create a separate directory for tests.
var (
	Embedded = sync.OnceValue(func() *bundle.Bundle {
		return rio.MustLoadRegalBundleFS(regalBundle)
	})
	Dev = &devMode{}

	//go:embed *
	regalBundle embed.FS

	lastErrMsg     = atomic.Pointer[string]{}
	successLogOnce = sync.OnceFunc(func() {
		fmt.Fprintln(os.Stderr, "Successfully loaded development bundle")
	})
)

type devMode struct {
	bundlePath  string
	bndl        *bundle.Bundle
	mux         sync.RWMutex
	subscribers []chan struct{}
}

// SetPath sets the path of a Regal development bundle, and attempts to load it immediately.
func (dm *devMode) SetPath(path string) {
	dm.mux.Lock()

	if path != dm.bundlePath {
		dm.bundlePath = path
		dm.mux.Unlock()

		dm.Reload()
	} else {
		dm.mux.Unlock()
	}
}

// Bundle returns the currently loaded Regal development bundle, or nil
// when no bundle is loaded.
func (dm *devMode) Bundle() *bundle.Bundle {
	dm.mux.RLock()
	defer dm.mux.RUnlock()

	return dm.bndl
}

func (dm *devMode) Reload() {
	dm.mux.Lock()
	defer dm.mux.Unlock()

	if dm.bundlePath == "" {
		dm.bndl = nil

		return
	}

	if b, err := rio.LoadRegalBundlePath(dm.bundlePath); err != nil {
		// Avoid flooding the console/logs with the same error message
		if curr, last := err.Error(), lastErrMsg.Load(); last == nil || *last != curr {
			fmt.Fprintf(os.Stderr, "error loading development bundle from %s:\n%v\n", dm.bundlePath, err)

			lastErrMsg.Store(&curr)
		}

		return
	} else {
		dm.bndl = b
	}

	if last := lastErrMsg.Load(); last != nil {
		lastErrMsg.Store(nil)

		fmt.Fprintln(os.Stderr, "development bundle back to a good state, no longer using embedded bundle")
	}

	for _, c := range dm.subscribers {
		c <- struct{}{}
	}

	successLogOnce()
}

func (dm *devMode) Subscribe(c chan struct{}) {
	dm.mux.Lock()
	dm.subscribers = append(dm.subscribers, c)
	dm.mux.Unlock()
}

// DevModeEnabled returns true if the Regal binary is compiled without
// the standalone tags added in release builds, and the REGAL_BUNDLE_PATH
// environment variable is set. This is an experimental feature intended
// for Regal LSP development only, and can change at any time without notice.
// Also note that this mode impacts only the Regal bundle used, and not things
// like log levels, etc.
func DevModeEnabled() bool {
	// NOTE: For now, only the env var can be used to enable dev mode.
	return !mode.Standalone && os.Getenv("REGAL_BUNDLE_PATH") != ""
}

// Loaded contains the loaded Regal bundle. For LSP development, allow bundle
// to be loaded dynamically from path instead of the normal one embedded in
// the compiled binary. This allows editing handler  policies while the language
// server is running. This should be considered *very* experimental at this point.
// In the (unlikely) case that you absolutely need to refer to the embedded bundle,
// just call [Embedded] directly instead.
func Loaded() *bundle.Bundle {
	if bndl := Dev.Bundle(); bndl != nil && DevModeEnabled() {
		return bndl
	}

	return Embedded()
}
