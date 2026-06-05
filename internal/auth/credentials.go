// Package auth resolves the Glassfrog API token the CLI operates as. It answers
// one question — "what token are we acting as, right now, in this directory?" —
// by consulting the GLASSFROG_TOKEN environment variable, then a nearest-wins
// walk-up of .glassfrogrc files, then the home-directory file (Credential
// Discovery, 005). It registers no command and prints nothing; Request
// Authentication (007) consumes the Resolution it returns, and Credential
// Storage (006) adds the writer beside the shared file-format reader so the read
// and write sides cannot drift.
//
// The .glassfrogrc file format, parse, and nearest-wins walk are owned by the
// generic internal/rcfile package — token is just one of its keys (base URL is
// another). auth is a consumer: it reads the "token" key and adds the secret
// hygiene the token demands.
//
// Secret hygiene is a package-wide invariant: the token value never appears in
// any error message, log line, or other output. Errors carry only the offending
// file path.
package auth

import "github.com/Luscii/cli-glassfrog/internal/rcfile"

// credentialsFileName re-exports rcfile.FileName: auth reads and writes the same
// .glassfrogrc the rcfile package defines. envTokenVar and tokenKey are auth's
// own (token-specific) constants — the env variable and the file key it reads.
// All three are [ASSUMED], jointly held with Credential Storage (006).
const (
	credentialsFileName = rcfile.FileName
	envTokenVar         = "GLASSFROG_TOKEN"

	// tokenKey is the .glassfrogrc key carrying the credential.
	tokenKey = "token"
)

// FormatError and ReadError are the .glassfrogrc file errors. They are owned by
// rcfile (the generic file concern) and re-exported here as aliases so auth's
// credential-domain consumers — Credential Storage's command layer, which
// discriminates a malformed file from a write failure — keep referring to them
// as auth.FormatError / auth.ReadError. The aliases point at the one canonical
// type, so errors.As across the two package names is the same check.
type (
	// FormatError reports a malformed .glassfrogrc (a non-comment line without
	// '='). It names only the path, never the token.
	FormatError = rcfile.FormatError
	// ReadError reports a .glassfrogrc that could not be read (a missing file
	// unwraps to os.ErrNotExist). It names only the path, never the token.
	ReadError = rcfile.ReadError
)
