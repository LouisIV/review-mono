package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"review/internal/models"
)

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(port int) Client {
	return Client{BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port), HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c Client) Health() error {
	var out map[string]any
	return c.do(http.MethodGet, "/health", nil, &out)
}

func (c Client) Open(repo, base string) (models.Session, error) {
	var session models.Session
	err := c.do(http.MethodPost, "/session", map[string]string{"repo": repo, "base": base}, &session)
	return session, err
}

func (c Client) Sessions(repo string) ([]models.Session, error) {
	var out struct {
		Sessions []models.Session `json:"sessions"`
	}
	err := c.do(http.MethodGet, "/sessions?repo_uri="+url.QueryEscape(repo), nil, &out)
	return out.Sessions, err
}

func (c Client) Session(repo string, session models.Session) (models.Session, error) {
	var out models.Session
	err := c.do(http.MethodGet, fmt.Sprintf("/session/%s?repo=%s", session.ID, url.QueryEscape(repo)), nil, &out)
	return out, err
}

func (c Client) CloseSession(repo string, session models.Session) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/session/%s?repo=%s", session.ID, url.QueryEscape(repo)), nil, nil)
}

func (c Client) Commits(repo string, session models.Session) ([]models.Commit, error) {
	var out struct {
		Commits []models.Commit `json:"commits"`
	}
	err := c.do(http.MethodGet, fmt.Sprintf("/session/%s/commits?repo=%s", session.ID, url.QueryEscape(repo)), nil, &out)
	return out.Commits, err
}

func (c Client) Diff(repo string, session models.Session, file, commit string, skipHunks bool) ([]models.DiffFile, string, error) {
	q := url.Values{}
	q.Set("repo", repo)
	if file != "" {
		q.Set("file", file)
	}
	if commit != "" {
		q.Set("commit", commit)
	}
	if skipHunks {
		q.Set("skip_hunks", "true")
	}
	var out struct {
		Files []models.DiffFile `json:"files"`
		Raw   string            `json:"raw"`
	}
	err := c.do(http.MethodGet, fmt.Sprintf("/session/%s/diff?%s", session.ID, q.Encode()), nil, &out)
	return out.Files, out.Raw, err
}

func (c Client) Comments(repo string, session models.Session, filters url.Values) ([]models.Comment, error) {
	if filters == nil {
		filters = url.Values{}
	}
	filters.Set("repo", repo)
	var out struct {
		Comments []models.Comment `json:"comments"`
	}
	err := c.do(http.MethodGet, fmt.Sprintf("/session/%s/comments?%s", session.ID, filters.Encode()), nil, &out)
	return out.Comments, err
}

func (c Client) AddComment(repo string, session models.Session, comment models.Comment) (models.Comment, error) {
	var out models.Comment
	err := c.do(http.MethodPost, fmt.Sprintf("/session/%s/comments?repo=%s", session.ID, url.QueryEscape(repo)), comment, &out)
	return out, err
}

func (c Client) PatchComment(repo string, session models.Session, id string, payload any) (models.Comment, error) {
	var out models.Comment
	err := c.do(http.MethodPatch, fmt.Sprintf("/session/%s/comments/%s?repo=%s", session.ID, id, url.QueryEscape(repo)), payload, &out)
	return out, err
}

func (c Client) DeleteComment(repo string, session models.Session, id string) error {
	return c.do(http.MethodDelete, fmt.Sprintf("/session/%s/comments/%s?repo=%s", session.ID, id, url.QueryEscape(repo)), nil, nil)
}

func (c Client) Description(repo string, session models.Session) (models.Description, error) {
	var out models.Description
	err := c.do(http.MethodGet, fmt.Sprintf("/session/%s/description?repo=%s", session.ID, url.QueryEscape(repo)), nil, &out)
	return out, err
}

func (c Client) SetDescription(repo string, session models.Session, body string) (models.Description, error) {
	var out models.Description
	err := c.do(http.MethodPost, fmt.Sprintf("/session/%s/description?repo=%s", session.ID, url.QueryEscape(repo)), map[string]string{"body": body}, &out)
	return out, err
}

func (c Client) GenerateDescription(repo string, session models.Session, prompt string) (models.Description, error) {
	var out models.Description
	err := c.do(http.MethodPost, fmt.Sprintf("/session/%s/description/generate?repo=%s", session.ID, url.QueryEscape(repo)), map[string]string{"prompt": prompt}, &out)
	return out, err
}

func (c Client) Approve(repo string, dryRun bool) (map[string]any, error) {
	q := url.Values{"repo": []string{repo}}
	if dryRun {
		q.Set("dry_run", "true")
	}
	var out map[string]any
	err := c.do(http.MethodPost, "/approve?"+q.Encode(), map[string]string{}, &out)
	return out, err
}

func (c Client) RequestChanges(repo, message string) (models.Session, error) {
	var out models.Session
	err := c.do(http.MethodPost, "/request-changes?repo="+url.QueryEscape(repo), map[string]string{"message": message}, &out)
	return out, err
}

func (c Client) Watch(onEvent func(models.Event) bool) error {
	resp, err := http.Get(c.BaseURL + "/events")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s", strings.TrimSpace(string(b)))
	}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event models.Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			continue
		}
		if !onEvent(event) {
			return nil
		}
	}
	return sc.Err()
}

func (c Client) do(method, path string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var errObj struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &errObj) == nil && errObj.Error != "" {
			return fmt.Errorf("%s", errObj.Error)
		}
		return fmt.Errorf("%s", strings.TrimSpace(string(data)))
	}
	if target == nil {
		return nil
	}
	return json.Unmarshal(data, target)
}
