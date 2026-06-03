// Command glassfrog is a self-contained command-line surface over the
// Glassfrog v5 governance API.
package main

import (
	"os"

	"github.com/Luscii/cli-glassfrog/internal/cli"
)

func main() {
	if err := cli.Assemble().Execute(); err != nil {
		// Non-zero on any execution error. Standardized exit codes are the
		// Exit-Code Convention capability's concern (a later spec).
		os.Exit(1)
	}
}
