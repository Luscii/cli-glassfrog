// Command glassfrog is a self-contained command-line surface over the
// Glassfrog v5 governance API.
package main

import (
	"os"

	"github.com/Luscii/cli-glassfrog/internal/cli"
)

func main() {
	// Dispatch through cli.Run, which resolves the invocation against the
	// assembled tree and returns a code-free Outcome category. The category has
	// no consumer yet: until Exit-Code Convention (a later spec) maps it to a
	// process code, the entrypoint maps minimally — success → 0, any error →
	// non-zero. This minimal mapping is a documented placeholder, not the final
	// convention.
	if _, err := cli.Run(cli.Assemble(), os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
