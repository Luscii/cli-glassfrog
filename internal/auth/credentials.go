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

	// baseURLKey is the .glassfrogrc key carrying the Glassfrog base URL, read by
	// Base URL Resolution (008) through the same shared parser. It is a
	// .glassfrogrc format detail and lives here beside tokenKey; the precedence,
	// validation, env var, and flag that surround it are connection-configuration
	// concerns owned by internal/apiclient. [ASSUMED] pending reconciliation with
	// Credential Storage (006).
	baseURLKey = "base_url"
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
	return parseCredentials(path, data)
}

// readBaseURLFile reads the .glassfrogrc at path and returns the value under the
// "base_url" key, through the same shared parser readCredentialsFile uses (no
// second reader — Base URL Resolution 008, ADR-3). The token is never read out
// of the parsed fields on this path, so it can never appear in the returned
// value or in any error (secret hygiene). It performs no network call.
//
// found is false when the file parses but carries no usable base_url (no key, or
// an empty/whitespace-only value). A missing or unreadable file returns a
// *ReadError (a missing file unwraps to os.ErrNotExist so the caller can treat
// absence as "skip"); a malformed file returns a *FormatError.
func readBaseURLFile(path string) (baseURL string, found bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, &ReadError{Path: path, Err: err}
	}
	f, err := parseFields(path, data)
	if err != nil {
		return "", false, err
	}
	return f.baseURL, f.baseURLFound, nil
}

// fileFields are the recognized .glassfrogrc keys captured from one parse pass.
// A value that is empty or whitespace-only after trimming is treated as absent
// (its found flag is false), so a blank entry never counts as usable —
// uniformly across keys (the shared "usable value" rule, LEARNINGS).
type fileFields struct {
	token        string
	tokenFound   bool
	baseURL      string
	baseURLFound bool
}

// parseCredentials applies the .glassfrogrc structural contract to data already
// read from path, returning the token under the "token" key. It is a thin view
// over the shared parseFields step (so the token reader exposes only the token,
// never the base_url): readCredentialsFile feeds it bytes from disk, and
// Credential Storage's writer feeds it the same bytes it is about to merge — so
// validation and merge operate on one snapshot and cannot diverge under a
// concurrent edit (no re-read TOCTOU). path is used only for error messages.
//
// It performs no I/O, so the only error it can return is a *FormatError (a
// non-blank, non-comment line without '='); a *ReadError is the reader's concern.
func parseCredentials(path string, data []byte) (token string, found bool, err error) {
	f, err := parseFields(path, data)
	if err != nil {
		return "", false, err
	}
	return f.token, f.tokenFound, nil
}

// parseFields is the one shared parse step over a .glassfrogrc snapshot. It
// applies the structural contract once and captures every recognized key, so the
// token reader, the base_url reader (008), and the writer's pre-merge validation
// all agree on the same parse — no second reader drifts from this one (ADR-3).
//
// Parsing rules (the .glassfrogrc structural contract):
//   - blank lines and lines whose first non-whitespace character is '#' are
//     ignored;
//   - every other line is split on its first '=', with key and value trimmed of
//     surrounding whitespace; unknown keys are ignored;
//   - a non-blank, non-comment line without '=' makes the file malformed,
//     returning a *FormatError rather than being silently skipped.
//
// Each key is last-occurrence-wins, matching the reader's long-standing token
// behavior. A value that trims to "" reads as "key not present" (found = false).
// parseFields does no I/O; the only error it returns is a *FormatError.
func parseFields(path string, data []byte) (fileFields, error) {
	var f fileFields
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
			return fileFields{}, &FormatError{Path: path}
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		switch key {
		case tokenKey:
			f.token = value
		case baseURLKey:
			f.baseURL = value
		}
	}

	// A whitespace-only value trims to "", so an empty entry reads as "not
	// present" (found = false) — a blank value never counts as usable.
	f.tokenFound = f.token != ""
	f.baseURLFound = f.baseURL != ""
	return f, nil
}
