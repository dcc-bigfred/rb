package main

import (
	"os"

	"github.com/dcc-bigfred/rb/cli"
)

func main() {
	cmd := cli.NewRootCommand()
	args := os.Args
	if args != nil {
		args = args[1:]
		cmd.SetArgs(args)
	}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
