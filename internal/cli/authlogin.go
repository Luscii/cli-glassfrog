package cli

import (
	"path/filepath"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// tokenInputs are the candidate token sources plus interactivity, injected into
// the pure resolution logic so every precedence and interactivity branch is
// testable without touching the real stdin/TTY/env/home (ADR-3). The production
// seam (authlogin_seam.go) binds the real os values to this shape.
type tokenInputs struct {
	// arg is the positional TOKEN; argGiven says whether it was supplied.
	arg      string
	argGiven bool
	// stdin is piped standard input; stdinPiped says whether stdin was a pipe
	// that was read (only meaningful, and only read, when not a TTY).
	stdin      string
	stdinPiped bool
	// env is the GLASSFROG_TOKEN value.
	env string
	// isTTY says whether standard input is a terminal (interactive session).
	isTTY bool
}

// tokenSource names which source the precedence chain selected.
type tokenSource int

const (
	// tokenFromArg .. tokenFromEnv: a source supplied a candidate value.
	tokenFromArg tokenSource = iota
	tokenFromStdin
	tokenFromEnv
	// tokenNeedsPrompt: no arg/stdin/env source, but the session is interactive
	// — the caller must prompt for the token.
	tokenNeedsPrompt
	// tokenNone: no source and not interactive — "no token to store".
	tokenNone
)

// resolveTokenSource picks the winning source by precedence — argument → piped
// stdin → GLASSFROG_TOKEN → interactive prompt — and returns its raw value (the
// caller validates it with usableToken). It is pure: it never reads the
// process, and never invokes the prompt (the prompt is the production seam's
// job; this returns tokenNeedsPrompt instead).
//
// "Present" means: an argument was supplied at all (even blank — a blank
// argument is the operator's input and is rejected downstream rather than
// falling through); piped stdin or the environment carried a non-whitespace
// value (an empty pipe or unset/empty env is absent and falls through, matching
// 005's empty-env rule).
func resolveTokenSource(in tokenInputs) (raw string, source tokenSource) {
	switch {
	case in.argGiven:
		return in.arg, tokenFromArg
	case in.stdinPiped && strings.TrimSpace(in.stdin) != "":
		return in.stdin, tokenFromStdin
	case strings.TrimSpace(in.env) != "":
		return in.env, tokenFromEnv
	case in.isTTY:
		return "", tokenNeedsPrompt
	default:
		return "", tokenNone
	}
}

// usableToken trims surrounding whitespace from a candidate and reports whether
// what remains is a usable token. An empty or whitespace-only value is not
// usable (same rule as 005's reader). A value containing an embedded line break
// (\n or \r) is also rejected: writing it would split into multiple
// .glassfrogrc lines and could inject extra keys, so a multi-line token fails
// the command rather than corrupting the file. The writer is never invoked for
// an unusable token.
func usableToken(raw string) (string, bool) {
	t := strings.TrimSpace(raw)
	if t == "" || strings.ContainsAny(t, "\r\n") {
		return t, false
	}
	return t, true
}

// targetPath selects the credentials file to write: the home-directory file by
// default, or the current-directory file when cwd is set (--cwd). Pure over the
// injected roots.
func targetPath(homeDir, startDir string, cwd bool) string {
	if cwd {
		return filepath.Join(startDir, auth.CredentialsFileName)
	}
	return filepath.Join(homeDir, auth.CredentialsFileName)
}

// homeTokenPath and cwdTokenPath are the two locations Discovery searches and
// Storage can write — used by the interactive existing-token flow to offer a
// location choice.
func homeTokenPath(homeDir string) string { return filepath.Join(homeDir, auth.CredentialsFileName) }
func cwdTokenPath(startDir string) string { return filepath.Join(startDir, auth.CredentialsFileName) }

// guardDecision is the existing-token guard's verdict before a write.
type guardDecision int

const (
	// guardProceed: write the single target (no existing token, or a
	// non-interactive --overwrite).
	guardProceed guardDecision = iota
	// guardBlocked: non-interactive, an existing token, and no --overwrite —
	// a UsageError; the operator must pass --overwrite.
	guardBlocked
	// guardInteractive: interactive with an existing token — confirm the change
	// and offer the location choice.
	guardInteractive
)

// existingTokenGuard decides whether a store may proceed when a token may
// already exist at the target. Pure over the injected facts (ADR-5): no
// existing token → proceed; interactive → confirm/choose; non-interactive →
// blocked unless an overwrite signal is set.
func existingTokenGuard(hasExisting, isTTY, overwrite bool) guardDecision {
	if !hasExisting {
		return guardProceed
	}
	if isTTY {
		return guardInteractive
	}
	if overwrite {
		return guardProceed
	}
	return guardBlocked
}
