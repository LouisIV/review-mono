package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	AnthropicAPIKey      string `json:"anthropic_api_key"`
	AIProvider           string `json:"ai_provider"`
	DefaultBaseBranch    string `json:"default_base_branch"`
	DaemonPort           int    `json:"daemon_port"`
	OpenBrowserOnApprove bool   `json:"open_browser_on_approve"`
	GitHubHost           string `json:"github_host"`
	GitLabHost           string `json:"gitlab_host"`
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
