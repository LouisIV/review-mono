package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type entry struct {
	client  *Client
	opened  map[string]bool
}

// Manager owns one LSP client per server type and multiplexes hover requests
// across files.  It is safe for concurrent use.
type Manager struct {
	rootPath string
	clients  map[string]*entry
	mu       sync.Mutex
}

// NewManager creates a Manager rooted at rootPath (the git repository root).
func NewManager(rootPath string) *Manager {
	return &Manager{
		rootPath: rootPath,
		clients:  make(map[string]*entry),
	}
}

// Hover returns hover information for the given file at the 1-indexed line.
// Returns an empty string (no error) when no server is available or the server
// has no info for that location.
func (m *Manager) Hover(repoPath, filename string, line int) (string, error) {
	cmd := ServerCommand(filename)
	if cmd == nil {
		return "", nil
	}

	key := cmd[0]
	m.mu.Lock()
	e, ok := m.clients[key]

	if !ok {
		client, err := Start(cmd)
		if err != nil {
			m.mu.Unlock()

			return "", fmt.Errorf("lsp: start server: %w", err)
		}

		if err := client.Initialize(m.rootPath); err != nil {
			client.Shutdown()
			m.mu.Unlock()

			return "", fmt.Errorf("lsp: initialize: %w", err)
		}

		e = &entry{client: client, opened: make(map[string]bool)}
		m.clients[key] = e
	}
	m.mu.Unlock()

	fullPath := filepath.Join(repoPath, filename)

	m.mu.Lock()
	alreadyOpen := e.opened[fullPath]
	if !alreadyOpen {
		e.opened[fullPath] = true
	}
	m.mu.Unlock()

	if !alreadyOpen {
		text, err := os.ReadFile(fullPath) //nolint:gosec
		if err != nil {
			return "", fmt.Errorf("lsp: read file: %w", err)
		}

		langID := LanguageID(filename)
		if err := e.client.DidOpen(fullPath, string(text), langID); err != nil {
			return "", fmt.Errorf("lsp: didOpen: %w", err)
		}
	}

	// LSP uses 0-indexed lines; diff line numbers are 1-indexed.
	return e.client.Hover(fullPath, line-1, 0)
}

// Close shuts down every managed language server.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.clients {
		e.client.Shutdown()
	}

	m.clients = make(map[string]*entry)
}
