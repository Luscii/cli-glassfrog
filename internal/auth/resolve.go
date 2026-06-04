package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// returns the first source that yields a usable token (ADR-2). A present-but-
// tokenless file is skipped; a missing file is skipped; an unreadable or
// unparseable file fails loud with a typed error naming the path (never falling
// through to another source). Nothing found anywhere is Source: None with no
// error. The token value never appears in any error.
func resolve(startDir, homeDir string) (Resolution, error) {
	if token := getenv(envTokenVar); strings.TrimSpace(token) != "" {
		// A usable environment value (non-empty after trimming) is used as-is
		// and short-circuits all file reads. Empty, unset, or whitespace-only
		// falls through to the file search — a blank value is treated as absent,
		// matching the "usable token" rule applied to file tokens.
		return Resolution{Token: token, Source: SourceEnvironment}, nil
	}

	for _, dir := range candidateDirs(startDir, homeDir) {
		path := filepath.Join(dir, credentialsFileName)
		token, found, err := readCredentialsFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue // missing at this location → try the next
			}
			return Resolution{}, err // unreadable/unparseable → fail loud
		}
		if !found {
			continue // exists but holds no usable token → skip to the next source
		}
		return Resolution{Token: token, Source: SourceFile, Path: path}, nil
	}

	return Resolution{Source: SourceNone}, nil
}

// candidateDirs builds the ordered, de-duplicated directory search list: the
// start directory, each ancestor up to the filesystem root, then the home
// directory if it was not already visited on the ascent. De-duplication keeps a
// home directory that lies on the ascent path from being read twice and is read
// at its natural walk-up position.
func candidateDirs(startDir, homeDir string) []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(d string) {
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		dirs = append(dirs, d)
	}

	if startDir != "" {
		dir := filepath.Clean(startDir)
		for {
			add(dir)
			parent := filepath.Dir(dir)
			if parent == dir { // reached the filesystem root — stop, no infinite loop
				break
			}
			dir = parent
		}
	}
	if homeDir != "" {
		add(filepath.Clean(homeDir))
	}
	return dirs
}
