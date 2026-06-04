// Command glassfrog is a self-contained command-line surface over the
// Glassfrog v5 governance API.
package main

import (
	"os"

	"github.com/Luscii/cli-glassfrog/internal/cli"
)

func main() {
	// cli.Main assembles the command tree, dispatches the invocation, and maps
	// the outcome to the canonical exit code (Exit-Code Convention, 004),
	// recovering a panic to code 1. The entrypoint only forwards that code to
	// the process — every code selection lives behind cli.Main.
	os.Exit(cli.Main())
}
