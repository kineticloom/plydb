// Copyright 2026 Paul Tzen
// SPDX-License-Identifier: Apache-2.0

package cmd

import "fmt"

func RunLicense(text string) {
	fmt.Printf("Copyright 2026 Paul Tzen\n")
	fmt.Printf("Licensed under the Apache License, Version 2.0.\n\n")
	fmt.Print(text)
}
