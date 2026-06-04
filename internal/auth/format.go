// Package auth owns the CLI's token concerns: the shared .glassfrogrc
// credentials-file format (read and write share one module so they cannot
// drift), and the writer that persists a token. It registers no command and
// makes no network call (CONSTITUTION V) — the command surface lives in
// internal/cli, the request attachment with the API client.
//
// The file name and environment variable are [ASSUMED] and jointly held with
// Credential Discovery (005); they are centralized here so a change happens
// once. Credential Storage (006) created this module; whichever of 005/006
// lands first owns it, and a write→read round-trip pins the format contract.
package auth

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

const (
	// CredentialsFileName is the credentials file's base name, searched in the
	// current-directory ancestry and the home directory. [ASSUMED], shared with
	// Credential Discovery (005).
	CredentialsFileName = ".glassfrogrc"

	// EnvVarToken is the environment variable carrying a token directly.
	// [ASSUMED], shared with Credential Discovery (005): a runtime override for
	// discovery, a persistable source for storage.
	EnvVarToken = "GLASSFROG_TOKEN"

	// tokenKey is the .glassfrogrc key that carries the credential.
	tokenKey = "token"
)

// FormatError reports that a .glassfrogrc file is malformed — it holds a
// non-blank, non-comment line without an '='. It names only the path (never any
// value): a broken credentials file fails loud rather than being silently
// skipped or overwritten (CONSTITUTION III; secret hygiene). It is a named type
// so callers discriminate a format failure from other errors via errors.As.
type FormatError struct {
	// Path is the offending file.
	Path string
	// Line is the 1-based number of the malformed line.
	Line int
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("credentials file %s is malformed: line %d is neither blank, a comment, nor a key=value pair", e.Path, e.Line)
}

// ReadCredentialsFile reads path and returns the token it holds. It is the
// shared reader both Discovery (005) and Storage (006) depend on, and the
// round-trip target for the writer.
//
// Outcomes:
//   - file absent → ("", false, nil): a missing file is not an error here; the
//     caller decides what absence means.
//   - file present, usable token → (token, true, nil), trimmed of surrounding
//     whitespace.
//   - file present, no usable token (no token key, or a blank/whitespace-only
//     value) → ("", false, nil).
//   - file present, malformed → ("", false, *FormatError) naming the path.
//   - file present, unreadable (e.g. permission denied) → ("", false, err)
//     wrapping the read failure (never a token value).
func ReadCredentialsFile(path string) (token string, hasToken bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("reading credentials file %s: %w", path, err)
	}
	return parseCredentials(path, data)
}

// parseCredentials applies the shared .glassfrogrc line rules to data and
// returns the token. A non-blank, non-comment line without '=' makes the file
// malformed (a *FormatError). The last token= line wins (single-token contract,
// forward-compatible with unknown keys).
func parseCredentials(path string, data []byte) (token string, hasToken bool, err error) {
	text := strings.TrimSuffix(string(data), "\n")
	if text == "" && len(data) == 0 {
		return "", false, nil
	}
	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return "", false, &FormatError{Path: path, Line: i + 1}
		}
		key := strings.TrimSpace(line[:eq])
		if key != tokenKey {
			continue // unknown key: ignored (forward-compatible)
		}
		value := strings.TrimSpace(line[eq+1:])
		if value == "" {
			// A blank/whitespace-only token value is treated as no token present;
			// clear any earlier token so a later empty entry does not falsely
			// report a credential.
			token, hasToken = "", false
			continue
		}
		token, hasToken = value, true
	}
	return token, hasToken, nil
}

// isTokenLine reports whether line is the credential's token= entry (used by the
// line-preserving merge to find the line whose value to replace). Blank, comment
// and non-'=' lines are not token lines; malformed lines are rejected upstream
// by parseCredentials before the merge runs.
func isTokenLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return false
	}
	return strings.TrimSpace(line[:eq]) == tokenKey
}
