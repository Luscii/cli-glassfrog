package auth

import (
	"fmt"
	"os"

	resolvepkg "github.com/Luscii/cli-glassfrog/internal/resolve"
)

// Source names where a resolved credential came from. SourceNone is the zero
// value, so a bare Resolution{} reads as "nothing found".
type Source int

const (
	SourceNone        Source = iota // no credential found anywhere — a normal outcome
	SourceEnvironment               // the GLASSFROG_TOKEN environment variable
	SourceFile                      // a .glassfrogrc file (Path names which one)
)

func (s Source) String() string {
	switch s {
	case SourceEnvironment:
		return "environment"
	case SourceFile:
		return "file"
	default:
		return "none"
	}
}

// Resolution is Credential Discovery's code-free output, consumed by Request
// Authentication (007). Source and Path are safe to display; Token is a secret
// and must never be rendered, logged, or placed in an error. Its String method
// redacts Token so accidental formatting (e.g. fmt.Errorf("%+v", res) in tests
// or future logging) cannot leak the credential; read the value through the
// Token field, never through formatting.
type Resolution struct {
	Token  string // the resolved credential; set only when Source is Environment or File
	Source Source
	Path   string // the file the token was read from, when Source is File; empty otherwise
}

// String renders a Resolution with the Token redacted, so the common formatting
// verbs (%v, %s, %+v) cannot leak the secret. Source and Path — the safe-to-
// display parts — are shown in full. The token is reported as present-but-hidden
// or absent, never verbatim.
func (r Resolution) String() string {
	token := "<none>"
	if r.Token != "" {
		token = "<redacted>"
	}
	return fmt.Sprintf("Resolution{Source: %s, Path: %q, Token: %s}", r.Source, r.Path, token)
}

// Production seam: the only places the package reads process/OS globals. They
// are package variables so tests can exercise resolution hermetically over temp
// directories and a controlled environment, never the developer's real home
// directory or working directory (ADR-5).
var (
	getwd       = os.Getwd
	userHomeDir = os.UserHomeDir
	getenv      = os.Getenv
)

// Resolve answers "what token are we acting as, right now, in this directory?"
// using the real working directory, home directory, and environment. It is the
// thin production entrypoint over resolve — the seam binding the pure algorithm
// to the OS globals. A working directory that cannot be determined is an error;
// a home directory that cannot be determined simply drops the home fallback.
func Resolve() (Resolution, error) {
	startDir, err := getwd()
	if err != nil {
		return Resolution{}, fmt.Errorf("could not determine the working directory: %w", err)
	}
	homeDir, err := userHomeDir()
	if err != nil {
		homeDir = "" // no home → skip the home fallback rather than fail
	}
	return resolve(startDir, homeDir)
}

// resolve walks the precedence chain — environment variable, then the nearest
// .glassfrogrc up the directory tree from startDir, then the home file — and
// returns the first source that yields a usable token (ADR-2). It composes the one
// shared resolve walk (039) over two sources and no default: FromEnv reads
// GLASSFROG_TOKEN (a value empty after trimming does not yield) and FromFile is
// rcfile.Resolve over the "token" key (a present-but-tokenless or missing file is
// skipped; an unreadable or unparseable file fails loud with rcfile's typed error
// naming the path, never falling through). The optional-setting shape omits a
// Default rung, so nothing found anywhere is the resolve KindNone outcome, mapped
// back to Source: None with no error.
//
// The generic resolve.Resolution{Value, Provenance} is mapped onto auth's existing
// surface — KindEnv→SourceEnvironment, KindFile→SourceFile (Path = Origin),
// KindNone→SourceNone — so consumers (007) are untouched. The token reaches
// auth.Resolution.Token only; the intermediate resolve.Resolution (which has no
// redacting String) is never formatted or logged, preserving secret hygiene.
func resolve(startDir, homeDir string) (Resolution, error) {
	res, err := resolvepkg.Resolve(
		resolvepkg.FromEnv(getenv, envTokenVar),
		resolvepkg.FromFile(startDir, homeDir, tokenKey),
	)
	if err != nil {
		return Resolution{}, err // unreadable/unparseable → fail loud
	}

	switch res.Provenance.Kind {
	case resolvepkg.KindEnv:
		return Resolution{Token: res.Value, Source: SourceEnvironment}, nil
	case resolvepkg.KindFile:
		return Resolution{Token: res.Value, Source: SourceFile, Path: res.Provenance.Origin}, nil
	default:
		return Resolution{Source: SourceNone}, nil
	}
}
