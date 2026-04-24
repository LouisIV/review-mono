package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"review/internal/client"
	"review/internal/config"
	"review/internal/daemon"
	"review/internal/git"
	"review/internal/models"
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
		printJSON(map[string]any{"id": session.ID, "branch": session.Branch, "base": session.Base, "commits": len(commits), "files_changed": len(files), "status": session.Status})
	} else {
		fmt.Printf("Opened review session for %s -> %s\n%d commits, %d files changed\n", session.Branch, session.Base, len(commits), len(files))
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
		printJSON(map[string]any{"branch": session.Branch, "base": session.Base, "status": session.Status, "commits": len(commits), "comments": map[string]int{"open": open, "resolved": resolved}})
	} else {
		fmt.Printf("Branch:   %s -> %s\nStatus:   %s\nCommits:  %d\nComments: %d open, %d resolved\n", session.Branch, session.Base, session.Status, len(commits), open, resolved)
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
			fmt.Printf("%-8s %-32s %-16s %s\n", commit.Hash, truncate(commit.Message, 32), truncate(commit.Author, 16), commit.Timestamp.Format(time.RFC3339))
		}
	}
	return nil
}

func commentCmd(args []string, g globals) error {
	if len(args) == 0 {
		return errors.New("comment subcommand required")
	}
	switch args[0] {
	case "add":
		return commentAdd(args[1:], g)
	case "list":
		return commentList(args[1:], g)
	case "resolve":
		return commentResolve(args[1:], g)
	case "delete":
		return commentDelete(args[1:], g)
	default:
		return fmt.Errorf("unknown comment subcommand %q", args[0])
	}
}

func commentAdd(args []string, g globals) error {
	author := models.AuthorHuman
	pos := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--author" && i+1 < len(args) {
			author = args[i+1]
			i++
		} else {
			pos = append(pos, args[i])
		}
	}
	if len(pos) < 2 {
		return errors.New("usage: review comment add file:line body")
	}
	file, line, err := parseLocation(pos[0])
	if err != nil {
		return err
	}
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}
	comment, err := c.AddComment(repo.Path, session, models.Comment{File: file, Line: line, Body: strings.Join(pos[1:], " "), Author: author})
	if err != nil {
		return err
	}
	if g.json {
		printJSON(comment)
	} else {
		fmt.Printf("Added comment %s on %s:%d\n", comment.ID, comment.File, comment.Line)
	}
	return nil
}

func commentList(args []string, g globals) error {
	filters := url.Values{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--unresolved":
			filters.Set("resolved", "false")
		case "--file":
			if i+1 < len(args) {
				filters.Set("file", args[i+1])
				i++
			}
		case "--author":
			if i+1 < len(args) {
				filters.Set("author", args[i+1])
				i++
			}
		}
	}
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}
	comments, err := c.Comments(repo.Path, session, filters)
	if err != nil {
		return err
	}
	if g.json {
		printJSON(map[string]any{"comments": comments})
	} else {
		for _, comment := range comments {
			state := "open"
			if comment.Resolved {
				state = "resolved"
			}
			fmt.Printf("%s %s %s:%d %s\n", comment.ID, state, comment.File, comment.Line, comment.Body)
		}
	}
	return nil
}

func commentResolve(args []string, g globals) error {
	all := len(args) == 1 && args[0] == "--all"
	if len(args) == 0 {
		return errors.New("comment id or --all required")
	}
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}
	ids := args
	if all {
		comments, err := c.Comments(repo.Path, session, url.Values{"resolved": []string{"false"}})
		if err != nil {
			return err
		}
		ids = nil
		for _, comment := range comments {
			ids = append(ids, comment.ID)
		}
	}
	for _, id := range ids {
		if _, err := c.PatchComment(repo.Path, session, id, map[string]bool{"resolved": true}); err != nil {
			return err
		}
	}
	if g.json {
		printJSON(map[string]any{"resolved": ids})
	} else {
		fmt.Printf("Resolved %d comment(s)\n", len(ids))
	}
	return nil
}

func commentDelete(args []string, g globals) error {
	if len(args) != 1 {
		return errors.New("comment id required")
	}
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}
	if err := c.DeleteComment(repo.Path, session, args[0]); err != nil {
		return err
	}
	if g.json {
		printJSON(map[string]any{"deleted": true, "id": args[0]})
	} else {
		fmt.Printf("Deleted comment %s\n", args[0])
	}
	return nil
}

func describeCmd(args []string, g globals) error {
	printOnly := false
	prompt := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--print" {
			printOnly = true
		}
		if args[i] == "--prompt" && i+1 < len(args) {
			prompt = args[i+1]
			i++
		}
	}
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}
	desc, err := c.GenerateDescription(repo.Path, session, prompt)
	if err != nil {
		return err
	}
	if printOnly {
		fmt.Print(desc.Body)
		return nil
	}
	if g.json {
		printJSON(desc)
	} else {
		fmt.Printf("Generated MR description (saved to .git/reviews/sessions/%s/description.md)\n\n%s", session.ID, desc.Body)
	}
	return nil
}

func descriptionCmd(args []string, g globals) error {
	if len(args) == 0 {
		return errors.New("description subcommand required")
	}
	session, repo, c, err := current(g)
	if err != nil {
		return err
	}
	switch args[0] {
	case "show":
		desc, err := c.Description(repo.Path, session)
		if err != nil {
			return err
		}
		if g.json {
			printJSON(desc)
		} else {
			fmt.Print(desc.Body)
		}
	case "edit":
		return editDescription(repo.Path, session, c)
	default:
		return fmt.Errorf("unknown description subcommand %q", args[0])
	}
	return nil
}

func approveCmd(args []string, g globals, cfg config.Config) error {
	noBrowser, dryRun := false, false
	for _, arg := range args {
		if arg == "--no-browser" {
			noBrowser = true
		}
		if arg == "--dry-run" {
			dryRun = true
		}
	}
	repo, err := git.Open(g.repo)
	if err != nil {
		return err
	}
	if err := ensureDaemon(g.port); err != nil {
		return err
	}
	out, err := client.New(g.port).Approve(repo.Path, dryRun)
	if err != nil {
		return err
	}
	mrURL, _ := out["mr_url"].(string)
	if !dryRun && !noBrowser && cfg.OpenBrowserOnApprove && mrURL != "" {
		_ = openBrowser(mrURL)
	}
	if g.json {
		printJSON(out)
	} else {
		if dryRun {
			fmt.Println("Dry run: push skipped")
		} else {
			fmt.Printf("Pushed %s to origin\n", out["branch"])
		}
		if mrURL != "" {
			fmt.Println(mrURL)
		}
	}
	return nil
}

func requestChangesCmd(args []string, g globals) error {
	message := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--message" && i+1 < len(args) {
			message = args[i+1]
			i++
		}
	}
	repo, err := git.Open(g.repo)
	if err != nil {
		return err
	}
	if err := ensureDaemon(g.port); err != nil {
		return err
	}
	session, err := client.New(g.port).RequestChanges(repo.Path, message)
	if err != nil {
		return err
	}
	if g.json {
		printJSON(session)
	} else {
		fmt.Printf("Changes requested for %s\n", session.Branch)
	}
	return nil
}

func watchCmd(args []string, g globals) error {
	eventFilter := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--event" && i+1 < len(args) {
			eventFilter = args[i+1]
			i++
		}
	}
	if err := ensureDaemon(g.port); err != nil {
		return err
	}
	return client.New(g.port).Watch(func(event models.Event) bool {
		if eventFilter == "" || event.Event == eventFilter {
			b, _ := json.Marshal(event)
			fmt.Println(string(b))
		}
		return event.Event != "approved"
	})
}

func current(g globals) (models.Session, git.Repo, client.Client, error) {
	repo, err := git.Open(g.repo)
	if err != nil {
		return models.Session{}, git.Repo{}, client.Client{}, err
	}
	if err := ensureDaemon(g.port); err != nil {
		return models.Session{}, git.Repo{}, client.Client{}, err
	}
	c := client.New(g.port)
	sessions, err := c.Sessions(repo.Path)
	if err != nil {
		return models.Session{}, git.Repo{}, client.Client{}, err
	}
	branch, err := repo.Branch()
	if err != nil {
		return models.Session{}, git.Repo{}, client.Client{}, err
	}
	for i := len(sessions) - 1; i >= 0; i-- {
		if sessions[i].Repo == repo.Path && sessions[i].Branch == branch && sessions[i].Status != models.StatusApproved {
			return sessions[i], repo, c, nil
		}
	}
	return models.Session{}, git.Repo{}, client.Client{}, errors.New("no active review session; run review open")
}

func parseGlobalFlags(args []string, g *globals) []string {
	out := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			g.json = true
		case "--repo":
			if i+1 < len(args) {
				g.repo = args[i+1]
				i++
			}
		case "--port":
			if i+1 < len(args) {
				g.port, _ = strconv.Atoi(args[i+1])
				i++
			}
		default:
			out = append(out, args[i])
		}
	}
	return out
}

func ensureDaemon(port int) error {
	c := client.New(port)
	if c.Health() == nil {
		return nil
	}
	if _, err := startDaemon(port); err != nil {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c.Health() == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("daemon did not become healthy")
}

func startDaemon(port int) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	logFile, err := os.OpenFile(filepath.Join(dir, "daemon.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	cmd := exec.Command(exe, "daemon", "run", "--port", strconv.Itoa(port))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	_ = os.WriteFile(filepath.Join(dir, "daemon.pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	pid := cmd.Process.Pid
	return pid, cmd.Process.Release()
}

func stopDaemon(jsonOut bool) error {
	pid, err := readPID()
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(config.ConfigDir(), "daemon.pid"))
	if jsonOut {
		printJSON(map[string]any{"stopped": true, "pid": pid})
	} else {
		fmt.Printf("Stopped daemon pid %d\n", pid)
	}
	return nil
}

func daemonStatus(g globals) error {
	running := client.New(g.port).Health() == nil
	pid, _ := readPID()
	if g.json {
		printJSON(map[string]any{"running": running, "port": g.port, "pid": pid})
	} else if running {
		fmt.Printf("Daemon running on port %d", g.port)
		if pid > 0 {
			fmt.Printf(" (pid %d)", pid)
		}
		fmt.Println()
	} else {
		fmt.Println("Daemon not running")
	}
	return nil
}

func readPID() (int, error) {
	b, err := os.ReadFile(filepath.Join(config.ConfigDir(), "daemon.pid"))
	if err != nil {
		return 0, errors.New("daemon pid not found")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func parseLocation(s string) (string, int, error) {
	idx := strings.LastIndex(s, ":")
	if idx <= 0 || idx == len(s)-1 {
		return "", 0, errors.New("location must be file:line")
	}
	line, err := strconv.Atoi(s[idx+1:])
	if err != nil {
		return "", 0, err
	}
	return s[:idx], line, nil
}

func editDescription(repoPath string, session models.Session, c client.Client) error {
	desc, _ := c.Description(repoPath, session)
	tmp, err := os.CreateTemp("", "review-description-*.md")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(desc.Body); err != nil {
		return err
	}
	_ = tmp.Close()
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	b, err := os.ReadFile(tmp.Name())
	if err != nil {
		return err
	}
	_, err = c.SetDescription(repoPath, session, string(b))
	return err
}

func openBrowser(rawurl string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawurl).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawurl).Start()
	default:
		return exec.Command("xdg-open", rawurl).Start()
	}
}

func printJSON(value any) {
	b, _ := json.MarshalIndent(value, "", "  ")
	fmt.Println(string(b))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "..."
}

func usage() {
	fmt.Println("review <command> [flags]")
	fmt.Println("commands: daemon, open, status, close, diff, commits, comment, describe, description, approve, request-changes, watch")
}
