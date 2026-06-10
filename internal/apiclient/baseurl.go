package apiclient

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Luscii/cli-glassfrog/internal/rcfile"
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
// home directory, and environment, given the flag value supplied at invocation.
// It is the thin production entrypoint over ResolveBaseURL — the seam binding the
// pure algorithm to the OS globals (parallel to auth.Resolve). The --base-url
// flag's cobra registration is deferred to the consuming command; this accepts
// the flag value as an input now (the build-ahead pattern of 005/007). A working
// directory that cannot be determined is an error; a home directory that cannot
// be determined simply drops the home fallback.
func ResolveBaseURLFromOS(flagValue string) (BaseURL, error) {
	startDir, err := getwd()
	if err != nil {
		return BaseURL{}, fmt.Errorf("could not determine the working directory: %w", err)
	}
	homeDir, err := userHomeDir()
	if err != nil {
		homeDir = "" // no home → skip the home fallback rather than fail
	}
	return ResolveBaseURL(flagValue, startDir, homeDir)
}

// ResolveBaseURL walks the precedence chain — the --base-url flag value, then
// GLASSFROG_BASE_URL, then the nearest .glassfrogrc base_url up the tree (and the
// home file), then the built-in default — and returns the first source that
// yields a usable value (ADR-2). It reads GLASSFROG_BASE_URL through the env seam
// and the file rung through the generic internal/rcfile walk (rcfile.Resolve over
// the base_url key); startDir and homeDir are injected so the walk is hermetic.
//
// "Usable" means an absolute URL carrying an http or https scheme (ADR-4). A
// whitespace-only flag/env/file value is treated as absent and falls through; a
// non-empty value that is not a usable URL fails loud with a typed BaseURLError
// naming the source, with NO fall-through to a lower-precedence source. A file
// that exists but cannot be read or parsed surfaces internal/rcfile's typed
// read/format error (also no fall-through). The built-in default is valid by
// construction and never re-validated, so the chain always yields a value.
//
// The token is never in scope on any base-URL path; the value carries no secret
// and is passed through verbatim. The resolver makes no network call, no write,
// and emits no exit code.
func ResolveBaseURL(flagValue, startDir, homeDir string) (BaseURL, error) {
	// Rung 1: the flag. A non-empty value (after trimming) short-circuits all
	// else; a usable value is used verbatim, a malformed one fails loud.
	if strings.TrimSpace(flagValue) != "" {
		if !isUsableURL(flagValue) {
			return BaseURL{}, &BaseURLError{Source: "--" + FlagBaseURL}
		}
		return BaseURL{Value: flagValue, Source: SourceFlag}, nil
	}

	// Rung 2: GLASSFROG_BASE_URL. Same usable/malformed/absent rules, uniformly.
	if envValue := getenv(EnvVarBaseURL); strings.TrimSpace(envValue) != "" {
		if !isUsableURL(envValue) {
			return BaseURL{}, &BaseURLError{Source: EnvVarBaseURL}
		}
		return BaseURL{Value: envValue, Source: SourceEnvironment}, nil
	}

	// Rung 3: the nearest .glassfrogrc base_url up the tree, then home, via the
	// generic rcfile walk. A typed read/format error fails loud (no fall-through
	// to the default); a base_url-less or missing file is skipped inside the walk.
	fileValue, filePath, found, err := rcfile.Resolve(startDir, homeDir, baseURLKey)
	if err != nil {
		return BaseURL{}, err
	}
	if found {
		if !isUsableURL(fileValue) {
			return BaseURL{}, &BaseURLError{Source: filePath}
		}
		return BaseURL{Value: fileValue, Source: SourceFile, Path: filePath}, nil
	}

	// Rung 4: the built-in default backstops the chain — known-valid, not
	// re-validated, always a value.
	return BaseURL{Value: DefaultBaseURL, Source: SourceDefault}, nil
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
