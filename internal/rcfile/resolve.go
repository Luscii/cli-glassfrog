package rcfile

import (
	"errors"
	"os"
	"path/filepath"
)

// Resolve walks the .glassfrogrc precedence chain — the nearest file up the
// directory tree from startDir, then the home file — and returns the value for
// key from the first file that carries a usable one, along with that file's path.
// It is the one nearest-wins walk shared by every consumer (token, base URL,
// future settings), so they cannot drift in search order or skip/fail-loud
// semantics.
//
//   - a file that exists and parses but has no usable value for key is skipped
//     (the search continues, so a key-less nearest file does not shadow a lower
//     one that has the key);
//   - a missing file at a candidate location is skipped (os.ErrNotExist);
//   - an unreadable (*ReadError) or unparseable (*FormatError) file fails loud
//     naming the path, with no fall-through to a lower-precedence location.
//
// No file with a usable value anywhere is ("", "", false, nil) — a normal
// outcome, not an error. Only the requested key's value is ever returned (secret
// hygiene: a base-URL walk never returns the token). Resolve makes no network
// call.
func Resolve(startDir, homeDir, key string) (value string, path string, found bool, err error) {
	for _, dir := range candidateDirs(startDir, homeDir) {
		candidate := filepath.Join(dir, FileName)
		v, ok, readErr := ReadValue(candidate, key)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				continue // missing at this location → try the next
			}
			return "", "", false, readErr // unreadable/unparseable → fail loud
		}
		if !ok {
			continue // exists but holds no usable value for key → skip to the next
		}
		return v, candidate, true, nil
	}
	return "", "", false, nil
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
