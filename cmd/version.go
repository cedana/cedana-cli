package cmd

import (
	"fmt"
)

// used in main.go to set version info
func SetVersionInfo(version, commit, date string) {
	RootCmd.Version = fmt.Sprintf("%s (%s)", version, commit)
}
