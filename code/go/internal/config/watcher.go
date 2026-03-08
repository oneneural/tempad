package config

import (
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a WORKFLOW.md file for changes and sends reloaded configs
// on a channel. It watches the directory (not the file) to handle
// rename-and-replace edits (vim, emacs, etc.).
type Watcher struct {
	path      string // absolute path to workflow file
	cli       *CLIFlags
	reloadCh  chan *ServiceConfig
	logger    *slog.Logger
	fsWatcher *fsnotify.Watcher
	stopOnce  sync.Once
	done      chan struct{}
}

// NewWatcher creates a file watcher for the given workflow path.
// The reload channel receives new configs when the file changes.
func NewWatcher(workflowPath string, cli *CLIFlags, logger *slog.Logger) (*Watcher, error) {
	absPath, err := filepath.Abs(workflowPath)
	if err != nil {
		return nil, err
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the directory, not the file, to catch rename-and-replace.
	dir := filepath.Dir(absPath)
	if err := fsw.Add(dir); err != nil {
		fsw.Close()
		return nil, err
	}

	return &Watcher{
		path:      absPath,
		cli:       cli,
		reloadCh:  make(chan *ServiceConfig, 1),
		logger:    logger,
		fsWatcher: fsw,
		done:      make(chan struct{}),
	}, nil
}

// ReloadCh returns the channel that receives reloaded configs.
func (w *Watcher) ReloadCh() <-chan *ServiceConfig {
	return w.reloadCh
}

// Start begins watching for file changes. Call Stop to clean up.
func (w *Watcher) Start() {
	go w.watch()
}

// Stop closes the watcher and releases resources.
func (w *Watcher) Stop() {
	w.stopOnce.Do(func() {
		w.fsWatcher.Close()
		<-w.done
	})
}

func (w *Watcher) watch() {
	defer close(w.done)

	var debounceTimer *time.Timer
	fileName := filepath.Base(w.path)

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Only react to events for our specific file.
			if filepath.Base(event.Name) != fileName {
				continue
			}

			// Filter to write/create/rename events.
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Rename) {
				continue
			}

			// Debounce: reset timer on each event, fire after 500ms of quiet.
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(500*time.Millisecond, func() {
				w.reload()
			})

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			w.logger.Warn("file watcher error", "error", err)
		}
	}
}

func (w *Watcher) reload() {
	cfg, _, err := Load(w.cli)
	if err != nil {
		w.logger.Warn("config reload failed, keeping current config", "error", err)
		return
	}

	// Basic validation — consumer does mode-specific validation.
	if cfg.TrackerKind == "" || cfg.TrackerAPIKey == "" {
		w.logger.Warn("reloaded config missing required fields, keeping current config")
		return
	}

	w.logger.Info("config reloaded from workflow file")

	// Non-blocking send — drop if consumer hasn't read the last one.
	select {
	case w.reloadCh <- cfg:
	default:
	}
}
