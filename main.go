package main

import (
	"os"

	"review/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:]))
}
