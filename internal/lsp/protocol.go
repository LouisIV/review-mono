package lsp

import "encoding/json"

// RequestMessage is a JSON-RPC 2.0 request.
type RequestMessage struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// NotificationMessage is a JSON-RPC 2.0 notification (no ID, no response).
type NotificationMessage struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

// ResponseMessage is a JSON-RPC 2.0 response.
type ResponseMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError is an error embedded in a JSON-RPC response.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeParams is the params for the LSP initialize request.
type InitializeParams struct {
	RootURI      string             `json:"rootUri"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

// ClientCapabilities advertises what this client supports.
type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument"`
}

// TextDocumentClientCapabilities holds text-document feature flags.
type TextDocumentClientCapabilities struct {
	Hover HoverClientCapabilities `json:"hover"`
}

// HoverClientCapabilities declares accepted hover content formats.
type HoverClientCapabilities struct {
	ContentFormat []string `json:"contentFormat"`
}

// TextDocumentIdentifier identifies a document by URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// Position is a zero-based line/character offset inside a document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// TextDocumentPositionParams pairs a document URI with a cursor position.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// TextDocumentItem is the full representation of an open document.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// DidOpenTextDocumentParams is the params for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// parseHover extracts a plain-text string from a raw hover result.
// The LSP hover result can be null, a MarkupContent, a MarkedString, or an
// array of MarkedStrings; this function handles all variants.
func parseHover(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}

	var result struct {
		Contents json.RawMessage `json:"contents"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", err
	}

	return unmarshalContents(result.Contents)
}

func unmarshalContents(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}

	// MarkupContent: {"kind":"markdown"|"plaintext","value":"..."}
	var mc struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(raw, &mc) == nil && mc.Value != "" {
		return mc.Value, nil
	}

	// Plain string (deprecated MarkedString shorthand).
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, nil
	}

	// Array of MarkedString — take the first element.
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil && len(arr) > 0 {
		return unmarshalContents(arr[0])
	}

	return "", nil
}
