package main

import (
	"fmt"
)

// Injected during build time via -ldflags linker settings
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func main() {
	fmt.Printf("FlexiConnect Daemon %s (commit: %s, build time: %s)\n", Version, Commit, BuildTime)
	fmt.Println("Run 'make help' to see all available commands.")
}
