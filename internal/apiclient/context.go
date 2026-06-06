package apiclient

import (
	"fmt"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// ConnectionContext is the single bundle pairing the resolved endpoint with the
// resolved identity — the connection context every request hangs off. It carries
// 008's base-URL outcome (Value/Source/Path or a typed BaseURLErr) and 005's
// credential outcome (auth.Resolution, with Source == auth.SourceNone meaning
// absent, or a typed CredErr). Exactly one of {BaseURL valid, BaseURLErr set}
// holds; CredErr is nil on success or absence.
//
// It carries a secret: Cred.Token. Read the token through the field, never
// through formatting — String() (and therefore %v/%+v/%s) redacts it. Readiness
// (Complete/Problems) is derived from the carried fields, never stored: a missing
// or broken part makes the context incomplete and is named in Problems(), but is
// never a reason to refuse a call, fabricate a value, or pick an exit code — that
// stays with Request Authentication (007) and the consuming command.
type ConnectionContext struct {
	// BaseURL is 008's resolved base-URL outcome; meaningful when BaseURLErr is nil.
	BaseURL BaseURL
	// BaseURLErr is 008's typed base-URL failure carried verbatim — a *BaseURLError
	// (names the source) or an internal/rcfile read/format error (names the file).
	// nil on success.
	BaseURLErr error
	// Cred is 005's resolved credential outcome. Source == auth.SourceNone means no
	// credential was found (absence, not an error). Token is the secret.
	Cred auth.Resolution
	// CredErr is 005's typed credential failure carried verbatim — an internal/rcfile
	// read/format error naming the file (path-only). nil on success or absence.
	CredErr error
}

// Complete reports whether the context is ready for a request: a usable base URL
// (no base-URL error) and a present token (Cred.Source != auth.SourceNone) and no
// credential error. It is a derived view, not stored.
func (c ConnectionContext) Complete() bool {
	return c.BaseURLErr == nil && c.CredErr == nil && c.Cred.Source != auth.SourceNone
}

// Problems returns safe-to-display labels for each missing or errored part, in a
// stable order — base URL first, then credential. It is empty when Complete().
// Every entry is built only from safe sources: the base-URL error's message
// (source label or file path — never a value) and the credential Source/Path
// (path-only by the 005/007 contract) or a fixed "no credentials found" phrase.
// It never contains the token.
func (c ConnectionContext) Problems() []string {
	var problems []string

	if c.BaseURLErr != nil {
		problems = append(problems, "base URL: "+c.BaseURLErr.Error())
	}

	switch {
	case c.CredErr != nil:
		problems = append(problems, "credential: "+c.CredErr.Error())
	case c.Cred.Source == auth.SourceNone:
		problems = append(problems, "credential: no credentials found")
	}

	return problems
}

// String renders the context with the token redacted, so the common formatting
// verbs (%v, %s, %+v) cannot leak the secret. It shows the base-URL source (or
// its error label), the credential Source/Path, and the readiness, reporting the
// token as present-but-hidden or absent — never verbatim. Value receiver so a
// ConnectionContext value (not just a pointer) redacts.
func (c ConnectionContext) String() string {
	var baseURL string
	if c.BaseURLErr != nil {
		baseURL = "error (" + c.BaseURLErr.Error() + ")"
	} else {
		baseURL = c.BaseURL.Source.String()
		if c.BaseURL.Path != "" {
			baseURL += " (" + c.BaseURL.Path + ")"
		}
	}

	var credential string
	switch {
	case c.CredErr != nil:
		credential = "error (" + c.CredErr.Error() + ")"
	case c.Cred.Source == auth.SourceNone:
		credential = "none"
	default:
		credential = c.Cred.Source.String()
		if c.Cred.Path != "" {
			credential += " (" + c.Cred.Path + ")"
		}
	}

	token := "<none>"
	if c.Cred.Token != "" {
		token = "<redacted>"
	}

	readiness := "incomplete"
	if c.Complete() {
		readiness = "complete"
	} else if probs := c.Problems(); len(probs) > 0 {
		readiness = "incomplete: [" + strings.Join(probs, "; ") + "]"
	}

	return fmt.Sprintf("ConnectionContext{baseURL: %s, credential: %s, token: %s, %s}", baseURL, credential, token, readiness)
}
