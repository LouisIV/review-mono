package repoconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"review/internal/git"
)

// Config holds repo-local review configuration stored in .git/review/config.yaml.
type Config struct {
	DefaultBranch string
}

// Path returns the path to the repo-local config file.
func Path(repo git.Repo) (string, error) {
	gitDir, err := repo.GitDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(gitDir, "review", "config.yaml"), nil
}

// Load reads .git/review/config.yaml for the given repo.
// Returns an empty Config (no error) if the file doesn't exist yet.
func Load(repo git.Repo) (Config, error) {
	path, err := Path(repo)
	if err != nil {
		return Config{}, err
	}

	//nolint:gosec // Path is derived from git dir, not user input.
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}

	if err != nil {
		return Config{}, err
	}

	return parse(string(b)), nil
}

// Save writes cfg to .git/review/config.yaml, creating the directory if needed.
func Save(repo git.Repo, cfg Config) error {
	path, err := Path(repo)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(marshal(cfg)), 0o600)
}

func parse(content string) Config {
	cfg := Config{}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		switch strings.TrimSpace(key) {
		case "defaultBranch":
			cfg.DefaultBranch = strings.TrimSpace(val)
		}
	}

	return cfg
}

func marshal(cfg Config) string {
	var sb strings.Builder

	if cfg.DefaultBranch != "" {
		fmt.Fprintf(&sb, "defaultBranch: %s\n", cfg.DefaultBranch)
	}

	return sb.String()
}
