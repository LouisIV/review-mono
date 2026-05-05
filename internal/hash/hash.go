// Package hash wraps calls to the hash-cli binary for structural code analysis.
package hash

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// StructureNode represents a top-level declaration in a source file.
type StructureNode struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Skeleton extracts the structural skeleton of a source file.
// Detects the language from the file extension and delegates to the
// appropriate tree-sitter parser via hash-cli.
func Skeleton(path string) ([]StructureNode, error) {
	lang := langFromPath(path)
	if lang == "" {
		return nil, fmt.Errorf("unsupported file type: %s", path)
	}

	out, err := execHashCLI("skeleton", path, "--lang", lang)
	if err != nil {
		return nil, err
	}

	var nodes []StructureNode
	if err := json.Unmarshal([]byte(out), &nodes); err != nil {
		return nil, fmt.Errorf("hash-cli skeleton: %w", err)
	}

	return nodes, nil
}

// CommentLines returns the 0-indexed line numbers that are inside comments.
func CommentLines(path string) ([]int, error) {
	lang := langFromPath(path)
	if lang == "" {
		return nil, nil // silently skip unsupported types
	}

	out, err := execHashCLI("comments", path, "--lang", lang)
	if err != nil {
		return nil, err
	}

	var lines []int
	if err := json.Unmarshal([]byte(out), &lines); err != nil {
		return nil, fmt.Errorf("hash-cli comments: %w", err)
	}

	return lines, nil
}

// AnchorFile reads a file and returns it with hash anchors prepended to
// each line. Uses a stable seed so anchors persist across calls.
func AnchorFile(path string) (string, error) {
	return execHashCLI("anchor", path, "--seed", "42")
}

// StripAnchors removes hash anchors from content.
func StripAnchors(path string) (string, error) {
	return execHashCLI("strip", path)
}

// SkeletonSummary returns a compact text summary of a file's structure
// suitable for inclusion in LLM prompts.
func SkeletonSummary(path string) string {
	nodes, err := Skeleton(path)
	if err != nil || len(nodes) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("File: %s\n", path))
	for _, n := range nodes {
		b.WriteString(fmt.Sprintf("  [%s] %s (lines %d-%d)\n",
			n.Kind, n.Name, n.StartLine+1, n.EndLine+1))
	}
	return b.String()
}

// execHashCLI runs the hash-cli binary with the given arguments.
func execHashCLI(args ...string) (string, error) {
	cmd := exec.Command("hash-cli", args...)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		return "", fmt.Errorf("hash-cli %s: %w%s", strings.Join(args, " "), err, stderr)
	}
	return strings.TrimSpace(string(out)), nil
}

// LangFromPath returns the hash-cli language key for a file path.
func langFromPath(path string) string {
	switch {
	case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".jsx"), strings.HasSuffix(path, ".tsx"):
		return "js"
	case strings.HasSuffix(path, ".py"):
		return "py"
	case strings.HasSuffix(path, ".go"):
		return "go"
	case strings.HasSuffix(path, ".rs"):
		return "rs"
	default:
		return ""
	}
}
