package config

import (
	"context"
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"

	"github.com/open-policy-agent/regal/internal/lsp/log"
)

type Watcher struct {
	log    *log.Logger
	Reload chan string
	Drop   chan struct{}

	pathUpdates chan string

	fsWatcher *fsnotify.Watcher

	path          string
	fsWatcherLock sync.Mutex
}

type WatcherOpts struct {
	Logger *log.Logger
	Path   string
}

func NewWatcher(opts *WatcherOpts) *Watcher {
	w := &Watcher{
		Reload:      make(chan string, 1),
		Drop:        make(chan struct{}, 1),
		pathUpdates: make(chan string, 1),
	}

	if opts != nil {
		w.log = opts.Logger
		w.path = opts.Path
	}

	return w
}

func (w *Watcher) Start(ctx context.Context) error {
	err := w.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop existing watcher: %w", err)
	}

	w.fsWatcherLock.Lock()
	w.fsWatcher, err = fsnotify.NewWatcher()
	w.fsWatcherLock.Unlock()

	if err != nil {
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	go func() {
		w.loop(ctx)
	}()

	return nil
}

func (w *Watcher) Watch(configFilePath string) {
	w.pathUpdates <- configFilePath
}

func (w *Watcher) Stop() error {
	if w.fsWatcher != nil {
		if err := w.fsWatcher.Close(); err != nil {
			return fmt.Errorf("failed to close fsnotify watcher: %w", err)
		}
	}

	return nil
}

func (w *Watcher) loop(ctx context.Context) {
	for {
		select {
		case path := <-w.pathUpdates:
			if w.path != "" {
				err := w.fsWatcher.Remove(w.path)
				if err != nil {
					w.log.Message("failed to remove existing watch: %v\n", err)
				}
			}

			if err := w.fsWatcher.Add(path); err != nil {
				w.log.Debug("failed to add watch: %v\n", err)
			}

			w.path = path

			// when the path itself is changed, then this is an event too
			w.Reload <- path
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				w.log.Message("config watcher event channel closed\n")

				return
			}

			if event.Has(fsnotify.Write) {
				w.Reload <- event.Name
			}

			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				w.path = ""
				w.Drop <- struct{}{}
			}
		case err := <-w.fsWatcher.Errors:
			w.log.Message("config watcher error: %v\n", err)
		case <-ctx.Done():
			if err := w.Stop(); err != nil {
				w.log.Message("failed to stop watcher: %v\n", err)
			}

			return
		}
	}
}
