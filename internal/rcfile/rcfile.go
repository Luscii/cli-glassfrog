// Package rcfile reads the project's .glassfrogrc settings file: a small
// key=value file located by a nearest-wins walk up from the working directory and
// finally the home directory. It is deliberately generic — it knows the file
// format and the search order, not what any particular key means. The token
// (internal/auth) and the base URL (internal/apiclient) are two consumers; future
// settings are more.
//
// Centralizing the read, the parse, and the walk here means every setting — for
// any key, present or future — is retrieved through the same code path, so no two
// consumers can drift in how they read the file (the failure mode that motivated
// extracting this package out of internal/auth).
//
// rcfile performs no network I/O and registers no command. Its errors name only
// the file path, never any value, so a consumer holding a secret (the token) can
// surface a broken-file error without leaking it.
package rcfile

import (
	"fmt"
	"os"
	"strings"
)

// FileName is the base name of the settings file.
const FileName = ".glassfrogrc"

// FormatError reports that a candidate .glassfrogrc exists and was read but could
// not be parsed — it held a non-blank, non-comment line with no '='. It names
// only the path, never the file's contents, so a broken file fails loud without
// leaking any value (e.g. a token on another line).
type FormatError struct {
	Path string
}

func (e *FormatError) Error() string {
	return fmt.Sprintf("config file %s is malformed: a non-comment line is not a key=value pair", e.Path)
}

// ReadError reports that a candidate .glassfrogrc could not be read (e.g.
// permission denied, or — distinguishable via errors.Is(err, os.ErrNotExist) —
// simply absent). It wraps the underlying filesystem error, which names only the
// path, never any file contents.
type ReadError struct {
	Path string
	Err  error
}

func (e *ReadError) Error() string {
	return fmt.Sprintf("config file %s could not be read: %v", e.Path, e.Err)
}

func (e *ReadError) Unwrap() error { return e.Err }

// Settings holds the key=value entries parsed from one .glassfrogrc snapshot.
// Keys map to their trimmed values; a key that appeared more than once holds the
// last occurrence's value (last-wins).
type Settings map[string]string

// Value returns the setting under key and whether it is usable. A value that is
// empty or whitespace-only is treated as absent (found = false) — the uniform
// "usable value" rule applied to every key; a missing key reads the same as a
// blank one.
func (s Settings) Value(key string) (value string, found bool) {
	value = s[key]
	return value, value != ""
}

// Parse applies the .glassfrogrc structural contract to data already read from
// path (path is used only for error messages, never the contents). It is the one
// shared parse step: Read feeds it bytes from disk, and a writer can feed it the
// exact bytes it is about to merge — so validation and merge operate on one
// snapshot and cannot diverge under a concurrent edit (no re-read TOCTOU).
//
// Parsing rules:
//   - blank lines and lines whose first non-whitespace character is '#' are
//     ignored;
//   - every other line is split on its first '=', with key and value trimmed of
//     surrounding whitespace; a value may itself contain '=';
//   - a non-blank, non-comment line without '=' makes the file malformed,
//     returning a *FormatError rather than being silently skipped.
//
// Keys are last-occurrence-wins. Parse does no I/O, so the only error it can
// return is a *FormatError.
func Parse(path string, data []byte) (Settings, error) {
	s := Settings{}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			// A non-blank, non-comment line that is not a key=value pair makes the
			// file unparseable — fail loud rather than guess (CONSTITUTION III).
			// The line content is deliberately omitted from the error.
			return nil, &FormatError{Path: path}
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		s[key] = value
	}
	return s, nil
}

// Read reads and parses the .glassfrogrc at path. A missing or unreadable file
// returns a *ReadError (a missing file unwraps to os.ErrNotExist, so callers can
// treat absence as "skip"); a malformed file returns a *FormatError.
func Read(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ReadError{Path: path, Err: err}
	}
	return Parse(path, data)
}

// ReadValue reads the .glassfrogrc at path and returns the value for key alone —
// never any other key's value, so a caller asking for one setting never comes
// into possession of another (a base-URL read never sees the token). found is
// false when the file parses but carries no usable value for key. Errors are
// Read's: a *ReadError (missing/unreadable) or a *FormatError (malformed).
func ReadValue(path, key string) (value string, found bool, err error) {
	s, err := Read(path)
	if err != nil {
		return "", false, err
	}
	value, found = s.Value(key)
	return value, found, nil
}
