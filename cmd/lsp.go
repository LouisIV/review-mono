package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"review/internal/config"
	"review/internal/lsp"
)

func lspCmd(args []string, cfg config.Config) error {
	sub := subcommandList
	if len(args) > 0 && !isHelpArg(args[0]) {
		sub = args[0]
		args = args[1:]
	}

	if len(args) > 0 && isHelpArg(args[0]) {
		lspUsage()

		return nil
	}

	switch sub {
	case subcommandList:
		return lspListCmd(cfg)
	case "install":
		return lspInstallCmd(args, cfg)
	default:
		return fmt.Errorf("unknown lsp subcommand %q (try: list, install)", sub)
	}
}

// lspListCmd prints a table of all known servers with their discovery status.
func lspListCmd(cfg config.Config) error {
	const (
		colLang   = -22
		colBinary = -30
		colStatus = -24
	)
	sep := strings.Repeat("─", 80)

	fmt.Printf("%*s  %*s  %*s  %s\n", colLang, "Language", colBinary, "Server", colStatus, "Status", "Install")
	fmt.Println(sep)

	for _, def := range lsp.Registry {
		status, installHint := serverStatus(def, cfg)
		fmt.Printf("%*s  %*s  %*s  %s\n",
			colLang, def.Language,
			colBinary, def.Binary,
			colStatus, status,
			installHint,
		)
	}

	fmt.Println()
	fmt.Println("Configure custom paths in ~/.config/review/config.json:")
	fmt.Println(`  { "lsp": { "servers": { ".go": "/path/to/gopls" } } }`)

	return nil
}

// serverStatus returns a display status string and an install hint for def.
func serverStatus(def lsp.ServerDef, cfg config.Config) (string, string) {
	found := lsp.FindServer(def.Extensions[0], cfg.LSP.Servers)
	if found == nil {
		cmd := def.InstallCommand()
		if cmd != nil {
			return "✗ missing", strings.Join(cmd, " ")
		}

		return "✗ missing", def.Install.Note
	}

	switch found.Source {
	case "custom":
		return "✓ custom", found.Args[0]
	case "path":
		return "✓ installed", ""
	default:
		return "✓ " + found.Source, ""
	}
}

// lspInstallCmd installs one or all language servers.
func lspInstallCmd(args []string, _ config.Config) error {
	if len(args) == 0 {
		lspUsage()

		return nil
	}

	target := strings.ToLower(args[0])
	if target == "all" {
		return lspInstallAll()
	}

	def := lsp.DefForLanguage(target)
	if def == nil {
		return fmt.Errorf("unknown language %q — run 'review lsp list' to see supported languages", target)
	}

	return installServer(*def)
}

func lspInstallAll() error {
	var failed []string

	for _, def := range lsp.Registry {
		if err := installServer(def); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", def.Language, err)
			failed = append(failed, def.Language)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to install: %s", strings.Join(failed, ", "))
	}

	return nil
}

func installServer(def lsp.ServerDef) error {
	argv := def.InstallCommand()
	if argv == nil {
		fmt.Printf("%-24s  manual install required:\n  %s\n\n", def.Language, def.Install.Note)

		return nil
	}

	fmt.Printf("%-24s  %s\n", def.Language, strings.Join(argv, " "))

	cmd := exec.CommandContext(context.Background(), argv[0], argv[1:]...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install %s: %w", def.Binary, err)
	}

	fmt.Printf("%-24s  ✓ done\n\n", "")

	return nil
}

func lspUsage() {
	fmt.Println("review lsp [list]")
	fmt.Println("review lsp install <language|all>")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list              show all known language servers and their status")
	fmt.Println("  install <lang>    install the server for a language (e.g. go, typescript, rust)")
	fmt.Println("  install all       install every server that can be installed automatically")
	fmt.Println()
	fmt.Println("Languages: go, typescript, python, rust, c, java, ruby, lua, shell")
}
