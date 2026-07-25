package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mermonia/peridot/internal/logger"
	"github.com/mermonia/peridot/internal/paths"
)

const MaxDebounceDelay = 5 * time.Second
const eventBuffer = 16

type EventKind int

const (
	ModuleChanged EventKind = iota
	GlobalVarsChanged
	StateChanged
)

func (k EventKind) String() string {
	switch k {
	case ModuleChanged:
		return "module-changed"
	case GlobalVarsChanged:
		return "global-vars-changed"
	case StateChanged:
		return "state-changed"
	default:
		return "unknown"
	}
}

type Event struct {
	Kind    EventKind
	Modules []string
}

type watchMode int

const (
	modeFull watchMode = iota
	modeConfig
)

type Watcher struct {
	dotfilesDir string
	debounce    time.Duration

	fsw    *fsnotify.Watcher
	events chan Event

	// mu guards modules and watched, which are read and written by both
	// Run's goroutine and whoever calls SetModules.
	mu sync.Mutex

	modules map[string]watchMode
	watched map[string]struct{}
}

func New(dotfilesDir string, debounce time.Duration) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		dotfilesDir: dotfilesDir,
		debounce:    debounce,
		fsw:         fsw,
		events:      make(chan Event, eventBuffer),
		modules:     map[string]watchMode{},
		watched:     map[string]struct{}{},
	}

	if err := fsw.Add(paths.PeridotDir(dotfilesDir)); err != nil {
		fsw.Close()
		return nil, fmt.Errorf("could not watch the peridot dir: %w", err)
	}

	varsDir := paths.GlobalVarsDir(dotfilesDir)
	if err := fsw.Add(varsDir); err != nil && !os.IsNotExist(err) {
		fsw.Close()
		return nil, fmt.Errorf("could not watch the global variables dir: %w", err)
	}

	return w, nil
}

func (w *Watcher) Events() <-chan Event {
	return w.events
}

func (w *Watcher) Close() error {
	return w.fsw.Close()
}

func (w *Watcher) SetModules(deployed, known []string) error {
	desired := make(map[string]watchMode, len(deployed)+len(known))
	for _, name := range known {
		desired[name] = modeConfig
	}

	// A module that is fully watched does not also need a config watch.
	for _, name := range deployed {
		desired[name] = modeFull
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	for name, mode := range w.modules {
		// A mode change is handled as a removal followed by an add, so that
		// the previous watches are cleaned up.
		if want, keep := desired[name]; !keep || want != mode {
			w.unwatchModuleLocked(name)
		}
	}

	for name, mode := range desired {
		if current, already := w.modules[name]; already && current == mode {
			continue
		}

		moduleDir := paths.ModuleDir(w.dotfilesDir, name)

		if mode == modeFull {
			if err := w.watchDirTreeLocked(moduleDir); err != nil {
				return fmt.Errorf("could not watch module %s: %w", name, err)
			}
		} else if err := w.watchDirLocked(moduleDir); err != nil {
			return fmt.Errorf("could not watch module %s: %w", name, err)
		}

		w.modules[name] = mode
	}

	return nil
}

func (w *Watcher) WatchedModules() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	names := make([]string, 0, len(w.modules))
	for name, mode := range w.modules {
		if mode == modeFull {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (w *Watcher) watchDirLocked(dir string) error {
	if _, already := w.watched[dir]; already {
		return nil
	}

	if err := w.fsw.Add(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("could not watch %s: %w", dir, err)
	}

	w.watched[dir] = struct{}{}
	logger.Debug("Watching directory", "path", dir)
	return nil
}

func (w *Watcher) watchDirTreeLocked(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if !d.IsDir() {
			return nil
		}

		if _, already := w.watched[path]; already {
			return nil
		}

		if err := w.fsw.Add(path); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("could not watch %s: %w", path, err)
		}

		w.watched[path] = struct{}{}
		logger.Debug("Watching directory", "path", path)
		return nil
	})
}

func (w *Watcher) unwatchModuleLocked(name string) {
	moduleDir := paths.ModuleDir(w.dotfilesDir, name)

	for path := range w.watched {
		if path == moduleDir || strings.HasPrefix(path, moduleDir+string(os.PathSeparator)) {
			w.fsw.Remove(path)
			delete(w.watched, path)
		}
	}

	delete(w.modules, name)
	logger.Debug("Stopped watching module", "module", name)
}

func (w *Watcher) Run(ctx context.Context) error {
	defer close(w.events)

	pendingModules := map[string]struct{}{}
	pendingGlobals := false
	pendingState := false

	var timer *time.Timer
	var timerC <-chan time.Time
	var firstPending time.Time

	stopTimer := func() {
		if timer != nil && !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timerC = nil
	}

	schedule := func() {
		now := time.Now()
		if timerC == nil {
			firstPending = now
		}

		delay := w.debounce
		if deadline := firstPending.Add(MaxDebounceDelay); now.Add(delay).After(deadline) {
			delay = max(deadline.Sub(now), 0)
		}

		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(delay)
		}
		timerC = timer.C
	}

	flush := func() []Event {
		stopTimer()

		out := []Event{}

		// A global variables change invalidates the rendered output of every
		// module, so it subsumes any pending per-module change.
		if pendingGlobals {
			out = append(out, Event{Kind: GlobalVarsChanged})
		} else if len(pendingModules) > 0 {
			names := make([]string, 0, len(pendingModules))
			for name := range pendingModules {
				names = append(names, name)
			}
			sort.Strings(names)
			out = append(out, Event{Kind: ModuleChanged, Modules: names})
		}

		if pendingState {
			out = append(out, Event{Kind: StateChanged})
		}

		pendingModules = map[string]struct{}{}
		pendingGlobals = false
		pendingState = false

		return out
	}

	for {
		select {
		case <-ctx.Done():
			stopTimer()
			return nil

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			logger.Warn("Filesystem watch error", "error", err.Error())

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}

			kind, moduleName, relevant := w.classify(ev)
			if !relevant {
				continue
			}

			logger.Debug("Filesystem event", "path", ev.Name, "op", ev.Op.String(),
				"kind", kind.String())

			switch kind {
			case ModuleChanged:
				pendingModules[moduleName] = struct{}{}
			case GlobalVarsChanged:
				pendingGlobals = true
			case StateChanged:
				pendingState = true
			}

			schedule()

		case <-timerC:
			for _, out := range flush() {
				select {
				case w.events <- out:
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}

func (w *Watcher) classify(ev fsnotify.Event) (EventKind, string, bool) {
	if ev.Op == fsnotify.Chmod {
		return 0, "", false
	}

	path := filepath.Clean(ev.Name)
	varsDir := paths.GlobalVarsDir(w.dotfilesDir)
	peridotDir := paths.PeridotDir(w.dotfilesDir)

	if path == varsDir {
		if ev.Has(fsnotify.Create) {
			w.mu.Lock()
			if err := w.fsw.Add(varsDir); err != nil {
				logger.Warn("Could not watch the global variables dir",
					"error", err.Error())
			}
			w.mu.Unlock()
		}
		return GlobalVarsChanged, "", true
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)

	if dir == varsDir {
		if isIgnoredFile(base) {
			return 0, "", false
		}
		return GlobalVarsChanged, "", true
	}

	if dir == peridotDir {
		// Everything else under .peridot is peridot's own output: rendered
		// intermediates, the log file, and the temporary file that SaveState
		// renames into place. Reacting to those would be a feedback loop.
		if base == paths.StateFileName {
			return StateChanged, "", true
		}
		return 0, "", false
	}

	rel, err := filepath.Rel(w.dotfilesDir, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return 0, "", false
	}

	moduleName := paths.SplitPath(rel)[0]

	w.mu.Lock()
	mode, watched := w.modules[moduleName]
	w.mu.Unlock()

	if !watched {
		return 0, "", false
	}

	if mode == modeConfig {
		if base != paths.ModuleConfigFileName {
			return 0, "", false
		}
		return StateChanged, "", true
	}

	if isIgnoredFile(base) {
		return 0, "", false
	}

	if ev.Has(fsnotify.Create) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			w.mu.Lock()
			if err := w.watchDirTreeLocked(path); err != nil {
				logger.Warn("Could not watch new directory", "path", path,
					"error", err.Error())
			}
			w.mu.Unlock()
		}
	}

	if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
		w.mu.Lock()
		delete(w.watched, path)
		w.mu.Unlock()
	}

	return ModuleChanged, moduleName, true
}

func isIgnoredFile(base string) bool {
	switch {
	case base == "4913": // vim's writability probe
		return true
	case strings.HasSuffix(base, "~"):
		return true
	case strings.HasSuffix(base, ".tmp"):
		return true
	case strings.HasSuffix(base, ".swp"), strings.HasSuffix(base, ".swx"),
		strings.HasSuffix(base, ".swo"):
		return true
	case strings.HasPrefix(base, ".#"): // emacs lock files
		return true
	case strings.HasPrefix(base, "#") && strings.HasSuffix(base, "#"):
		return true
	}

	return false
}
