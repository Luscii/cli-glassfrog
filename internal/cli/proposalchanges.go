package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// maxChangesBytes caps how much piped stdin the `--changes stdin` path reads. A
// governance change set can be larger than a token (006's 64 KiB token cap), but is
// still bounded so the command never slurps an arbitrarily large pipe into memory.
// The cap and the overflow message are changes-specific so a large change set never
// surfaces the token-worded error (mirrors maxPipedTemplateBytes, 035).
const maxChangesBytes = 1 << 20 // 1 MiB

// resolveChangesSource classifies the --changes flag value into the raw JSON bytes of
// the change set, from one of three sources (plan ADR-2), and is PURE over its
// injected stat/readFile/isTTY/stdin so every branch is offline-testable:
//
//   - a trimmed value equal (case-insensitive) to the reserved keyword "stdin" reads
//     the array from piped standard input, via the 006 bounded reader, guarded by the
//     035 TTY fail-fast (a terminal stdin) and the empty-pipe fail-fast;
//   - else a value whose stat reports an EXISTING REGULAR FILE (not a directory) is
//     read from disk via readFile;
//   - else the value's own bytes are returned as inline JSON.
//
// A file literally named `stdin` is reachable as `./stdin` (the 035 reserved-name
// escape). Every read/stat/empty-pipe failure returns an error NAMING the source,
// which the command reports as a UsageError(2) before any request.
func resolveChangesSource(value string, stat func(string) (os.FileInfo, error), readFile func(string) ([]byte, error), isTTY bool, stdin io.Reader) ([]byte, error) {
	if strings.EqualFold(strings.TrimSpace(value), "stdin") {
		if isTTY {
			return nil, errors.New("--changes stdin requires a change set piped on standard input, but standard input is a terminal — pipe a JSON array, e.g. `cat changes.json | glassfrog proposal create … --changes stdin`")
		}
		text, err := readBoundedStdinN(stdin, maxChangesBytes, "changes")
		if err != nil {
			return nil, fmt.Errorf("could not read the change set from stdin: %w", err)
		}
		if strings.TrimSpace(text) == "" {
			return nil, errors.New("--changes stdin: no change set was piped to standard input (the pipe was empty)")
		}
		return []byte(text), nil
	}

	// A value that stats to an existing regular file is read from disk; a directory or
	// any other non-regular entry is not a change set, so it is rejected rather than
	// read as JSON (the existing-regular-file shape). stat follows symlinks, so a
	// symlink to a regular file is read like any other file — this is not a
	// symlink/path-traversal guard, only a "don't read a directory as a change set" one.
	info, statErr := stat(value)
	if statErr == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("--changes %q is not a regular file (a change set must be inline JSON, a file path, or the reserved keyword stdin)", value)
		}
		b, rerr := readFile(value)
		if rerr != nil {
			return nil, fmt.Errorf("could not read the change set file %q: %w", value, rerr)
		}
		return b, nil
	}

	// A permission error on stat means the value names a path the operator likely meant
	// but the process cannot reach (e.g. a parent directory lacks traversal permission).
	// Surface it as a named source error rather than silently treating the path as
	// inline JSON, which would mislead with a "must be a JSON array" parse error. Any
	// OTHER stat error — the path does not exist (the common inline case), or the value
	// is too long to be a path name (a large inline JSON array overflows NAME_MAX and
	// stats as ENAMETOOLONG) — falls through to the inline branch.
	if errors.Is(statErr, fs.ErrPermission) {
		return nil, fmt.Errorf("could not access the change set source %q: %w", value, statErr)
	}

	// Anything else is the inline JSON array itself.
	return []byte(value), nil
}

// validateChanges applies the type floor (plan ADR-3) to the raw change-set bytes and
// returns the verbatim change slice on success. It is PURE over the bytes: it
// unmarshals into []json.RawMessage (rejecting non-JSON and a non-array with one
// message), rejects an empty array, then probes EACH element by decoding into a
// minimal {type string} struct — rejecting an element that is not a JSON object or
// that lacks a non-empty `type` (the one key the schema requires). It reads ONLY each
// element's type; every other command-specific key rides through untouched in the
// returned slice. Every rejection is a usage-class error the command surfaces as a
// UsageError(2) before any request — a clearer local error than a server 422.
func validateChanges(raw []byte) ([]json.RawMessage, error) {
	var changes []json.RawMessage
	if err := json.Unmarshal(raw, &changes); err != nil {
		return nil, errors.New("--changes must be a JSON array of change objects")
	}
	// JSON null unmarshals into a nil slice WITHOUT error — but null is not an array, so
	// reject it with the array message rather than the empty-array one. An empty array
	// (`[]`) unmarshals into a non-nil empty slice, so a nil slice here means null.
	if changes == nil {
		return nil, errors.New("--changes must be a JSON array of change objects")
	}
	if len(changes) == 0 {
		return nil, errors.New("at least one change is required")
	}
	for _, elem := range changes {
		var probe struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(elem, &probe); err != nil {
			// A non-object element (a number, string, array, …) cannot decode into the
			// struct — it fails the "must carry a type" floor.
			return nil, errors.New(`every change must be an object carrying a "type"`)
		}
		if strings.TrimSpace(probe.Type) == "" {
			return nil, errors.New(`every change must carry a "type"`)
		}
	}
	return changes, nil
}
