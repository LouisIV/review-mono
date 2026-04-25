package daemon

import (
	"errors"
	"net/http"
	"os"

	"review/internal/ai"
	"review/internal/git"
	"review/internal/models"
	"review/internal/store"
)

func (s *Server) description(
	w http.ResponseWriter,
	r *http.Request,
	repo git.Repo,
	st store.Store,
	session models.Session,
	parts []string,
) {
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
		s.generateDescription(w, r, repo, st, session)

		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

func (s *Server) generateDescription(
	w http.ResponseWriter,
	r *http.Request,
	repo git.Repo,
	st store.Store,
	session models.Session,
) {
	var req struct {
		Prompt   string `json:"prompt"`
		Provider string `json:"provider"`
	}

	_ = readJSON(r, &req)

	raw, err := repo.RawDiff(session.Base)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	provider := s.cfg.AIProvider
	if req.Provider != "" {
		provider = req.Provider
	}

	body, err := ai.GenerateDescription(s.cfg.AnthropicAPIKey, provider, raw, req.Prompt)
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
}
