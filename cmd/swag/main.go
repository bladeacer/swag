// Command swag is the CLI of swag, the subtitle converter.
//
// This file is the v0.1.0 stub: it reports the version so that release
// builds and the .goreleaser.yaml build block are valid. The conversion
// commands arrive with v0.2.0 (kong flags, pterm output).
package main

import (
	"fmt"
	"os"
)

// Build information, set by -X ldflags in .goreleaser.yaml and the
// Makefile snapshot target.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && (args[0] == "-v" || args[0] == "--version") {
		fmt.Printf("swag %s (commit %s, built %s)\n", version, commit, date)
		return
	}
	fmt.Fprintln(os.Stderr, "swag does not convert yet. See ROADMAP.md: the CLI lands in v0.2.0.")
	os.Exit(2)
}
