package cmd

import (
	"fmt"

	"review/internal/config"
	"review/internal/git"
	"review/internal/repoconfig"
)

func configCmd(args []string, g globals, cfg config.Config) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		configUsage()

		return nil
	}

	switch args[0] {
	case "show":
		return configShowCmd(g, cfg)
	case "set":
		return configSetCmd(args[1:], g)
	default:
		return fmt.Errorf("unknown config subcommand %q", args[0])
	}
}

func configSetCmd(args []string, g globals) error {
	if len(args) < 2 || isHelpArg(args[0]) {
		fmt.Println("review config set <key> <value>")
		fmt.Println("keys: defaultBranch")

		return nil
	}

	key, val := args[0], args[1]

	repo, err := git.Open(g.repo)
	if err != nil {
		return err
	}

	repoCfg, err := repoconfig.Load(repo)
	if err != nil {
		return err
	}

	switch key {
	case "defaultBranch":
		repoCfg.DefaultBranch = val
	default:
		return fmt.Errorf("unknown config key %q", key)
	}

	if err := repoconfig.Save(repo, repoCfg); err != nil {
		return err
	}

	path, _ := repoconfig.Path(repo)

	if g.json {
		printJSON(map[string]any{"key": key, "value": val, "path": path})
	} else {
		fmt.Printf("Set %s = %s in %s\n", key, val, path)
	}

	return nil
}

func configShowCmd(g globals, cfg config.Config) error {
	repo, err := git.Open(g.repo)
	if err != nil {
		return err
	}

	repoCfg, err := repoconfig.Load(repo)
	if err != nil {
		return err
	}

	path, _ := repoconfig.Path(repo)

	// Effective value: repo config overrides user config.
	effectiveBranch := cfg.DefaultBaseBranch
	if repoCfg.DefaultBranch != "" {
		effectiveBranch = repoCfg.DefaultBranch
	}

	if g.json {
		printJSON(map[string]any{
			"repo_config": map[string]any{
				"path":          path,
				"defaultBranch": repoCfg.DefaultBranch,
			},
			"user_config": map[string]any{
				"defaultBaseBranch": cfg.DefaultBaseBranch,
			},
			"effective": map[string]any{
				"defaultBranch": effectiveBranch,
			},
		})

		return nil
	}

	fmt.Printf("Repo config (%s):\n", path)
	fmt.Printf("  defaultBranch: %s\n", orUnset(repoCfg.DefaultBranch))
	fmt.Println()
	fmt.Printf("User config:\n")
	fmt.Printf("  defaultBaseBranch: %s\n", cfg.DefaultBaseBranch)
	fmt.Println()
	fmt.Printf("Effective:\n")
	fmt.Printf("  defaultBranch: %s\n", effectiveBranch)

	return nil
}

func orUnset(s string) string {
	if s == "" {
		return "(not set)"
	}

	return s
}

func configUsage() {
	fmt.Println("review config <subcommand>")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  show                      display current configuration")
	fmt.Println("  set <key> <value>         set a repo-local config value")
	fmt.Println()
	fmt.Println("Keys:")
	fmt.Println("  defaultBranch             base branch for diffs (e.g. main, develop)")
	fmt.Println()
	fmt.Println("Config is stored in .git/review/config.yaml")
}
