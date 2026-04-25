package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"review/internal/tuidemo"
)

func widgetCmd(args []string, _ globals) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		widgetUsage()

		return nil
	}

	if args[0] == "list" {
		fmt.Println(strings.Join(tuidemo.Names(), "\n"))

		return nil
	}

	name := args[0]
	props := tuidemo.DefaultProps()
	tree := false
	interactive := false
	rawProps := ""
	events := []string{}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--tree":
			tree = true
		case "--interactive", "-I":
			interactive = true
		case "--props":
			if i+1 >= len(args) {
				return errors.New("--props requires JSON")
			}

			rawProps = args[i+1]
			i++
		case "--event":
			if i+1 >= len(args) {
				return errors.New("--event requires a name")
			}

			events = append(events, args[i+1])
			i++
		case "--width":
			if i+1 >= len(args) {
				return errors.New("--width requires a number")
			}

			width, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}

			props.Width = width
			i++
		case "--height":
			if i+1 >= len(args) {
				return errors.New("--height requires a number")
			}

			height, err := strconv.Atoi(args[i+1])
			if err != nil {
				return err
			}

			props.Height = height
			i++
		default:
			return fmt.Errorf("unknown widget flag %q", args[i])
		}
	}

	var err error
	props, err = tuidemo.MergeProps(rawProps, props)
	if err != nil {
		return err
	}

	for _, event := range events {
		props = tuidemo.ApplyEvent(props, event)
	}

	if interactive {
		if tree {
			return errors.New("--interactive cannot be combined with --tree")
		}

		return tuidemo.RunInteractive(name, props)
	}

	out, err := tuidemo.Render(name, props, tree)
	if err != nil {
		return err
	}

	fmt.Print(out)

	return nil
}

func widgetUsage() {
	fmt.Println("review widget <name> [--tree] [--interactive|-I] [--props JSON] [--event name] [--width N] [--height N]")
	fmt.Println("review widget list")
	fmt.Println("widgets: " + strings.Join(tuidemo.Names(), ", "))
	fmt.Println("events: comment-line, comment-range, clear-selection, menu-down, menu-up, search-root, search-usage, goto-picker, goto-doc, page-down, page-up, half-page-down, half-page-up, scroll-right, scroll-left")
}
