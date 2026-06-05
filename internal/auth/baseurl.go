package auth

import (
	"errors"
	"os"
	"path/filepath"
)

// ResolveBaseURLFile walks the .glassfrogrc precedence chain — the nearest file
// up the directory tree from startDir, then the home file — and returns the
// base_url value from the first file that carries a usable one, along with the
// path it was read from. It is the file rung of Base URL Resolution (008); the
// flag, the GLASSFROG_BASE_URL environment variable, URL validation, and the
// built-in default are connection-configuration concerns owned by
// internal/apiclient, which injects startDir and homeDir.
//
// It reuses Credential Discovery's (005) candidateDirs walk and the one shared
// parser — no second .glassfrogrc reader (ADR-3). The token is never returned
// through this path and never appears in any error (secret hygiene). It makes no
// network call.
//
// Skip / fail-loud semantics mirror the token resolver:
//   - a file that exists and parses but holds no usable base_url is skipped (the
//     walk continues to the next location);
//   - a missing file at a candidate location is skipped (unwraps to
//     os.ErrNotExist);
//   - an unreadable file (a *ReadError that is not os.ErrNotExist) or an
//     unparseable file (a *FormatError) fails loud naming the path, with no
//     fall-through to a lower-precedence source.
//
// No usable base_url anywhere is ("", "", false, nil) — a normal outcome, not an
// error; the caller (internal/apiclient) supplies the built-in default.
func ResolveBaseURLFile(startDir, homeDir string) (value string, path string, found bool, err error) {
	for _, dir := range candidateDirs(startDir, homeDir) {
		candidate := filepath.Join(dir, credentialsFileName)
		v, ok, readErr := readBaseURLFile(candidate)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				continue // missing at this location → try the next
			}
			return "", "", false, readErr // unreadable/unparseable → fail loud
		}
		if !ok {
			continue // exists but holds no usable base_url → skip to the next
		}
		return v, candidate, true, nil
	}
	return "", "", false, nil
}
