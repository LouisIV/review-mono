package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"review/internal/client"
	"review/internal/config"
	"review/internal/daemon"
	"review/internal/git"
)

type globals struct {
	json bool
	repo string
	port int
}

func Execute(args []string) int {
	cfg := config.Load()
	g := globals{repo: ".", port: cfg.DaemonPort}

	args = parseGlobalFlags(args, &g)
	if len(args) > 0 && isHelpArg(args[0]) {
		usage()

		return 0
	}

	if len(args) == 0 {
		usage()

		return 1
	}

	if g.port == 0 {
		g.port = 7080
	}

	err := run(args, g, cfg)
	if err != nil {
		if g.json {
			printJSON(map[string]any{"error": err.Error()})
		} else {
			fmt.Fprintln(os.Stderr, "error:", err)
		}

		return 1
	}

	return 0
}

func run(args []string, g globals, cfg config.Config) error {
	switch args[0] {
	case "help":
		usage()

		return nil
	case "daemon":
		return daemonCmd(args[1:], g, cfg)
	case "open":
		return openCmd(args[1:], g, cfg)
	case "status":
		return statusCmd(g)
	case "close":
		return closeCmd(g)
	case "diff":
		return diffCmd(args[1:], g)
	case "commits":
		return commitsCmd(g)
	case "comment":
		return commentCmd(args[1:], g)
	case "describe":
		return describeCmd(args[1:], g)
	case "description":
		return descriptionCmd(args[1:], g)
	case "approve":
		return approveCmd(args[1:], g, cfg)
	case "request-changes":
		return requestChangesCmd(args[1:], g)
	case "watch":
		return watchCmd(args[1:], g)
	case "tui":
		return tuiCmd(args[1:], g, cfg)
	case "widget":
		return widgetCmd(args[1:], g)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func daemonCmd(args []string, g globals, cfg config.Config) error {
	if len(args) == 0 {
		return errors.New("daemon subcommand required")
	}

	switch args[0] {
	case "run":
		return daemon.New(cfg).ListenAndServe(g.port)
	case "start":
		if client.New(g.port).Health() == nil {
			if g.json {
				printJSON(map[string]any{"running": true, "port": g.port})
			} else {
				fmt.Printf("Daemon already running on port %d\n", g.port)
			}

			return nil
		}

		pid, err := startDaemon(g.port)
		if err != nil {
			return err
		}

		if g.json {
			printJSON(map[string]any{"running": true, "port": g.port, "pid": pid})
		} else {
			fmt.Printf("Daemon started on port %d (pid %d)\n", g.port, pid)
		}

		return nil
	case "stop":
		return stopDaemon(g.json)
	case "status":
		return daemonStatus(g)
	default:
		return fmt.Errorf("unknown daemon subcommand %q", args[0])
	}
}

func openCmd(args []string, g globals, cfg config.Config) error {
	base := cfg.DefaultBaseBranch
	pos := []string{}

	for i := 0; i < len(args); i++ {
		if args[i] == "--base" && i+1 < len(args) {
			base = args[i+1]
			i++
		} else {
			pos = append(pos, args[i])
		}
	}

	if len(pos) > 0 {
		g.repo = pos[0]
	}

	if err := ensureDaemon(g.port); err != nil {
		return err
	}

	repo, err := git.Open(g.repo)
	if err != nil {
		return err
	}

	c := client.New(g.port)

	session, err := c.Open(repo.Path, base)
	if err != nil {
		return err
	}

	commits, _ := c.Commits(repo.Path, session)

	files, _, _ := c.Diff(repo.Path, session, "", "", true)
	if g.json {
		printJSON(
			map[string]any{
				"id":            session.ID,
				"branch":        session.Branch,
				"base":          session.Base,
				"commits":       len(commits),
				"files_changed": len(files),
				"status":        session.Status,
			},
		)
	} else {
		fmt.Printf(
			"Opened review session for %s -> %s\n%d commits, %d files changed\n",
			session.Branch,
			session.Base,
			len(commits),
			len(files),
		)
	}

	return nil
}

func statusCmd(g globals) error {
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	commits, _ := c.Commits(repo.Path, session)
	comments, _ := c.Comments(repo.Path, session, nil)
	open, resolved := 0, 0

	for _, comment := range comments {
		if comment.Resolved {
			resolved++
		} else {
			open++
		}
	}

	if g.json {
		printJSON(
			map[string]any{
				"branch":   session.Branch,
				"base":     session.Base,
				"status":   session.Status,
				"commits":  len(commits),
				"comments": map[string]int{"open": open, "resolved": resolved},
			},
		)
	} else {
		fmt.Printf(
			"Branch:   %s -> %s\nStatus:   %s\nCommits:  %d\nComments: %d open, %d resolved\n",
			session.Branch,
			session.Base,
			session.Status,
			len(commits),
			open,
			resolved,
		)
	}

	return nil
}

func closeCmd(g globals) error {
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	if err := c.CloseSession(repo.Path, session); err != nil {
		return err
	}

	if g.json {
		printJSON(map[string]any{"closed": true, "id": session.ID})
	} else {
		fmt.Printf("Closed review session %s\n", session.ID)
	}

	return nil
}

func diffCmd(args []string, g globals) error {
	file, commit := "", ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 < len(args) {
				file = args[i+1]
				i++
			}
		case "--commit":
			if i+1 < len(args) {
				commit = args[i+1]
				i++
			}
		}
	}

	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	files, raw, err := c.Diff(repo.Path, session, file, commit, false)
	if err != nil {
		return err
	}

	if g.json {
		printJSON(map[string]any{"files": files})
	} else {
		fmt.Print(raw)
	}

	return nil
}

func commitsCmd(g globals) error {
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}

	commits, err := c.Commits(repo.Path, session)
	if err != nil {
		return err
	}

	if g.json {
		printJSON(map[string]any{"commits": commits})
	} else {
		for _, commit := range commits {
			fmt.Printf(
				"%-8s %-32s %-16s %s\n",
				commit.Hash,
				truncate(commit.Message, 32),
				truncate(commit.Author, 16),
				commit.Timestamp.Format(time.RFC3339),
			)
		}
	}

	return nil
}

func usage() {
	fmt.Println("review <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, command := range rootCommands() {
		fmt.Printf("  %-16s %s\n", command.name, command.summary)
	}
	fmt.Println()
	fmt.Println("Global flags:")
	fmt.Println("  --repo <path>      repository path (default .)")
	fmt.Println("  --port <port>      daemon port")
	fmt.Println("  --json             print machine-readable JSON")
	fmt.Println()
	fmt.Println("run 'review <command> --help' for command-specific help")
}

type commandHelp struct {
	name    string
	summary string
}

func rootCommands() []commandHelp {
	return []commandHelp{
		{name: "daemon", summary: "run, start, stop, or inspect the local daemon"},
		{name: "open", summary: "open a review session for the current branch"},
		{name: "status", summary: "show the current review session summary"},
		{name: "close", summary: "close the current review session"},
		{name: "diff", summary: "print the review diff"},
		{name: "commits", summary: "list commits included in the review"},
		{name: "comment", summary: "add, list, resolve, or delete review comments"},
		{name: "describe", summary: "generate an MR description"},
		{name: "description", summary: "show or edit the saved MR description"},
		{name: "approve", summary: "push and mark the review approved"},
		{name: "request-changes", summary: "mark the review as changes requested"},
		{name: "watch", summary: "stream review daemon events"},
		{name: "tui", summary: "open the keyboard-first review interface"},
		{name: "widget", summary: "render or interact with TUI widget demos"},
	}
}

func describeUsage() {
	fmt.Println("review describe [--print] [--prompt <text>] [--provider <provider>]")
	fmt.Println("providers: auto, anthropic, claude-cli, codex-cli, fallback")
}
