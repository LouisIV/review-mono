package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	hoverTimeout = 10 * time.Second
	msgBufSize   = 64
)

// Client manages a single language server process via JSON-RPC over stdio.
type Client struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	msgs  chan json.RawMessage
	done  chan struct{}
	mu    sync.Mutex
	seq   atomic.Int32
}

// Start spawns the language server identified by args and begins reading its
// stdout in a background goroutine.  The caller must call Shutdown when done.
func Start(args []string) (*Client, error) {
	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...) //nolint:gosec
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp: stdout pipe: %w", err)
	}
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("lsp: start %q: %w", args[0], err)
	}

	c := &Client{
		cmd:   cmd,
		stdin: stdin,
		msgs:  make(chan json.RawMessage, msgBufSize),
		done:  make(chan struct{}),
	}
	go c.readLoop(bufio.NewReader(stdout))

	return c, nil
}

// Initialize sends the LSP initialize/initialized handshake for rootPath.
func (c *Client) Initialize(rootPath string) error {
	rootURI := "file://" + filepath.ToSlash(rootPath)
	id := c.nextID()
	req := RequestMessage{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "initialize",
		Params: InitializeParams{
			RootURI: rootURI,
			Capabilities: ClientCapabilities{
				TextDocument: TextDocumentClientCapabilities{
					Hover: HoverClientCapabilities{
						ContentFormat: []string{"plaintext", "markdown"},
					},
				},
			},
		},
	}

	c.mu.Lock()
	err := c.send(req)
	c.mu.Unlock()

	if err != nil {
		return err
	}

	if _, err := c.readResponseFor(id); err != nil {
		return fmt.Errorf("lsp: initialize: %w", err)
	}

	c.mu.Lock()
	err = c.send(NotificationMessage{JSONRPC: "2.0", Method: "initialized", Params: map[string]any{}})
	c.mu.Unlock()

	return err
}

// DidOpen sends a textDocument/didOpen notification for the given file.
func (c *Client) DidOpen(path, text, languageID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.send(NotificationMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params: DidOpenTextDocumentParams{
			TextDocument: TextDocumentItem{
				URI:        "file://" + filepath.ToSlash(path),
				LanguageID: languageID,
				Version:    1,
				Text:       text,
			},
		},
	})
}

// Hover requests hover info at the given zero-based line/character position.
// Returns an empty string when the server has no info for that location.
func (c *Client) Hover(path string, line, char int) (string, error) {
	id := c.nextID()
	req := RequestMessage{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "textDocument/hover",
		Params: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: "file://" + filepath.ToSlash(path)},
			Position:     Position{Line: line, Character: char},
		},
	}

	c.mu.Lock()
	err := c.send(req)
	c.mu.Unlock()

	if err != nil {
		return "", err
	}

	resp, err := c.readResponseFor(id)
	if err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("lsp: %s", resp.Error.Message)
	}

	return parseHover(resp.Result)
}

// Shutdown sends shutdown + exit and waits for the server process to end.
func (c *Client) Shutdown() {
	id := c.nextID()
	req := RequestMessage{JSONRPC: "2.0", ID: id, Method: "shutdown", Params: nil}

	c.mu.Lock()
	_ = c.send(req)
	c.mu.Unlock()

	_, _ = c.readResponseFor(id)

	c.mu.Lock()
	_ = c.send(NotificationMessage{JSONRPC: "2.0", Method: "exit", Params: nil})
	_ = c.stdin.Close()
	c.mu.Unlock()

	close(c.done)
	_ = c.cmd.Wait()
}

// ── internal helpers ──────────────────────────────────────────────────────────

func (c *Client) nextID() int {
	return int(c.seq.Add(1))
}

func (c *Client) send(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	_, err = io.WriteString(c.stdin, header+string(data))

	return err
}

func (c *Client) readLoop(r *bufio.Reader) {
	for {
		raw, err := readLSPMessage(r)
		if err != nil {
			return
		}

		select {
		case c.msgs <- raw:
		case <-c.done:
			return
		}
	}
}

func (c *Client) readResponseFor(id int) (*ResponseMessage, error) {
	timeout := time.After(hoverTimeout)
	for {
		select {
		case raw, ok := <-c.msgs:
			if !ok {
				return nil, errors.New("lsp: server closed")
			}

			var peek struct {
				ID *int `json:"id"`
			}
			_ = json.Unmarshal(raw, &peek)
			if peek.ID == nil || *peek.ID != id {
				continue // notification or unrelated response
			}

			var resp ResponseMessage
			if err := json.Unmarshal(raw, &resp); err != nil {
				return nil, err
			}

			return &resp, nil

		case <-timeout:
			return nil, errors.New("lsp: request timed out")
		}
	}
}

func readLSPMessage(r *bufio.Reader) (json.RawMessage, error) {
	length := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		if value, ok := strings.CutPrefix(line, "Content-Length: "); ok {
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("lsp: bad Content-Length: %w", err)
			}

			length = n
		}
	}

	if length == 0 {
		return nil, errors.New("lsp: missing Content-Length header")
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}

	return body, nil
}
