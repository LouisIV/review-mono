package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type entry struct {
	client *Client
	opened map[string]bool
}

// Manager owns one LSP client per server binary and multiplexes hover requests
// across files.  It is safe for concurrent use.
type Manager struct {
	rootPath  string
	overrides map[string]string // file-extension → custom binary path from config
	clients   map[string]*entry // keyed by binary name
	mu        sync.Mutex
}

// NewManager creates a Manager rooted at rootPath.
// overrides maps file extensions (e.g. ".go") to custom binary paths;
// pass nil or an empty map to rely solely on auto-discovery.
func NewManager(rootPath string, overrides map[string]string) *Manager {
	if overrides == nil {
		overrides = map[string]string{}
	}

	return &Manager{
		rootPath:  rootPath,
		overrides: overrides,
		clients:   make(map[string]*entry),
	}
}

// Hover returns hover information for the given file at the 1-indexed line.
// Returns an empty string (no error) when no server is available or the server
// has no info for that location.
func (m *Manager) Hover(repoPath, filename string, line int) (string, error) {
	found := FindServer(filename, m.overrides)
	if found == nil {
		return "", nil
	}

	key := found.Args[0]
	m.mu.Lock()
	e, ok := m.clients[key]

	if !ok {
		client, err := Start(found.Args)
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

		if err := e.client.DidOpen(fullPath, string(text), LanguageID(filename)); err != nil {
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
