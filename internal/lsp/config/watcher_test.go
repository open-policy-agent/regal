package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-policy-agent/regal/internal/lsp/log"
	"github.com/open-policy-agent/regal/internal/test/must"
	"github.com/open-policy-agent/regal/internal/testutil"
)

func TestWatcher(t *testing.T) {
	t.Parallel()

	tempDir := testutil.TempDirectoryOf(t, map[string]string{"config.yaml": "---\nfoo: bar\n"})
	watcher := NewWatcher(&WatcherOpts{Logger: log.NewLogger(log.LevelDebug, t.Output())})
	configFilePath := filepath.Join(tempDir, "config.yaml")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		must.Equal(t, nil, watcher.Start(ctx))
	}()

	watcher.Watch(configFilePath)

	select {
	case <-watcher.Reload:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for initial config event")
	}

	newConfigFileContents := "---\nfoo: baz\n"
	must.WriteFile(t, configFilePath, []byte(newConfigFileContents))

	select {
	case <-watcher.Reload:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for config event")
	}

	must.Equal(t, nil, os.Rename(configFilePath, configFilePath+".new"))

	select {
	case <-watcher.Drop:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for config drop event")
	}
}
