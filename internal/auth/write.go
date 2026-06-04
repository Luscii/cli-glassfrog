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

// WriteError reports that a credentials file could not be written (an unwritable
// directory, a failed rename, …). It names only the path and wraps the cause —
// never any token value (secret hygiene). On any WriteError the filesystem is
// left unchanged: a store is all-or-nothing.
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
// It parse-validates any existing file through the shared reader first: a
// malformed file (a non-blank, non-comment line without '=') aborts with a
// *FormatError and NO write. On success it rewrites at the line level —
// replacing the value on the existing token= line, or appending a token= line
// if absent — preserving every other line and comment and their order. An
// absent target is created. The write is atomic (a temp file in the same
// directory, chmod 0600 before any token bytes, then rename over the target),
// so a mid-write failure leaves the original (or its absence) intact and the
// secret is never briefly more-permissive than 0600.
//
// The token value never appears in a returned error.
func WriteCredentials(path, token string) error {
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		// Validate the existing file before touching it.
		if _, _, perr := parseCredentials(path, existing); perr != nil {
			return perr // *FormatError → no write
		}
	case errors.Is(err, fs.ErrNotExist):
		existing = nil // absent target: create it
	default:
		return &WriteError{Path: path, cause: err}
	}

	merged := mergeTokenLine(existing, token)
	if err := writeAtomic(path, merged); err != nil {
		return err
	}
	return nil
}

// mergeTokenLine returns the file content with the token entry set to token:
// the existing token= line's value is replaced in place, or a token= line is
// appended when none exists, preserving every other line and comment and their
// order. The result always ends with a trailing newline.
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

	replaced := false
	for i, line := range lines {
		if isTokenLine(line) {
			lines[i] = tokenLine
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines, tokenLine)
	}
	return []byte(strings.Join(lines, "\n") + "\n")
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
