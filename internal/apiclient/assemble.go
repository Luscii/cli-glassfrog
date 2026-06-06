package apiclient

import (
	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// Assemble is the transparent aggregator at the heart of Connection Context
// Assembly: it pairs 008's base-URL outcome with 005's credential outcome into a
// single ConnectionContext. It calls BOTH resolvers exactly once and does not
// short-circuit when the first returns an error (carry-both) — a base-URL failure
// must not skip the credential walk, or an also-absent credential would be lost.
// Each (value, error) is packed verbatim into the matching context fields.
//
// Assembly always yields a context: it has no error return, never refuses a call,
// transforms nothing, makes no network call or write, and decides no exit code.
// Readiness (Complete/Problems) is a derived view the consumer reads; the
// refuse-to-call fail-safe stays in Request Authentication (007).
//
// Precondition: both resolvers are required and must be non-nil. A nil resolver is
// a wiring bug and panics (fail-fast — no nil-default, per DECISIONS/PR #20).
func Assemble(
	resolveBaseURL func() (BaseURL, error),
	resolveCred func() (auth.Resolution, error),
) ConnectionContext {
	if resolveBaseURL == nil {
		panic("apiclient.Assemble: resolveBaseURL must not be nil")
	}
	if resolveCred == nil {
		panic("apiclient.Assemble: resolveCred must not be nil")
	}

	baseURL, baseURLErr := resolveBaseURL()
	cred, credErr := resolveCred()

	return ConnectionContext{
		BaseURL:    baseURL,
		BaseURLErr: baseURLErr,
		Cred:       cred,
		CredErr:    credErr,
	}
}

// AssembleFromOS is the thin production seam over Assemble: it binds the real
// resolvers — ResolveBaseURLFromOS(flagValue) for the base URL and auth.Resolve
// for the credential — and delegates. The --base-url flag value is an input now;
// its cobra registration is deferred to the consuming command.
//
// Intended to be called ONCE per invocation: assembly is the single resolution
// point for an invocation (both walks happen here), and the command layer threads
// the returned context to every request. Calling it repeatedly would re-resolve;
// Request Authentication's own sync.Once backstops the request layer.
func AssembleFromOS(flagValue string) ConnectionContext {
	return Assemble(
		func() (BaseURL, error) { return ResolveBaseURLFromOS(flagValue) },
		auth.Resolve,
	)
}
