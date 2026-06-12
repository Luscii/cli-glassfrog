package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"golang.org/x/term"
)

// loginSeam supplies everything the auth login command reads from the outside
// world. It is the only place that touches the real stdin / TTY / environment /
// directories and the terminal (ADR-3) — the resolution logic (authlogin.go)
// stays pure, and tests inject a fake seam so no test reads the real stdin,
// terminal, env, or home directory.
type loginSeam interface {
	// gatherInputs reads the token sources and interactivity for this
	// invocation, honoring precedence's need to not read stdin when an argument
	// is present or stdin is a terminal.
	gatherInputs(args []string) (tokenInputs, error)
	homeDir() (string, error)
	startDir() (string, error)
	// interactor is consulted only in an interactive (TTY) session.
	interactor() interactor
}

// interactor is the terminal-interaction surface used only when stdin is a TTY:
// the non-echoing token prompt, the confirm-before-replace, and the location
// choice. A fake implementation drives these in tests (the production terminal
// paths are exercised through the BDD seam, ADR-3).
type interactor interface {
	// promptToken reads a token from the terminal without echoing the typed
	// characters. Called only when interactive and no other source supplied one.
	promptToken() (string, error)
	// confirmReplace asks whether to change the existing token at path.
	confirmReplace(path string) (bool, error)
	// chooseLocations offers the current-directory file, the home file, or both
	// as write targets, returning the chosen paths.
	chooseLocations(homePath, cwdPath string) ([]string, error)
}

// gatherInputsFrom builds tokenInputs from already-read raw materials. Pure and
// testable; the production seam supplies the real os values. Stdin is read only
// when no argument was supplied and stdin is not a terminal — reading a TTY here
// would block on the prompt's eventual input rather than returning a pipe.
func gatherInputsFrom(args []string, env string, isTTY bool, readStdin func() (string, error)) (tokenInputs, error) {
	in := tokenInputs{env: env, isTTY: isTTY}
	if len(args) >= 1 {
		in.arg, in.argGiven = args[0], true
	}
	if !in.argGiven && !isTTY {
		s, err := readStdin()
		if err != nil {
			return in, err
		}
		in.stdin, in.stdinPiped = s, true
	}
	return in, nil
}

// productionSeam binds the real os values. It is the single reader of
// os.Stdin / the TTY / os.Getenv / the working and home directories.
type productionSeam struct{}

// maxPipedTokenBytes caps how much piped stdin the command will read. A token
// is tiny; this bounds accidental (or hostile) large pipes so the command never
// slurps an arbitrarily large input into memory.
const maxPipedTokenBytes = 64 << 10 // 64 KiB

func (productionSeam) gatherInputs(args []string) (tokenInputs, error) {
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	return gatherInputsFrom(args, os.Getenv(auth.EnvVarToken), isTTY, func() (string, error) {
		return readBoundedStdin(os.Stdin)
	})
}

// readBoundedStdin reads a piped token from r up to maxPipedTokenBytes, naming
// "token" in the overflow message. It is the token-specific wrapper over
// readBoundedStdinN (other callers — e.g. the -o stdin template path — pass their
// own cap and noun so the error reads correctly for their input).
func readBoundedStdin(r io.Reader) (string, error) {
	return readBoundedStdinN(r, maxPipedTokenBytes, "token")
}

// readBoundedStdinN reads r up to limit bytes and errors (naming noun, e.g. "token"
// or "template") if the input exceeds the cap, rather than reading an unbounded
// amount into memory.
func readBoundedStdinN(r io.Reader, limit int, noun string) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return "", err
	}
	if len(data) > limit {
		return "", fmt.Errorf("piped %s exceeds the %d-byte limit", noun, limit)
	}
	return string(data), nil
}

func (productionSeam) homeDir() (string, error)  { return os.UserHomeDir() }
func (productionSeam) startDir() (string, error) { return os.Getwd() }
func (productionSeam) interactor() interactor    { return ttyInteractor{} }

// ttyInteractor is the production terminal interaction. Prompts are written to
// stderr so stdout stays machine-clean for the success line and piping; the
// token is read without echo and never printed.
type ttyInteractor struct{}

func (ttyInteractor) promptToken() (string, error) {
	fmt.Fprint(os.Stderr, "Token: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (ttyInteractor) confirmReplace(path string) (bool, error) {
	fmt.Fprintf(os.Stderr, "A credential already exists at %s. Replace it? [y/N]: ", path)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (ttyInteractor) chooseLocations(homePath, cwdPath string) ([]string, error) {
	fmt.Fprintf(os.Stderr, "Write to [h]ome (%s), [c]wd (%s), or [b]oth? [h/c/b]: ", homePath, cwdPath)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "c", "cwd":
		return []string{cwdPath}, nil
	case "b", "both":
		return []string{cwdPath, homePath}, nil
	default:
		return []string{homePath}, nil
	}
}
