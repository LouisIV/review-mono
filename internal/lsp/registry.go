package lsp

import (
	"path/filepath"
	"strings"
)

// Install describes how to obtain a language server.
type Install struct {
	// Method is one of: "go", "npm", "pip", "cargo", "rustup", "gem", "system".
	Method string
	// Pkgs is the list of packages to pass to the package manager.
	Pkgs []string
	// Note is shown for "system" installs where we cannot automate the step.
	Note string
}

// ExtPath locates a language server binary inside a VSCode-family extension dir.
type ExtPath struct {
	// ExtID is the extension publisher+name prefix, e.g. "rust-lang.rust-analyzer".
	ExtID string
	// BinGlob is a filepath.Glob pattern relative to the versioned extension dir.
	// Use "*" to match the version-stamped sub-directory, e.g. "clangd_*/bin/clangd".
	BinGlob string
}

// ServerDef is the full description of one language server.
type ServerDef struct {
	// Language is the human-readable name shown in `review lsp list`.
	Language string
	// Binary is the executable name looked up on $PATH, e.g. "gopls".
	Binary string
	// LanguageID is the LSP textDocument languageId value, e.g. "go".
	LanguageID string
	// Extensions are the file suffixes this server handles, e.g. [".go"].
	Extensions []string
	// Args are extra arguments appended when launching the server.
	Args []string
	// Install describes how to obtain this server if it is not found.
	Install Install
	// VSCodePaths lists known binary locations inside VSCode-family extension dirs.
	VSCodePaths []ExtPath
}

// InstallCommand returns the argv needed to install this server, or nil when
// the install method is "system" (manual steps required).
func (d ServerDef) InstallCommand() []string {
	switch d.Install.Method {
	case "go":
		return append([]string{"go", "install"}, d.Install.Pkgs...)
	case "npm":
		return append([]string{"npm", "install", "-g"}, d.Install.Pkgs...)
	case "pip":
		return append([]string{"pip", "install"}, d.Install.Pkgs...)
	case "cargo":
		return append([]string{"cargo", "install"}, d.Install.Pkgs...)
	case "rustup":
		return append([]string{"rustup", "component", "add"}, d.Install.Pkgs...)
	case "gem":
		return append([]string{"gem", "install"}, d.Install.Pkgs...)
	default:
		return nil
	}
}

// Registry is the built-in list of supported language servers.
// Entries are ordered by how commonly they are encountered in code reviews.
var Registry = []ServerDef{ //nolint:gochecknoglobals
	{
		Language:   "Go",
		Binary:     "gopls",
		LanguageID: "go",
		Extensions: []string{".go"},
		Install:    Install{Method: "go", Pkgs: []string{"golang.org/x/tools/gopls@latest"}},
		VSCodePaths: []ExtPath{
			{ExtID: "golang.go", BinGlob: "bin/gopls"},
		},
	},
	{
		Language:   "TypeScript / JavaScript",
		Binary:     "typescript-language-server",
		LanguageID: "typescript",
		Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs"},
		Args:       []string{"--stdio"},
		Install:    Install{Method: "npm", Pkgs: []string{"typescript-language-server", "typescript"}},
		// typescript-language-server is not bundled inside any VSCode extension;
		// the built-in tsserver cannot be reused outside the extension host.
	},
	{
		Language:   "Python",
		Binary:     "pyright-langserver",
		LanguageID: "python",
		Extensions: []string{".py"},
		Args:       []string{"--stdio"},
		Install:    Install{Method: "pip", Pkgs: []string{"pyright"}},
		// Pylance (ms-python.vscode-pylance) is closed-source and cannot be
		// reused outside VSCode.  Pyright is the open-source equivalent.
	},
	{
		Language:   "Rust",
		Binary:     "rust-analyzer",
		LanguageID: "rust",
		Extensions: []string{".rs"},
		Install:    Install{Method: "rustup", Pkgs: []string{"rust-analyzer"}},
		VSCodePaths: []ExtPath{
			{ExtID: "rust-lang.rust-analyzer", BinGlob: "server/rust-analyzer"},
		},
	},
	{
		Language:   "C / C++",
		Binary:     "clangd",
		LanguageID: "cpp",
		Extensions: []string{".c", ".h", ".cpp", ".cc", ".cxx", ".hpp"},
		Install: Install{
			Method: "system",
			Note:   "apt install clangd  OR  brew install llvm  OR  https://clangd.llvm.org/installation",
		},
		VSCodePaths: []ExtPath{
			// The clangd extension downloads a versioned binary into a nested dir.
			{ExtID: "llvm-vs-code-extensions.vscode-clangd", BinGlob: "clangd_*/bin/clangd"},
		},
	},
	{
		Language:   "Java",
		Binary:     "jdtls",
		LanguageID: "java",
		Extensions: []string{".java"},
		Install: Install{
			Method: "system",
			Note:   "brew install jdtls  OR  https://github.com/eclipse-jdtls/eclipse.jdt.ls/releases",
		},
	},
	{
		Language:   "Ruby",
		Binary:     "solargraph",
		LanguageID: "ruby",
		Extensions: []string{".rb"},
		Args:       []string{"stdio"},
		Install:    Install{Method: "gem", Pkgs: []string{"solargraph"}},
	},
	{
		Language:   "Lua",
		Binary:     "lua-language-server",
		LanguageID: "lua",
		Extensions: []string{".lua"},
		Args:       []string{"--stdio"},
		Install: Install{
			Method: "system",
			Note:   "brew install lua-language-server  OR  https://github.com/LuaLS/lua-language-server/releases",
		},
		VSCodePaths: []ExtPath{
			{ExtID: "sumneko.lua", BinGlob: "server/bin/lua-language-server"},
		},
	},
	{
		Language:   "Shell",
		Binary:     "bash-language-server",
		LanguageID: "shellscript",
		Extensions: []string{".sh", ".bash"},
		Args:       []string{"start"},
		Install:    Install{Method: "npm", Pkgs: []string{"bash-language-server"}},
	},
}

// DefForFile returns a pointer to the registry entry that handles filename,
// or nil if no supported server covers that file extension.
func DefForFile(filename string) *ServerDef {
	ext := strings.ToLower(filepath.Ext(filename))
	for i := range Registry {
		for _, e := range Registry[i].Extensions {
			if e == ext {
				return &Registry[i]
			}
		}
	}

	return nil
}

// DefForLanguage returns a pointer to the registry entry whose Language field
// contains the given name (case-insensitive prefix match), or nil if not found.
func DefForLanguage(name string) *ServerDef {
	name = strings.ToLower(name)
	for i := range Registry {
		if strings.HasPrefix(strings.ToLower(Registry[i].Language), name) {
			return &Registry[i]
		}

		// Also match by binary name.
		if strings.HasPrefix(strings.ToLower(Registry[i].Binary), name) {
			return &Registry[i]
		}
	}

	return nil
}

// LanguageIDForFile returns the LSP languageId for the given filename.
func LanguageIDForFile(filename string) string {
	if def := DefForFile(filename); def != nil {
		return def.LanguageID
	}

	return "plaintext"
}
