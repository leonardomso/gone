package main

import "github.com/leonardomso/gone/cmd"

// version is set by GoReleaser at build time via ldflags.
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
