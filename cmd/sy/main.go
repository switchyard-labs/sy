// sy — the first-class command-line client for Switchyard.
package main

import (
	"fmt"
	"os"

	"github.com/switchyard-labs/sy/internal/commands"
)

func main() {
	if err := commands.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err.Error())
		os.Exit(commands.ExitCodeFor(err))
	}
}