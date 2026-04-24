package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"review/internal/git"
	"review/internal/models"
)

type Store struct {
	Root string
}

func Open(repo git.Repo) (Store, error) {
	gitDir, err := repo.GitDir()
	if err != nil {
		return Store{}, err
	}
	root := filepath.Join(gitDir, "reviews")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Store{}, err
	}
	return Store{Root: root}, nil
}

func (s Store) ListSessions() ([]models.Session, error) {
	var sessions []models.Session
	err := readJSON(filepath.Join(s.Root, "sessions.json"), &sessions)
	if errors.Is(err, os.ErrNotExist) {
		return sessions, nil
	}
	return sessions, err
}

func (s Store) SaveSessions(sessions []models.Session) error {
	return writeJSON(filepath.Join(s.Root, "sessions.json"), sessions)
}

func (s Store) UpsertSession(session models.Session) error {
	sessions, err := s.ListSessions()
	if err != nil {
		return err
	}
	replaced := false
	for i := range sessions {
		if sessions[i].ID == session.ID {
			sessions[i] = session
			replaced = true
			break
		}
	}
	if !replaced {
		sessions = append(sessions, session)
	}
	if err := os.MkdirAll(filepath.Join(s.Root, "sessions", session.ID), 0o755); err != nil {
		return err
	}
	return s.SaveSessions(sessions)
}

func (s Store) DeleteSession(id string) error {
	sessions, err := s.ListSessions()
	if err != nil {
		return err
	}
	filtered := sessions[:0]
	for _, session := range sessions {
		if session.ID != id {
			filtered = append(filtered, session)
		}
	}
	if err := s.SaveSessions(filtered); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.Root, "sessions", id))
}

func (s Store) Session(id string) (models.Session, error) {
	sessions, err := s.ListSessions()
	if err != nil {
		return models.Session{}, err
	}
	for _, session := range sessions {
		if session.ID == id {
			return session, nil
		}
	}
	return models.Session{}, os.ErrNotExist
}

func (s Store) CurrentSession(repoPath, branch string) (models.Session, error) {
	sessions, err := s.ListSessions()
	if err != nil {
		return models.Session{}, err
	}
	for i := len(sessions) - 1; i >= 0; i-- {
		session := sessions[i]
		if session.Repo == repoPath && session.Branch == branch && session.Status != models.StatusApproved {
			return session, nil
		}
	}
	return models.Session{}, os.ErrNotExist
}

func (s Store) Comments(sessionID string) ([]models.Comment, error) {
	var comments []models.Comment
	err := readJSON(filepath.Join(s.Root, "sessions", sessionID, "comments.json"), &comments)
	if errors.Is(err, os.ErrNotExist) {
		return comments, nil
	}
	return comments, err
}

func (s Store) SaveComments(sessionID string, comments []models.Comment) error {
	if err := os.MkdirAll(filepath.Join(s.Root, "sessions", sessionID), 0o755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(s.Root, "sessions", sessionID, "comments.json"), comments)
}

func (s Store) Description(sessionID string) (models.Description, error) {
	path := filepath.Join(s.Root, "sessions", sessionID, "description.md")
	body, err := os.ReadFile(path)
	if err != nil {
		return models.Description{}, err
	}
	info, _ := os.Stat(path)
	generatedAt := time.Now().UTC()
	if info != nil {
		generatedAt = info.ModTime().UTC()
	}
	return models.Description{Body: string(body), GeneratedAt: generatedAt}, nil
}

func (s Store) SaveDescription(sessionID, body string) (models.Description, error) {
	if err := os.MkdirAll(filepath.Join(s.Root, "sessions", sessionID), 0o755); err != nil {
		return models.Description{}, err
	}
	path := filepath.Join(s.Root, "sessions", sessionID, "description.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return models.Description{}, err
	}
	return models.Description{Body: body, GeneratedAt: time.Now().UTC()}, nil
}

func readJSON(path string, target any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
