// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package cmd

import "fmt"

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func RunVersion() {
	fmt.Printf("PlyDB %s (Build: %s, Commit: %s)\n", Version, BuildDate, Commit)
	fmt.Printf("Copyright 2026 Paul Tzen\n")
	fmt.Printf("Licensed under the Apache License, Version 2.0.\n")
	fmt.Printf("Run 'plydb license' for the full license text.\n")
}
