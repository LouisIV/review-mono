package lsp

import "path/filepath"

// ServerCommand returns the argv needed to start the language server for
// filename, or nil if no supported server is known for that file type.
func ServerCommand(filename string) []string {
	switch filepath.Ext(filename) {
	case ".go":
		return []string{"gopls"}
	case ".py":
		return []string{"pyright-langserver", "--stdio"}
	case ".ts", ".tsx":
		return []string{"typescript-language-server", "--stdio"}
	case ".js", ".jsx", ".mjs":
		return []string{"typescript-language-server", "--stdio"}
	case ".rs":
		return []string{"rust-analyzer"}
	case ".c", ".h":
		return []string{"clangd"}
	case ".cpp", ".cc", ".cxx", ".hpp":
		return []string{"clangd"}
	case ".java":
		return []string{"jdtls"}
	case ".rb":
		return []string{"solargraph", "stdio"}
	case ".lua":
		return []string{"lua-language-server", "--stdio"}
	case ".sh", ".bash":
		return []string{"bash-language-server", "start"}
	default:
		return nil
	}
}

// LanguageID returns the LSP language identifier for filename.
func LanguageID(filename string) string {
	switch filepath.Ext(filename) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js", ".mjs":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".lua":
		return "lua"
	case ".sh", ".bash":
		return "shellscript"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "plaintext"
	}
}
