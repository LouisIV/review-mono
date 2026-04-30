package models

import "time"

const (
	StatusInReview         = "in_review"
	StatusChangesRequested = "changes_requested"
	StatusApproved         = "approved"

	AuthorHuman = "human"
	AuthorAgent = "agent"
)

type Session struct {
	ID         string     `json:"id"`
	Repo       string     `json:"repo"`
	Branch     string     `json:"branch"`
	Base       string     `json:"base"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ApprovedAt *time.Time `json:"approved_at"`
	Message    string     `json:"message,omitempty"`
}

type Comment struct {
	ID         string     `json:"id"`
	File       string     `json:"file"`
	Line       int        `json:"line,omitempty"`
	Lines      []int      `json:"lines,omitempty"`
	Body       string     `json:"body"`
	Author     string     `json:"author"`
	Resolved   bool       `json:"resolved"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
}

type Commit struct {
	Hash      string    `json:"hash"`
	HashFull  string    `json:"hash_full"`
	Author    string    `json:"author"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type DiffLine struct {
	Type    string `json:"type"`
	Number  *int   `json:"number"`
	Content string `json:"content"`
}

type DiffHunk struct {
	Header      string     `json:"header"`
	Lines       []DiffLine `json:"lines"`
	Uncommitted bool       `json:"uncommitted,omitempty"`
}

type DiffFile struct {
	Path      string     `json:"path"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	Hunks     []DiffHunk `json:"hunks,omitempty"`
	Comments  []Comment  `json:"comments,omitempty"`
}

type Event struct {
	Event     string    `json:"event"`
	SessionID string    `json:"session_id,omitempty"`
	Repo      string    `json:"repo,omitempty"`
	CommentID string    `json:"comment_id,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type Description struct {
	Body        string    `json:"body"`
	GeneratedAt time.Time `json:"generated_at"`
}
