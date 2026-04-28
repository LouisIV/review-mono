package cmd

import (
	"errors"
	"fmt"

	"review/internal/client"
	"review/internal/config"
	"review/internal/git"
	"review/internal/models"
	"review/internal/tui"
)

func tuiCmd(args []string, g globals, cfg config.Config) error {
	base := cfg.DefaultBaseBranch
	pos := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--base":
			if i+1 >= len(args) {
				return errors.New("--base requires a branch")
			}
			base = args[i+1]
			i++
		default:
			if isHelpArg(args[i]) {
				tuiUsage()

				return nil
			}
			pos = append(pos, args[i])
		}
	}

	if len(pos) > 0 {
		g.repo = pos[0]
	}

	repo, err := git.Open(g.repo)
	if err != nil {
		return err
	}
	if err := ensureDaemon(g.port); err != nil {
		return err
	}

	c := client.New(g.port)
	session, err := currentSessionOrOpen(repo, c, base)
	if err != nil {
		return err
	}

	return tui.Run(tui.Options{RepoPath: repo.Path, Session: session, Client: c, Config: cfg})
}

func currentSessionOrOpen(repo git.Repo, c client.Client, base string) (models.Session, error) {
	sessions, err := c.Sessions(repo.Path)
	if err != nil {
		return models.Session{}, err
	}

	branch, err := repo.Branch()
	if err != nil {
		return models.Session{}, err
	}

	for i := len(sessions) - 1; i >= 0; i-- {
		s := sessions[i]
		if s.Repo == repo.Path && s.Branch == branch && s.Status != models.StatusApproved {
			return s, nil
		}
	}

	return c.Open(repo.Path, base)
}

func tuiUsage() {
	fmt.Println("review tui [repo] [--base <branch>]")
	fmt.Println("opens or creates the current branch review session in a keyboard-first TUI")
}
