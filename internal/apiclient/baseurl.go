package apiclient

import (
	"fmt"
	"net/url"
	"os"

	"github.com/Luscii/cli-glassfrog/internal/resolve"
)

// Connection-configuration constants for base URL resolution, centralized here
// as the single source of truth — including the .glassfrogrc base_url file key
// (apiclient owns the base-URL key; internal/auth owns the token key; the generic
// file mechanics live in internal/rcfile).
//
// The default URL value is fixed; the env var and flag names are [ASSUMED] CLI
// conventions (not in the API spec), pending reconciliation with Credential
// Storage (006).
const (
	// EnvVarBaseURL is the environment variable carrying the base URL (precedence
	// rung 2, below the flag and above the file). [ASSUMED]
	EnvVarBaseURL = "GLASSFROG_BASE_URL"

	// FlagBaseURL is the command flag name carrying the base URL (precedence rung
	// 1, highest). Its cobra registration is deferred to the consuming command;
	// the resolver accepts the flag value as an input now. [ASSUMED]
	FlagBaseURL = "base-url"

	// DefaultBaseURL is the built-in default Glassfrog API base URL (the backstop,
	// precedence rung 4). The /api/v5 path is from spec/glassfrog-api-v5.yaml's
	// servers block (a host-less relative URL); the host https://app.glassfrog.com
	// is the live API host, confirmed by the spec's own source
	// (https://app.glassfrog.com/api/v5/docs/spec.yaml) and the published docs.
	// The earlier https://glassfrog.com guess (inferred from info.contact.url —
	// risk H-1) was wrong: that host redirects to the www marketing site and
	// returns 404 for /api/v5/* paths. It is valid by construction and never
	// re-validated, so the chain always yields a value.
	DefaultBaseURL = "https://app.glassfrog.com/api/v5"

	// baseURLKey is the .glassfrogrc key carrying the base URL, read through the
	// generic rcfile walk. It lives here, with the other base-URL connection
	// constants, rather than in internal/auth — auth owns the token key, apiclient
	// owns the base-URL key. [ASSUMED], jointly held with Credential Storage (006).
	baseURLKey = "base_url"
)

// BaseURLSource names where a resolved base URL came from. Unlike auth.Source
// there is no None member — base-URL resolution always yields a value (the
// default backstops the chain), so a bare BaseURL{} (Source SourceFlag, the zero
// value) never means "nothing found".
type BaseURLSource int

const (
	// SourceFlag is the --base-url command flag (highest precedence).
	SourceFlag BaseURLSource = iota
	// SourceEnvironment is the GLASSFROG_BASE_URL environment variable.
	SourceEnvironment
	// SourceFile is a .glassfrogrc base_url entry (Path names which file).
	SourceFile
	// SourceDefault is the built-in default (the backstop).
	SourceDefault
)

func (s BaseURLSource) String() string {
	switch s {
	case SourceFlag:
		return "flag"
	case SourceEnvironment:
		return "environment"
	case SourceFile:
		return "file"
	case SourceDefault:
		return "default"
	default:
		return "unknown"
	}
}

// BaseURL is Base URL Resolution's code-free output, consumed by the deferred
// connection-context half (which combines it with 005's token to build the base
// http.Client/transport that 007's AuthTransport wraps). Value, Source, and Path
// are all safe to display — the base URL is not a secret.
type BaseURL struct {
	// Value is the resolved base URL, always set on the success path. It is not
	// normalized — no trailing-slash rewrite, no scheme coercion. Flag and env
	// values are used exactly as given; a file-sourced value has only its
	// surrounding whitespace trimmed by the .glassfrogrc parser (rcfile.Parse),
	// never any change to the URL itself.
	Value string
	// Source is which rung supplied Value.
	Source BaseURLSource
	// Path is the file Value was read from, when Source is SourceFile; empty
	// otherwise.
	Path string
}

// BaseURLError reports that a non-empty source supplied a value that is not an
// absolute http(s) URL. It is code-free (the consuming command and Exit-Code
// Convention (004) map it to an exit code — ADR-4) and carries only the source
// label, never anything secret: the token is never in scope on a base-URL path.
type BaseURLError struct {
	// Source names where the malformed value came from: the flag (--base-url),
	// the environment variable (GLASSFROG_BASE_URL), or the .glassfrogrc file
	// path. It is safe to display.
	Source string
}

func (e *BaseURLError) Error() string {
	return fmt.Sprintf("base URL from %s is not a valid absolute http(s) URL", e.Source)
}

// Production seam: the only places this resolver reads process/OS globals. They
// are package variables so tests exercise resolution hermetically over temp
// directories and a controlled environment, never the developer's real home or
// working directory (ADR-5, parallel to internal/auth).
var (
	getenv      = os.Getenv
	getwd       = os.Getwd
	userHomeDir = os.UserHomeDir
)

// ResolveBaseURLFromOS resolves the base URL using the real working directory,
// home directory, and environment, given the flag value supplied at invocation
// and whether the --base-url flag was present (cobra Changed()). It is the thin
// production entrypoint over ResolveBaseURL — the seam binding the pure algorithm
// to the OS globals (parallel to auth.Resolve). A working directory that cannot
// be determined is an error; a home directory that cannot be determined simply
// drops the home fallback.
func ResolveBaseURLFromOS(flagValue string, flagPresent bool) (BaseURL, error) {
	startDir, err := getwd()
	if err != nil {
		return BaseURL{}, fmt.Errorf("could not determine the working directory: %w", err)
	}
	homeDir, err := userHomeDir()
	if err != nil {
		homeDir = "" // no home → skip the home fallback rather than fail
	}
	return ResolveBaseURL(flagValue, flagPresent, startDir, homeDir)
}

// ResolveBaseURL walks the precedence chain — the --base-url flag, then
// GLASSFROG_BASE_URL, then the nearest .glassfrogrc base_url up the tree (and the
// home file), then the built-in default — by composing the one shared resolve
// walk (039) over four sources. It reads GLASSFROG_BASE_URL through the env seam
// and the file rung through the generic internal/rcfile walk (resolve.FromFile
// over the base_url key); startDir and homeDir are injected so the walk is
// hermetic.
//
// The flag rung is PRESENCE-based (ADR-2): a supplied flag (flagPresent, cobra
// Changed()) wins its rung even when its value is empty or whitespace — the
// operator's explicit act of typing the flag is the signal. An unsupplied flag
// does not yield and the walk falls through to the environment. The env and file
// rungs keep their non-empty-after-trim yield rule, so a whitespace-only
// GLASSFROG_BASE_URL / file value still falls through (unchanged).
//
// Validation runs on the resolved WINNER, not inside the walk (ADR-3): because
// resolve returns the first YIELDING (not first valid) source, an invalid
// high-precedence value still wins and is rejected here — never silently
// superseded by a lower source. "Usable" means an absolute URL carrying an http
// or https scheme. A present-but-unusable value fails loud with a typed
// BaseURLError naming the source via Provenance.Origin, with NO fall-through. A
// file that exists but cannot be read or parsed surfaces internal/rcfile's typed
// read/format error verbatim (also no fall-through), before any validation. The
// built-in default is valid by construction and never re-validated, so the chain
// always yields a value.
//
// The token is never in scope on any base-URL path; the value carries no secret
// and is passed through verbatim. The resolver makes no network call, no write,
// and emits no exit code.
func ResolveBaseURL(flagValue string, flagPresent bool, startDir, homeDir string) (BaseURL, error) {
	res, err := resolve.Resolve(
		resolve.FromFlags(resolve.Flag{Name: "--" + FlagBaseURL, Present: flagPresent, Value: flagValue}),
		resolve.FromEnv(getenv, EnvVarBaseURL),
		resolve.FromFile(startDir, homeDir, baseURLKey),
		resolve.Default(DefaultBaseURL),
	)
	if err != nil {
		return BaseURL{}, err // unreadable/unparseable .glassfrogrc → fail loud, before validation
	}

	// Validate the winner unless it is the default (valid by construction).
	if res.Provenance.Kind != resolve.KindDefault && !isUsableURL(res.Value) {
		return BaseURL{}, &BaseURLError{Source: res.Provenance.Origin}
	}

	switch res.Provenance.Kind {
	case resolve.KindFlag:
		return BaseURL{Value: res.Value, Source: SourceFlag}, nil
	case resolve.KindEnv:
		return BaseURL{Value: res.Value, Source: SourceEnvironment}, nil
	case resolve.KindFile:
		return BaseURL{Value: res.Value, Source: SourceFile, Path: res.Provenance.Origin}, nil
	default:
		return BaseURL{Value: res.Value, Source: SourceDefault}, nil
	}
}

// isUsableURL reports whether value is an absolute URL carrying an http or https
// scheme — the "usable" contract (ADR-4). A scheme-less host, a non-http(s)
// scheme, a missing host, and an unparseable value are all not usable. The value
// is checked verbatim; no trimming or normalization is applied.
func isUsableURL(value string) bool {
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}
