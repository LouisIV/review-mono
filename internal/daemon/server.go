package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"review/internal/buildinfo"
	"review/internal/config"
	"review/internal/events"
	"review/internal/git"
	"review/internal/ids"
	"review/internal/models"
	"review/internal/store"
	"review/internal/watcher"
)

const queryTrue = "true"

type Server struct {
	cfg config.Config
	bus *events.Bus
	mux *http.ServeMux

	// watcherPool manages file watchers per repository.
	watcherMu   sync.Mutex
	watchers    map[string]*watcher.Watcher
	watcherSubs map[string]int
}

func New(cfg config.Config) *Server {
	_ = buildinfo.Current()

	s := &Server{
		cfg:         cfg,
		bus:         events.NewBus(),
		mux:         http.NewServeMux(),
		watchers:    make(map[string]*watcher.Watcher),
		watcherSubs: make(map[string]int),
	}
	s.routes()

	return s
}

func (s *Server) ListenAndServe(port int) error {
	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", port),
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return server.ListenAndServe()
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("GET /status", s.status)
	s.mux.HandleFunc("GET /events", s.events)
	s.mux.HandleFunc("GET /sessions", s.sessions)
	s.mux.HandleFunc("POST /session", s.openSession)
	s.mux.HandleFunc("POST /approve", s.approve)
	s.mux.HandleFunc("POST /request-changes", s.requestChanges)
	s.mux.HandleFunc("/", s.sessionRoutes)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	info := buildinfo.Current()
	writeJSON(
		w,
		http.StatusOK,
		map[string]any{"ok": true, "version": info.Version, "build_id": info.BuildID, "pid": os.Getpid()},
	)
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "checks": []string{"http"}})
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")

		return
	}

	// Get repo filter from query params
	repoFilter := r.URL.Query().Get("repo")

	// Start watcher if repo is specified
	var cleanup func()
	if repoFilter != "" {
		cleanup = s.startWatcherForRepo(repoFilter)
	}

	id, ch := s.bus.Subscribe()
	defer func() {
		s.bus.Unsubscribe(id)
		if cleanup != nil {
			cleanup()
		}
	}()

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-ch:
			// Filter events by repo if specified
			if repoFilter != "" && event.Repo != repoFilter {
				continue
			}

			b, err := json.Marshal(event)
			if err != nil {
				continue
			}

			if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// startWatcherForRepo ensures a watcher exists for the given repo.
// Returns a cleanup function to call when the SSE connection closes.
func (s *Server) startWatcherForRepo(repo string) func() {
	s.watcherMu.Lock()
	defer s.watcherMu.Unlock()

	if _, ok := s.watchers[repo]; ok {
		s.watcherSubs[repo]++

		return func() { s.removeWatcherSubscription(repo) }
	}

	w := watcher.New(repo, s.bus, "")
	if err := w.Start(); err != nil {
		_ = err

		return func() {}
	}

	s.watchers[repo] = w
	s.watcherSubs[repo] = 1

	return func() { s.removeWatcherSubscription(repo) }
}

// removeWatcherSubscription decrements the subscriber count and stops the watcher if it was the last subscriber.
func (s *Server) removeWatcherSubscription(repo string) {
	s.watcherMu.Lock()
	defer s.watcherMu.Unlock()

	subs := s.watcherSubs[repo]
	if subs > 0 {
		subs--
		s.watcherSubs[repo] = subs
	}

	if subs == 0 {
		if w, ok := s.watchers[repo]; ok {
			w.Stop()
			delete(s.watchers, repo)
			delete(s.watcherSubs, repo)
		}
	}
}

func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	repoPath := r.URL.Query().Get("repo_uri")
	if repoPath == "" {
		writeJSON(w, http.StatusOK, map[string]any{"sessions": []models.Session{}})

		return
	}

	repo, st, err := repoStore(repoPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	_ = repo

	sessions, err := st.ListSessions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	var filtered []models.Session

	for _, session := range sessions {
		if matchSession(session, r) {
			filtered = append(filtered, session)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"sessions": filtered})
}

func (s *Server) openSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Repo string `json:"repo"`
		Base string `json:"base"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	if req.Base == "" {
		req.Base = s.cfg.DefaultBaseBranch
	}

	repo, st, err := repoStore(req.Repo)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	branch, err := repo.Branch()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	now := time.Now().UTC()

	if existing, err := st.CurrentSession(repo.Path, branch); err == nil {
		existing.Base = req.Base
		existing.UpdatedAt = now
		_ = st.UpsertSession(existing)
		writeJSON(w, http.StatusOK, existing)

		return
	}

	session := models.Session{
		ID:        ids.New(),
		Repo:      repo.Path,
		Branch:    branch,
		Base:      req.Base,
		Status:    models.StatusInReview,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := st.UpsertSession(session); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	s.bus.Publish(models.Event{Event: "session_opened", SessionID: session.ID})
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) sessionRoutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "session" {
		writeError(w, http.StatusNotFound, "not found")

		return
	}

	sessionID := parts[1]

	repoPath := r.URL.Query().Get("repo")
	if repoPath == "" && r.Method != http.MethodGet {
		repoPath = r.Header.Get("X-Review-Repo")
	}

	if repoPath == "" {
		repoPath = "."
	}

	repo, st, err := repoStore(repoPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	session, err := st.Session(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")

		return
	}

	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, session)
	case len(parts) == 2 && r.Method == http.MethodDelete:
		err := st.DeleteSession(sessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	case len(parts) == 3 && parts[2] == "commits" && r.Method == http.MethodGet:
		commits, err := repo.Commits(session.Base)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"commits": commits})
	case len(parts) == 3 && parts[2] == "diff" && r.Method == http.MethodGet:
		files, raw, err := repo.Diff(
			session.Base,
			r.URL.Query().Get("file"),
			r.URL.Query().Get("commit"),
			r.URL.Query().Get("skip_hunks") == queryTrue,
		)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"files": files, "raw": raw})
	case len(parts) >= 3 && parts[2] == "comments":
		s.comments(w, r, st, sessionID, parts)
	case len(parts) >= 3 && parts[2] == "description":
		s.description(w, r, repo, st, session, parts)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	session, repo, st, ok := s.sessionFromRequest(w, r)
	if !ok {
		return
	}

	if r.URL.Query().Get("dry_run") != queryTrue {
		err := repo.Push()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}
	}

	desc, _ := st.Description(session.ID)
	now := time.Now().UTC()
	session.Status = models.StatusApproved
	session.ApprovedAt = &now

	session.UpdatedAt = now
	err := st.UpsertSession(session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	mrURL := repo.MRURL(session.Branch, desc.Body)
	s.bus.Publish(models.Event{Event: "approved", SessionID: session.ID})
	writeJSON(
		w,
		http.StatusOK,
		map[string]any{"pushed": r.URL.Query().Get("dry_run") != queryTrue, "branch": session.Branch, "mr_url": mrURL},
	)
}

func (s *Server) requestChanges(w http.ResponseWriter, r *http.Request) {
	session, _, st, ok := s.sessionFromRequest(w, r)
	if !ok {
		return
	}

	var req struct {
		Message string `json:"message"`
	}

	_ = readJSON(r, &req)
	session.Status = models.StatusChangesRequested
	session.Message = req.Message

	session.UpdatedAt = time.Now().UTC()
	err := st.UpsertSession(session)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	s.bus.Publish(models.Event{Event: "changes_requested", SessionID: session.ID, Message: req.Message})
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) sessionFromRequest(
	w http.ResponseWriter,
	r *http.Request,
) (models.Session, git.Repo, store.Store, bool) {
	repoPath := r.URL.Query().Get("repo")

	repo, st, err := repoStore(repoPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return models.Session{}, git.Repo{}, store.Store{}, false
	}

	branch, err := repo.Branch()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return models.Session{}, git.Repo{}, store.Store{}, false
	}

	session, err := st.CurrentSession(repo.Path, branch)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")

		return models.Session{}, git.Repo{}, store.Store{}, false
	}

	return session, repo, st, true
}

func repoStore(path string) (git.Repo, store.Store, error) {
	repo, err := git.Open(path)
	if err != nil {
		return git.Repo{}, store.Store{}, err
	}

	st, err := store.Open(repo)

	return repo, st, err
}

func matchSession(session models.Session, r *http.Request) bool {
	q := r.URL.Query()
	if v := q.Get("branch"); v != "" && session.Branch != v {
		return false
	}

	if v := q.Get("base"); v != "" && session.Base != v {
		return false
	}

	if v := q.Get("status"); v != "" && session.Status != v {
		return false
	}

	return true
}

func filterComments(comments []models.Comment, r *http.Request) []models.Comment {
	q := r.URL.Query()

	var out []models.Comment

	for _, comment := range comments {
		if v := q.Get("file"); v != "" && comment.File != v {
			continue
		}

		if v := q.Get("author"); v != "" && comment.Author != v {
			continue
		}

		if v := q.Get("resolved"); v != "" {
			want, _ := strconv.ParseBool(v)
			if comment.Resolved != want {
				continue
			}
		}

		out = append(out, comment)
	}

	return out
}

func readJSON(r *http.Request, target any) error {
	defer func() {
		_ = r.Body.Close()
	}()

	return json.NewDecoder(r.Body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
