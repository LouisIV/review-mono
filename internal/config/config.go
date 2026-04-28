package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LSPConfig holds per-user language server preferences.
type LSPConfig struct {
	// Servers maps a file extension (e.g. ".go") to a custom binary path.
	// When set, this path is used instead of auto-discovery for that extension.
	// Example: {".go": "/usr/local/bin/gopls", ".py": "/home/user/.venv/bin/pyright-langserver"}
	Servers map[string]string `json:"servers"`
}

type Config struct {
	AnthropicAPIKey      string    `json:"anthropic_api_key"`
	AIProvider           string    `json:"ai_provider"`
	DefaultBaseBranch    string    `json:"default_base_branch"`
	DaemonPort           int       `json:"daemon_port"`
	OpenBrowserOnApprove bool      `json:"open_browser_on_approve"`
	GitHubHost           string    `json:"github_host"`
	GitLabHost           string    `json:"gitlab_host"`
	LSP                  LSPConfig `json:"lsp"`
}

func Default() Config {
	return Config{
		AIProvider:           "auto",
		DefaultBaseBranch:    "main",
		DaemonPort:           7080,
		OpenBrowserOnApprove: true,
		GitHubHost:           "github.com",
		GitLabHost:           "gitlab.com",
	}
}

func Load() Config {
	cfg := Default()

	path := filepath.Join(ConfigDir(), "config.json")
	//nolint:gosec // Config path is intentionally user-scoped via ConfigDir.
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &cfg)
	}

	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.AnthropicAPIKey = key
	}

	if provider := os.Getenv("REVIEW_AI_PROVIDER"); provider != "" {
		cfg.AIProvider = provider
	}

	return cfg
}

func ConfigDir() string {
	if dir := os.Getenv("REVIEW_CONFIG_DIR"); dir != "" {
		return dir
	}

	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "review")
	}

	return filepath.Join(os.TempDir(), "review")
}
