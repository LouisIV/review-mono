package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// FindResult is returned by FindServer and carries both the argv and where
// the binary was found, for display in `review lsp list`.
type FindResult struct {
	// Args is the complete argv used to start the server (binary + extra flags).
	Args []string
	// Source is a short label: "custom", "path", or an editor name
	// ("vscode", "cursor", "windsurf", "vscode-oss", "code-server").
	Source string
}

// FindServer returns the argv needed to launch the language server for
// filename.  It probes in this order:
//
//  1. overrides[ext]  – user-supplied path from config
//  2. $PATH           – binary installed globally
//  3. VSCode-family extension directories
//
// Returns nil when no server is found.
func FindServer(filename string, overrides map[string]string) *FindResult {
	def := DefForFile(filename)
	if def == nil {
		return nil
	}

	ext := filepath.Ext(filename)

	// 1. User config override for this extension.
	if custom, ok := overrides[ext]; ok && custom != "" {
		return &FindResult{Args: append([]string{custom}, def.Args...), Source: "custom"}
	}

	// 2. Binary on $PATH.
	if path, err := exec.LookPath(def.Binary); err == nil {
		return &FindResult{Args: append([]string{path}, def.Args...), Source: "path"}
	}

	// 3. Bundled inside a VSCode-family extension directory.
	if result := probeExtensionDirs(def); result != nil {
		return result
	}

	return nil
}

// probeExtensionDirs searches all known VSCode-family extension roots for a
// binary matching one of def's VSCodePaths glob patterns.
func probeExtensionDirs(def *ServerDef) *FindResult {
	if len(def.VSCodePaths) == 0 {
		return nil
	}

	for _, root := range editorExtRoots() {
		for _, ep := range def.VSCodePaths {
			pattern := filepath.Join(root.path, ep.ExtID+"-*", ep.BinGlob)
			// On Windows the binary has a .exe suffix.
			if runtime.GOOS == "windows" {
				pattern += ".exe"
			}

			matches, _ := filepath.Glob(pattern)
			for _, m := range matches {
				if isExecutable(m) {
					return &FindResult{
						Args:   append([]string{m}, def.Args...),
						Source: root.editor,
					}
				}
			}
		}
	}

	return nil
}

type editorRoot struct {
	path   string
	editor string
}

// editorExtRoots returns all VSCode-family extension directories that exist on
// the current machine, in preference order.
func editorExtRoots() []editorRoot {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	candidates := []editorRoot{
		{filepath.Join(home, ".vscode", "extensions"), "vscode"},
		{filepath.Join(home, ".cursor", "extensions"), "cursor"},
		{filepath.Join(home, ".windsurf", "extensions"), "windsurf"},
		{filepath.Join(home, ".vscode-oss", "extensions"), "vscode-oss"},
	}

	// code-server uses XDG_DATA_HOME when set.
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	candidates = append(candidates, editorRoot{
		path:   filepath.Join(dataHome, "code-server", "extensions"),
		editor: "code-server",
	})

	var found []editorRoot
	for _, c := range candidates {
		if _, err := os.Stat(c.path); err == nil {
			found = append(found, c)
		}
	}

	return found
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		return !info.IsDir()
	}

	return !info.IsDir() && info.Mode()&0o111 != 0
}
