package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"review/internal/ai"
	"review/internal/config"
	"review/internal/events"
	"review/internal/git"
	"review/internal/ids"
	"review/internal/models"
	"review/internal/store"
)

type Server struct {
	cfg config.Config
	bus *events.Bus
	mux *http.ServeMux
}

func New(cfg config.Config) *Server {
	s := &Server{cfg: cfg, bus: events.NewBus(), mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ListenAndServe(port int) error {
	return http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), s.mux)
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

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": "0.1.0"})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
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
	id, ch := s.bus.Subscribe()
	defer s.bus.Unsubscribe(id)
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-ch:
			b, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
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
	session := models.Session{ID: ids.New(), Repo: repo.Path, Branch: branch, Base: req.Base, Status: models.StatusInReview, CreatedAt: now, UpdatedAt: now}
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
		if err := st.DeleteSession(sessionID); err != nil {
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
		files, raw, err := repo.Diff(session.Base, r.URL.Query().Get("file"), r.URL.Query().Get("commit"), r.URL.Query().Get("skip_hunks") == "true")
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

func (s *Server) comments(w http.ResponseWriter, r *http.Request, st store.Store, sessionID string, parts []string) {
	comments, err := st.Comments(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"comments": filterComments(comments, r)})
		return
	}
	if len(parts) == 3 && r.Method == http.MethodPost {
		var req struct {
			File   string `json:"file"`
			Line   int    `json:"line"`
			Lines  []int  `json:"lines"`
			Body   string `json:"body"`
			Author string `json:"author"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.File == "" || req.Body == "" {
			writeError(w, http.StatusBadRequest, "file and body are required")
			return
		}
		if req.Author == "" {
			req.Author = models.AuthorHuman
		}
		comment := models.Comment{ID: ids.New(), File: req.File, Line: req.Line, Lines: req.Lines, Body: req.Body, Author: req.Author, CreatedAt: time.Now().UTC()}
		comments = append(comments, comment)
		if err := st.SaveComments(sessionID, comments); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.bus.Publish(models.Event{Event: "comment_added", SessionID: sessionID, CommentID: comment.ID})
		writeJSON(w, http.StatusCreated, comment)
		return
	}
	if len(parts) == 4 && r.Method == http.MethodPatch {
		id := parts[3]
		var req struct {
			Resolved *bool   `json:"resolved"`
			Body     *string `json:"body"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		for i := range comments {
			if comments[i].ID == id {
				if req.Body != nil {
					comments[i].Body = *req.Body
				}
				if req.Resolved != nil {
					comments[i].Resolved = *req.Resolved
					if *req.Resolved {
						now := time.Now().UTC()
						comments[i].ResolvedAt = &now
					} else {
						comments[i].ResolvedAt = nil
					}
				}
				if err := st.SaveComments(sessionID, comments); err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				if req.Resolved != nil && *req.Resolved {
					s.bus.Publish(models.Event{Event: "comment_resolved", SessionID: sessionID, CommentID: id})
				}
				writeJSON(w, http.StatusOK, comments[i])
				return
			}
		}
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}
	if len(parts) == 4 && r.Method == http.MethodDelete {
		id := parts[3]
		next := comments[:0]
		found := false
		for _, comment := range comments {
			if comment.ID == id {
				found = true
				continue
			}
			next = append(next, comment)
		}
		if !found {
			writeError(w, http.StatusNotFound, "comment not found")
			return
		}
		if err := st.SaveComments(sessionID, next); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.bus.Publish(models.Event{Event: "comment_deleted", SessionID: sessionID, CommentID: id})
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		return
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) description(w http.ResponseWriter, r *http.Request, repo git.Repo, st store.Store, session models.Session, parts []string) {
	if len(parts) == 3 && r.Method == http.MethodGet {
		desc, err := st.Description(session.ID)
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "description not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, desc)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodPost {
		var req struct {
			Body string `json:"body"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		desc, err := st.SaveDescription(session.ID, req.Body)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.bus.Publish(models.Event{Event: "description_updated", SessionID: session.ID})
		writeJSON(w, http.StatusOK, desc)
		return
	}
	if len(parts) >= 4 && parts[3] == "generate" && r.Method == http.MethodPost {
		var req struct {
			Prompt string `json:"prompt"`
		}
		_ = readJSON(r, &req)
		raw, err := repo.RawDiff(session.Base)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		body, err := ai.GenerateDescription(s.cfg.AnthropicAPIKey, raw, req.Prompt)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		desc, err := st.SaveDescription(session.ID, body)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		s.bus.Publish(models.Event{Event: "description_updated", SessionID: session.ID})
		writeJSON(w, http.StatusOK, desc)
		return
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	session, repo, st, ok := s.sessionFromRequest(w, r)
	if !ok {
		return
	}
	if r.URL.Query().Get("dry_run") != "true" {
		if err := repo.Push(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	desc, _ := st.Description(session.ID)
	now := time.Now().UTC()
	session.Status = models.StatusApproved
	session.ApprovedAt = &now
	session.UpdatedAt = now
	if err := st.UpsertSession(session); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	mrURL := repo.MRURL(session.Branch, desc.Body)
	s.bus.Publish(models.Event{Event: "approved", SessionID: session.ID})
	writeJSON(w, http.StatusOK, map[string]any{"pushed": r.URL.Query().Get("dry_run") != "true", "branch": session.Branch, "mr_url": mrURL})
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
	if err := st.UpsertSession(session); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.bus.Publish(models.Event{Event: "changes_requested", SessionID: session.ID, Message: req.Message})
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) sessionFromRequest(w http.ResponseWriter, r *http.Request) (models.Session, git.Repo, store.Store, bool) {
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
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}
