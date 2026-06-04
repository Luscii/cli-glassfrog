// Package auth resolves the Glassfrog API token the CLI operates as. It answers
// one question — "what token are we acting as, right now, in this directory?" —
// by consulting the GLASSFROG_TOKEN environment variable, then a nearest-wins
// walk-up of .glassfrogrc files, then the home-directory file (Credential
// Discovery, 005). It registers no command and prints nothing; Request
// Authentication (007) consumes the Resolution it returns, and Credential
// Storage (006) will add the writer beside the shared file-format reader so the
// read and write sides cannot drift.
//
// Secret hygiene is a package-wide invariant: the token value never appears in
// any error message, log line, or other output. Errors carry only the offending
// file path.
package auth

import (
	"fmt"
	"os"
	"strings"
)

// credentialsFileName and envTokenVar are the [ASSUMED] file name and
// environment variable for token discovery, centralized here as the single
// source of truth shared with Credential Storage (006). Both are provisional
// pending reconciliation with 006 before either capability ships.
const (
	credentialsFileName = ".glassfrogrc"
	envTokenVar         = "GLASSFROG_TOKEN"

	// tokenKey is the .glassfrogrc key carrying the credential. Other keys are
	// ignored (forward-compatible).
	tokenKey = "token"
)

// FormatError reports that a candidate .glassfrogrc exists and was read but
// could not be parsed — it held a non-blank, non-comment line with no '='. It
// names only the path, never the file's contents, so a broken credential fails
// loud without leaking the token.
type FormatError struct {
	Path string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("credentials file %s is malformed: a non-comment line is not a key=value pair", e.Path)
}

// ReadError reports that a candidate .glassfrogrc could not be read (e.g.
// permission denied, or — distinguishable via errors.Is(err, os.ErrNotExist) —
// simply absent). It wraps the underlying filesystem error, which names only
// the path, never any token.
type ReadError struct {
	Path string
	Err  error
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("credentials file %s could not be read: %v", e.Path, e.Err)
}

func (e *ReadError) Unwrap() error { return e.Err }

// readCredentialsFile reads the .glassfrogrc at path and returns the token under
// the "token" key. It is the shared file-format reader (ADR-1/ADR-3) that
// Credential Storage's writer must round-trip with.
//
// Parsing rules (the .glassfrogrc structural contract):
//   - blank lines and lines whose first non-whitespace character is '#' are
//     ignored;
//   - every other line is split on its first '=', with key and value trimmed of
//     surrounding whitespace; unknown keys are ignored;
//   - a non-blank, non-comment line without '=' makes the file malformed,
//     returning a *FormatError rather than being silently skipped;
//   - a missing or unreadable file returns a *ReadError (a missing file unwraps
//     to os.ErrNotExist so the caller can treat absence as "skip").
//
// found is false when the file parses but carries no usable token — no token
// key, or a value that is empty/whitespace-only after trimming. That is a normal
// outcome, not an error.
func readCredentialsFile(path string) (token string, found bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, &ReadError{Path: path, Err: err}
	}

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			// A non-blank, non-comment line that is not a key=value pair makes
			// the file unparseable — fail loud rather than guess (CONSTITUTION
			// III). The line content is deliberately omitted from the error.
			return "", false, &FormatError{Path: path}
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		if key == tokenKey {
			token = value
		}
	}

	// A whitespace-only value trims to "", so an empty token reads as "no token
	// present" (found = false) — a blank credential never counts as usable.
	return token, token != "", nil
}
