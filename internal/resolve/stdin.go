package resolve

import (
	"fmt"
	"io"
	"strings"
)

// maxStdinBytes caps how much piped stdin a stdin source will read. A setting's
// value is tiny; this bounds an accidental (or hostile) large pipe so resolution
// never slurps an arbitrarily large input into memory. The value mirrors 006's
// maxPipedTokenBytes; tuning is deferred.
const maxStdinBytes = 64 << 10 // 64 KiB

// FromStdin yields trimmed piped input when isTTY is false and the content is
// non-empty after trimming. On a terminal (isTTY true) it never reads and does
// not yield — reading a TTY would block waiting on input rather than returning
// already-piped content. A read failure errs, and Resolve surfaces it the same
// way a config-file failure aborts the walk (uniform handling, not a uniform
// type). Provenance.Origin is empty.
//
// The injected read closure owns the read bound; StdinFromOS binds it to a
// maxStdinBytes-capped os.Stdin reader (readBoundedStdin) that errs rather than
// silently truncating an over-long pipe (Constitution VI).
func FromStdin(read func() (string, error), isTTY bool) Source {
	return Source{
		kind: KindStdin,
		eval: func() (string, string, bool, error) {
			if isTTY {
				return "", "", false, nil
			}
			content, err := read()
			if err != nil {
				return "", "", false, err
			}
			trimmed := strings.TrimSpace(content)
			if trimmed == "" {
				return "", "", false, nil
			}
			return trimmed, "", true, nil
		},
	}
}

// readBoundedStdin reads r up to maxStdinBytes and errs if the input exceeds the
// cap, rather than reading an unbounded amount into memory or silently truncating
// (Constitution VI). It is the bounded reader StdinFromOS binds to os.Stdin.
func readBoundedStdin(r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxStdinBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxStdinBytes {
		return "", fmt.Errorf("piped input exceeds the %d-byte limit", maxStdinBytes)
	}
	return string(data), nil
}
