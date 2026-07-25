// Command shrike is the single entrypoint for the Go side of EVE-KILL: the
// API, the queue workers, and the cron runner. Every run-mode is a subcommand,
// so one binary and one image cover every pod.
package main

import "github.com/eve-kill/shrike/internal/cli"

// Stamped by the linker; see the Makefile.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	cli.Execute(version, commit)
}
