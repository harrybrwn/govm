package main

import (
	"fmt"
	"os"

	"github.com/harrybrwn/govm/cmd/govm/cli"
)

func main() {
	root := cli.NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\nRun 'govm help' for usage\n", err)
		os.Exit(1)
	}
}
