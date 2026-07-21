package main

import "fund-trace/internal/cmd"

// version is injected via -ldflags at release build time (see .goreleaser.yml).
var version = "dev"

func main() {
	cmd.Execute(version)
}
