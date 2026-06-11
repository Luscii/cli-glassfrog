package resolve

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// OSRoots returns the directories the file walk searches: the working directory
// (an error if it cannot be determined) and the home directory ("" if it cannot
// be determined — the home fallback is dropped, never a hard failure). It mirrors
// the getwd/userHomeDir preamble the existing *FromOS resolvers share (§49).
func OSRoots() (startDir, homeDir string, err error) {
	startDir, err = os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("could not determine the working directory: %w", err)
	}
	if homeDir, err = os.UserHomeDir(); err != nil {
		homeDir = "" // no home → drop the home fallback rather than fail
	}
	return startDir, homeDir, nil
}

// EnvFromOS is FromEnv(os.Getenv, names...) — the thin production binding over the
// pure env constructor (ADR-4). Tests use FromEnv directly with a fake lookup.
func EnvFromOS(names ...string) Source {
	return FromEnv(os.Getenv, names...)
}

// StdinFromOS is FromStdin bound to a maxStdinBytes-bounded os.Stdin reader and
// term.IsTerminal(int(os.Stdin.Fd())) — the thin production binding over the pure
// stdin constructor (ADR-4). Tests use FromStdin directly with an in-memory
// reader and isTTY flag.
func StdinFromOS() Source {
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	return FromStdin(func() (string, error) {
		return readBoundedStdin(os.Stdin)
	}, isTTY)
}
