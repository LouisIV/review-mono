package daemon

import (
	"net/http"
	"time"

	"review/internal/ids"
	"review/internal/models"
	"review/internal/store"
)

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
		s.addComment(w, r, st, sessionID, comments)

		return
	}

	if len(parts) == 4 && r.Method == http.MethodPatch {
		s.patchComment(w, r, st, sessionID, parts[3], comments)

		return
	}

	if len(parts) == 4 && r.Method == http.MethodDelete {
		s.deleteComment(w, st, sessionID, parts[3], comments)

		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) addComment(
	w http.ResponseWriter,
	r *http.Request,
	st store.Store,
	sessionID string,
	comments []models.Comment,
) {
	var req struct {
		File      string `json:"file"`
		Line      int    `json:"line"`
		Lines     []int  `json:"lines"`
		Anchor    string `json:"anchor"`
		EndAnchor string `json:"end_anchor"`
		Body      string `json:"body"`
		Author    string `json:"author"`
	}
	err := readJSON(r, &req)
	if err != nil {
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

	comment := models.Comment{
		ID:        ids.New(),
		File:      req.File,
		Line:      req.Line,
		Lines:     req.Lines,
		Anchor:    req.Anchor,
		EndAnchor: req.EndAnchor,
		Body:      req.Body,
		Author:    req.Author,
		CreatedAt: time.Now().UTC(),
	}

	comments = append(comments, comment)
	err = st.SaveComments(sessionID, comments)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	s.bus.Publish(models.Event{Event: "comment_added", SessionID: sessionID, CommentID: comment.ID})
	writeJSON(w, http.StatusCreated, comment)
}

func (s *Server) patchComment(
	w http.ResponseWriter,
	r *http.Request,
	st store.Store,
	sessionID string,
	id string,
	comments []models.Comment,
) {
	var req struct {
		Resolved *bool   `json:"resolved"`
		Body     *string `json:"body"`
	}
	err := readJSON(r, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	for i := range comments {
		if comments[i].ID != id {
			continue
		}

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

		err = st.SaveComments(sessionID, comments)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())

			return
		}

		if req.Resolved != nil && *req.Resolved {
			s.bus.Publish(models.Event{Event: "comment_resolved", SessionID: sessionID, CommentID: id})
		}

		writeJSON(w, http.StatusOK, comments[i])

		return
	}

	writeError(w, http.StatusNotFound, "comment not found")
}

func (s *Server) deleteComment(
	w http.ResponseWriter,
	st store.Store,
	sessionID string,
	id string,
	comments []models.Comment,
) {
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

	err := st.SaveComments(sessionID, next)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	s.bus.Publish(models.Event{Event: "comment_deleted", SessionID: sessionID, CommentID: id})
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}
