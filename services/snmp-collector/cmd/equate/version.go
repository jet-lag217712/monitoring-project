package main

import (
	"fmt"
	"os"
)

func runVersion(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "version accepts no arguments")
		return 2
	}
	fmt.Printf("equate %s (%s) built %s\n", buildVersion, buildGitCommit, buildTime)
	return 0
}
