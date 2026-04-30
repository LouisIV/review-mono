package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rjeczalik/notify"

	"review/internal/events"
	"review/internal/models"
)

const (
	// WatchDebounce is the debounce period for file change events.
	WatchDebounce = 200 * time.Millisecond
)

// Watcher handles filesystem watching for a single repository.
type Watcher struct {
	repo     string
	bus      *events.Bus
	notify   chan notify.EventInfo
	stop     chan struct{}
	stopOnce sync.Once
	session  string
}

// New creates a new Watcher for the given repository path.
func New(repo string, bus *events.Bus, session string) *Watcher {
	return &Watcher{
		repo:    repo,
		bus:     bus,
		notify:  make(chan notify.EventInfo, 1024),
		stop:    make(chan struct{}),
		session: session,
	}
}

// Start begins watching the repository for filesystem changes.
func (w *Watcher) Start() error {
	// Watch .git/ for new commits
	gitDir := filepath.Join(w.repo, ".git")
	if err := w.watchDir(gitDir); err != nil {
		return err
	}

	// Watch worktree for changes
	if err := w.watchDir(w.repo); err != nil {
		return err
	}

	go w.run()

	return nil
}

// Stop stops the watcher and cleans up resources.
func (w *Watcher) Stop() {
	w.stopOnce.Do(func() {
		close(w.stop)
	})
}

// watchDir adds a watcher for the given directory.
func (w *Watcher) watchDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	events := notify.Create | notify.Remove | notify.Write | notify.Rename

	return notify.Watch(filepath.Join(dir, "..."), w.notify, events)
}

// run processes filesystem events and publishes them to the bus.
func (w *Watcher) run() {
	defer notify.Stop(w.notify)

	var debounce <-chan time.Time

	for {
		select {
		case <-w.stop:
			return
		case ei := <-w.notify:
			event := w.parseEvent(ei)
			if event == "" {
				continue
			}

			// Debounce rapid events
			if debounce == nil {
				debounce = time.After(WatchDebounce)
			}

			select {
			case <-w.stop:
				return
			case <-debounce:
				debounce = nil
				w.bus.Publish(models.Event{
					Event:     event,
					Repo:      w.repo,
					SessionID: w.session,
				})
			}
		}
	}
}

// parseEvent converts a notify event into an event name.
func (w *Watcher) parseEvent(ei notify.EventInfo) string {
	path := ei.Path()
	rel, err := filepath.Rel(w.repo, path)
	if err != nil {
		return ""
	}

	// Skip .git internal files that are noisy
	if strings.HasPrefix(rel, ".git/objects") || strings.HasPrefix(rel, ".git/index") {
		return ""
	}

	// Detect commit-related changes
	if strings.HasPrefix(rel, ".git/refs") {
		return "commit_created"
	}

	if strings.HasPrefix(rel, ".git/HEAD") {
		return "commit_created"
	}

	// Worktree changes (file edits, additions, deletions)
	if rel != "" && !strings.HasPrefix(rel, ".git/") {
		return "worktree_changed"
	}

	return ""
}
