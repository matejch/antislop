package main

import (
	"fmt"
	"os"

	"github.com/matej/antislop/cmd"
)

const version = "0.1.0"

func main() {
	root := cmd.NewRootCmd()
	root.Version = version

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
