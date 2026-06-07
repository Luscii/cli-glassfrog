// Package paging holds the multi-page pagination walker (016) — the reusable
// mechanism that turns 010's single-response Execute seam into a complete,
// multi-page result set. It is a library, not a command: it registers no cobra
// command, prints nothing, calls no os.Exit, and never reads the token (identity
// rides the *apiclient.Client it walks through → 007's AuthTransport).
//
// It composes 010 (the Executor seam, the request, the typed code-free errors)
// and the glassfrog schema (the generic Page[T] envelope, the shared Pagination)
// without inverting either's role: apiclient stays schema-agnostic transport and
// glassfrog stays logic-free schema. internal/paging imports apiclient +
// glassfrog; it does NOT import internal/cli — the walk's Stop error is
// classified by the consumer's existing classifyClientError (ADR-1, §101).
package paging

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// defaultPageSize is the per_page the walker requests unless WithPageSize
// overrides it: the API max (500), to minimize round-trips against the rolling
// rate limit (spec decision 4, ADR-4). It is sent as-is — an out-of-range
// override is NOT clamped client-side; the API's 400 surfaces as the walk's Stop.
const defaultPageSize = 500

// Executor is the one-method seam All walks through (ADR-1). It is exactly the
// shape of 010's *apiclient.Client.Execute, so the production client satisfies it
// as-is; tests pass a fake that decodes canned pages into out. The walker depends
// on this interface, never on a concrete client — so its unit tests are hermetic
// (no net/http).
type Executor interface {
	Execute(reqCtx context.Context, req apiclient.Request, out any) (*apiclient.Response, error)
}

// Result is the explicit outcome of a walk (ADR-3): partial-flagged-incomplete,
// never a bare ([]T, error). Records is ALWAYS the set gathered (possibly
// partial); Complete reports whether the walk reached the API's end; Stop names
// the cause iff !Complete. The invariant Complete == (Stop == nil) holds. The
// struct makes "never mistake a partial set for a complete one" structurally hard
// to get wrong: Records is a named field a consumer renders regardless, not a
// value an `if err != nil { return }` reflex silently discards.
type Result[T any] struct {
	// Records is every record gathered across pages, concatenated in API order —
	// never reordered, de-duplicated, or transformed. Partial when !Complete;
	// empty on a first-page failure.
	Records []T
	// Complete is true iff the walk reached the API's end (has_next_page=false, or
	// no meta.pagination). False on any early stop.
	Complete bool
	// Stop is nil iff Complete; otherwise the cause that stopped the walk — a 010
	// typed error (*apiclient.TransportError / *ResponseError / *AuthError /
	// *DecodeError) or *MalformedPageError. errors.As-discriminable; carries no token.
	Stop error
	// Pages is the number of page requests issued (>= 1) — observability + tests.
	Pages int
}

// MalformedPageError is the Stop cause when a page reports has_next_page=true but
// the cursor does not advance — next_cursor is blank/absent OR identical to the
// cursor just used — so the walker stops rather than re-issuing or looping
// forever (ADR-5). It carries no status, so a consumer's classifyClientError maps
// it via the default → RuntimeError(1) fail-safe (no new exit code). Page is the
// 1-based index of the page at which the non-advancing cursor was seen.
type MalformedPageError struct {
	Page int
}

func (e *MalformedPageError) Error() string {
	return fmt.Sprintf("malformed paging: the cursor did not advance at page %d", e.Page)
}

// Option configures a walk. Today only WithPageSize exists; a bounded-walk option
// (WithMaxPages) is a deferred future addition that needs no contract change.
type Option func(*config)

type config struct {
	pageSize int
}

// WithPageSize overrides the per-request per_page. The value is passed through as
// given — no client-side clamp (ADR-4): an out-of-range n (e.g. > 500) surfaces
// the API's 400 as the walk's Stop cause. Flag-level 1–500 validation, if any,
// belongs to the consumer's --per-page flag, not the walker.
func WithPageSize(n int) Option {
	return func(c *config) { c.pageSize = n }
}

// All walks a list endpoint to completion through ex, decoding each page into a
// glassfrog.Page[T], concatenating page.Data in API order until a page reports
// has_next_page=false (or carries no meta.pagination). It returns a Result[T] by
// value — never a bare error (ADR-3).
//
// Per page it CLONES req.Query (an url.Values is shared by reference — the
// caller's map is never mutated), sets per_page to the configured size and cursor
// to the prior page's next_cursor (omitted on the first request), and preserves
// every other caller parameter (q, include, …). It branches on the decoded
// page.Meta.Pagination:
//
//   - has_next_page=false (incl. absent meta) → complete;
//   - has_next_page=true + an advancing cursor (non-blank AND different from the
//     one just sent) → continue;
//   - has_next_page=true + a non-advancing cursor (blank/absent OR equal to the
//     one just sent) → stop with *MalformedPageError, no loop;
//   - any Execute error → stop, retaining the records gathered so far, Stop=err.
func All[T any](reqCtx context.Context, ex Executor, req apiclient.Request, opts ...Option) Result[T] {
	cfg := config{pageSize: defaultPageSize}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg) // skip a nil Option (common when building an option slice conditionally)
		}
	}

	var (
		records []T
		cursor  string // the cursor sent on the current request ("" on the first)
		pages   int
	)

	for {
		// Clone the caller's query per page so the shared url.Values is never
		// mutated; the walker is the SOLE owner of per_page and cursor. Drop any
		// caller-supplied cursor before conditionally setting our own: otherwise a
		// stray cursor in req.Query would survive into the first request (cursor is
		// "" then, so the conditional set below never fires) and start the walk
		// mid-stream — a partial set that reads as complete.
		q := cloneQuery(req.Query)
		q.Set("per_page", fmt.Sprintf("%d", cfg.pageSize))
		q.Del("cursor")
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		pageReq := req
		pageReq.Query = q

		var page glassfrog.Page[T]
		pages++
		if _, err := ex.Execute(reqCtx, pageReq, &page); err != nil {
			// Stop-and-retain: the records gathered so far are kept, never discarded
			// (ADR-3). On a first-page failure records is empty.
			return Result[T]{Records: records, Complete: false, Stop: err, Pages: pages}
		}

		records = append(records, page.Data...)

		pg := page.Meta.Pagination
		if !pg.HasNextPage {
			// Done — reached the API's end (covers an absent meta.pagination, whose
			// zero-valued HasNextPage is false → a non-paginated endpoint).
			return Result[T]{Records: records, Complete: true, Pages: pages}
		}
		if pg.NextCursor == "" || pg.NextCursor == cursor {
			// Non-advancing cursor under has_next_page=true: blank/absent, or identical
			// to the cursor just sent (an API ignoring an unrecognized cursor param
			// would return the same page forever — H-2/H-9). Fail loud, do not loop.
			return Result[T]{Records: records, Complete: false, Stop: &MalformedPageError{Page: pages}, Pages: pages}
		}
		cursor = pg.NextCursor
	}
}

// cloneQuery returns a deep copy of the caller's query so per-page mutation of
// per_page/cursor never touches the shared map. A nil input yields a fresh,
// non-nil url.Values ready for Set.
func cloneQuery(src url.Values) url.Values {
	dst := make(url.Values, len(src))
	for key, vals := range src {
		copied := make([]string, len(vals))
		copy(copied, vals)
		dst[key] = copied
	}
	return dst
}
