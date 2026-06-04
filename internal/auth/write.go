package auth

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// credentialsFileMode is the owner-read/write-only permission a credentials
// file is written with. [ASSUMED], POSIX permission bits — best-effort on
// platforms that lack them. The secret is never present in a more-permissive
// intermediate file (see writeAtomic).
const credentialsFileMode fs.FileMode = 0o600

// Exported surface for the command layer (internal/cli, Credential Storage
// 006). The shared file name, env var, and single-path reader are owned by the
// format module (credentials.go, Credential Discovery 005) and re-exported here
// so the auth login command can build target paths, read GLASSFROG_TOKEN as a
// persistable source, run the existing-token guard, and round-trip a written
// token through the very reader Discovery uses. Single source of truth stays in
// credentials.go; these only widen its visibility.
const (
	// CredentialsFileName is the credentials file's base name (`.glassfrogrc`).
	CredentialsFileName = credentialsFileName
	// EnvVarToken is the token environment variable (`GLASSFROG_TOKEN`).
	EnvVarToken = envTokenVar
)

// ReadCredentialsFile reads the token at path through the shared format reader
// (the round-trip target for the writer). It returns (token, found, err) with
// the same contract as the internal reader: a missing or unreadable file
// returns a *ReadError (a missing file unwraps to os.ErrNotExist), a malformed
// file a *FormatError, and a parsed-but-tokenless file (token, false, nil).
func ReadCredentialsFile(path string) (token string, found bool, err error) {
	return readCredentialsFile(path)
}

// ErrTokenNotSingleLine reports that a token passed to WriteCredentials contained
// a carriage return or newline. A multi-line token would inject extra lines (and
// thus extra keys) into the line-oriented .glassfrogrc, so the writer rejects it
// before any read or write — the file is left untouched. The error never includes
// the token value (secret hygiene). Callers that accept tokens from external
// sources (stdin/env/prompt) should still validate upstream; this is the shared
// writer API's own defensive floor.
var ErrTokenNotSingleLine = errors.New("token must be a single line (contains a carriage return or newline)")

// WriteError reports that a credentials file could not be written (an unwritable
// directory, a failed rename, …). It names only the path and wraps the cause —
// never any token value (secret hygiene). On any WriteError the target path is
// left unchanged: the atomic temp+rename guarantees no partial file at that
// path. (Distinct from *FormatError — a malformed existing file — and from
// *ReadError — an existing file that could not be read during the guard.)
type WriteError struct {
	// Path is the target credentials file.
	Path string
	// cause is the underlying I/O failure.
	cause error
}

func (e *WriteError) Error() string {
	return fmt.Sprintf("could not write credentials to %s: %v", e.Path, e.cause)
}

func (e *WriteError) Unwrap() error { return e.cause }

// WriteCredentials persists token into the .glassfrogrc file at path.
//
// It parse-validates any existing file through the shared parser first, on the
// exact bytes it then merges (one snapshot, no re-read race): a malformed file
// (a non-blank, non-comment line without '=') aborts with a *FormatError and NO
// write. On success it rewrites at the line level —
// replacing the value on the existing token= line, or appending a token= line
// if absent — preserving every other line and comment and their order. An
// absent target is created. The write is atomic (a temp file in the same
// directory, chmod 0600 before any token bytes, then rename over the target),
// so a mid-write failure leaves the original (or its absence) intact and the
// secret is never briefly more-permissive than 0600.
//
// A token containing a carriage return or newline is rejected up front with
// ErrTokenNotSingleLine and NO write — a multi-line token would inject extra
// .glassfrogrc lines/keys.
//
// The token value never appears in a returned error.
func WriteCredentials(path, token string) error {
	// Reject a multi-line token before touching the filesystem: it would inject
	// extra .glassfrogrc lines/keys. Fail without writing (ErrTokenNotSingleLine).
	if strings.ContainsAny(token, "\r\n") {
		return ErrTokenNotSingleLine
	}

	existing, readErr := os.ReadFile(path)
	switch {
	case readErr == nil:
		// Validate the exact bytes we are about to merge through the shared
		// parser, so the read and write sides cannot drift and validation can't
		// race a concurrent edit (no re-read between check and merge); a
		// malformed file aborts with no write. parseCredentials does no I/O, so
		// the only error it yields is a *FormatError.
		if _, _, perr := parseCredentials(path, existing); perr != nil {
			return perr // *FormatError → no write
		}
	case errors.Is(readErr, fs.ErrNotExist):
		existing = nil // absent target: create it
	default:
		return &WriteError{Path: path, cause: readErr}
	}

	merged := mergeTokenLine(existing, token)
	return writeAtomic(path, merged)
}

// mergeTokenLine returns the file content with the token entry set to token:
// the first existing token= line's value is replaced in place (or a token= line
// is appended when none exists), and any further token= lines are dropped, so
// exactly one token entry remains. Collapsing duplicates matters because the
// shared reader is last-token-wins: leaving a later stale token= line would keep
// the old value effective after a write and leave the old secret in the file.
// Every non-token line and comment is preserved in order; the result always ends
// with a trailing newline.
func mergeTokenLine(existing []byte, token string) []byte {
	tokenLine := tokenKey + "=" + token
	if len(existing) == 0 {
		return []byte(tokenLine + "\n")
	}

	text := string(existing)
	hadTrailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(text, "\n")
	if hadTrailingNewline {
		// Drop the empty segment after the final newline so a join does not
		// reintroduce it as a blank line; the trailing newline is re-added below.
		lines = lines[:len(lines)-1]
	}

	out := make([]string, 0, len(lines)+1)
	replaced := false
	for _, line := range lines {
		if isTokenLine(line) {
			if !replaced {
				out = append(out, tokenLine) // keep the first token entry's position
				replaced = true
			}
			continue // drop this (and any later) duplicate token line
		}
		out = append(out, line)
	}
	if !replaced {
		out = append(out, tokenLine)
	}
	return []byte(strings.Join(out, "\n") + "\n")
}

// isTokenLine reports whether line is the credential's token= entry (used by the
// line-preserving merge to find the line whose value to replace). Blank, comment
// and non-'=' lines are not token lines; malformed lines are rejected upstream
// by the shared reader before the merge runs.
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

// writeAtomic writes data to path atomically: a temp file in the same directory
// (so rename stays on one filesystem), chmod 0600 before the bytes are written,
// then rename over the target. Any failure removes the temp file and returns a
// *WriteError naming the path — leaving the original (or absence) untouched.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".glassfrogrc-*.tmp")
	if err != nil {
		return &WriteError{Path: path, cause: err}
	}
	tmpName := tmp.Name()

	// fail removes the temp file and wraps cause; used on every failure after
	// the temp file exists so no partial file is ever left behind.
	fail := func(cause error) error {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return &WriteError{Path: path, cause: cause}
	}

	// Restrict permissions before any token bytes are written, so the secret is
	// never briefly present in a more-permissive file. (CreateTemp already uses
	// 0600; the explicit Chmod documents and pins the guarantee.)
	if err := tmp.Chmod(credentialsFileMode); err != nil {
		return fail(err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return &WriteError{Path: path, cause: err}
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return &WriteError{Path: path, cause: err}
	}
	return nil
}
