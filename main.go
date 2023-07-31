package main

import (
	"github.com/cedana/cedana-cli/cmd"
)

// these get set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
