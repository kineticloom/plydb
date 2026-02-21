package cmd

import "fmt"

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func RunVersion() {
	fmt.Printf("PlyDB %s (Build: %s, Commit: %s)\n", Version, BuildDate, Commit)
}
