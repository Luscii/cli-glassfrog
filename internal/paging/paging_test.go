package paging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
)

// fakeExecutor is the hermetic stand-in for *apiclient.Client: it decodes a
// canned page body into out, keyed by the cursor the walker sent ("" for the
// first request). It records every call — count, the query sent, the cursor sent
// — so tests can assert cursor threading, query preservation, and the "did not
// loop / did not re-issue" tripwire. It never touches the network.
type fakeExecutor struct {
	pages map[string]fakePage // keyed by the cursor on the request ("" = first page)

	calls       int
	sentQueries []url.Values
	sentCursors []string

	// cancelOnCursor, when matched after serving a page, cancels the walk's context
	// so the NEXT Execute observes a cancelled reqCtx (models a mid-walk cancel).
	cancelOnCursor string
	cancel         context.CancelFunc
}

type fakePage struct {
	body string // JSON decoded into out on success
	err  error  // returned instead of a body when non-nil
}

func (f *fakeExecutor) Execute(reqCtx context.Context, req apiclient.Request, out any) (*apiclient.Response, error) {
	f.calls++
	// Capture a clone so a later mutation of the request can't retro-edit the record.
	f.sentQueries = append(f.sentQueries, cloneQuery(req.Query))
	cursor := req.Query.Get("cursor")
	f.sentCursors = append(f.sentCursors, cursor)

	if err := reqCtx.Err(); err != nil {
		return nil, err // a cancelled/expired context surfaces as the Execute error
	}

	p, ok := f.pages[cursor]
	if !ok {
		return nil, fmt.Errorf("fake: no canned page for cursor %q (calls=%d)", cursor, f.calls)
	}
	if p.err != nil {
		return nil, p.err
	}
	if err := json.Unmarshal([]byte(p.body), out); err != nil {
		return nil, err
	}
	if f.cancel != nil && cursor == f.cancelOnCursor {
		f.cancel()
	}
	return &apiclient.Response{StatusCode: 200}, nil
}

// page builds a page body with the given role names and pagination fields.
func page(hasNext bool, nextCursor string, names ...string) string {
	roles := make([]string, 0, len(names))
	for _, n := range names {
		roles = append(roles, fmt.Sprintf(`{"id":"role_%s","name":%q}`, n, n))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"pagination":{"per_page":500,"has_next_page":%t,"next_cursor":%q}}}`,
		joinComma(roles), hasNext, nextCursor,
	)
}

func joinComma(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

func names(rs []glassfrog.Role) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Name
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- Complete walks ---

func TestAll_SinglePageCompletes(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"": {body: page(false, "", "Alpha", "Bravo")},
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"})

	if !res.Complete || res.Stop != nil {
		t.Fatalf("Complete=%v Stop=%v, want complete with no stop", res.Complete, res.Stop)
	}
	if res.Pages != 1 {
		t.Errorf("Pages=%d, want 1", res.Pages)
	}
	if ex.calls != 1 {
		t.Errorf("Execute calls=%d, want 1 (no further request)", ex.calls)
	}
	if !eq(names(res.Records), []string{"Alpha", "Bravo"}) {
		t.Errorf("Records=%v, want [Alpha Bravo] in API order", names(res.Records))
	}
}

func TestAll_MultiplePagesAssembledInOrder(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"":   {body: page(true, "c1", "A1", "A2")},
		"c1": {body: page(true, "c2", "B1", "B2")},
		"c2": {body: page(false, "", "C1")},
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"})

	if !res.Complete || res.Stop != nil {
		t.Fatalf("Complete=%v Stop=%v, want complete", res.Complete, res.Stop)
	}
	if res.Pages != 3 || ex.calls != 3 {
		t.Errorf("Pages=%d calls=%d, want 3 and 3", res.Pages, ex.calls)
	}
	if !eq(names(res.Records), []string{"A1", "A2", "B1", "B2", "C1"}) {
		t.Errorf("Records=%v, want concatenated in API order across pages", names(res.Records))
	}
	// Cursor threaded: page 1 sent no cursor, pages 2/3 sent the prior next_cursor.
	if !eq(ex.sentCursors, []string{"", "c1", "c2"}) {
		t.Errorf("sentCursors=%v, want [\"\" c1 c2] (cursor threaded from prior response)", ex.sentCursors)
	}
}

func TestAll_AbsentPaginationCompletesInOnePage(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"": {body: `{"data":[{"id":"role_1","name":"Solo"}]}`}, // no meta.pagination
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/roles"})

	if !res.Complete || res.Stop != nil {
		t.Fatalf("Complete=%v Stop=%v, want complete for an absent meta.pagination", res.Complete, res.Stop)
	}
	if ex.calls != 1 {
		t.Errorf("calls=%d, want 1 (no cursor request)", ex.calls)
	}
	if !eq(names(res.Records), []string{"Solo"}) {
		t.Errorf("Records=%v, want [Solo]", names(res.Records))
	}
}

func TestAll_EmptyResultIsComplete(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"": {body: `{"data":[],"meta":{"pagination":{"has_next_page":false}}}`},
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"})

	if !res.Complete || res.Stop != nil {
		t.Fatalf("Complete=%v Stop=%v, want complete", res.Complete, res.Stop)
	}
	if len(res.Records) != 0 {
		t.Errorf("Records=%v, want empty", res.Records)
	}
	if res.Pages != 1 {
		t.Errorf("Pages=%d, want 1", res.Pages)
	}
}

// --- Mid-walk and first-page failures (partial retained) ---

func TestAll_MidWalkFailureRetainsPartialSet(t *testing.T) {
	rateLimited := &apiclient.ResponseError{StatusCode: 429}
	ex := &fakeExecutor{pages: map[string]fakePage{
		"":   {body: page(true, "c1", "A1")},
		"c1": {body: page(true, "c2", "B1")},
		"c2": {err: rateLimited}, // page 3 fails
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"})

	if res.Complete {
		t.Fatalf("Complete=true, want false on a mid-walk failure")
	}
	var respErr *apiclient.ResponseError
	if !errors.As(res.Stop, &respErr) || respErr.StatusCode != 429 {
		t.Fatalf("Stop=%v, want the 429 *ResponseError that stopped the walk", res.Stop)
	}
	if !eq(names(res.Records), []string{"A1", "B1"}) {
		t.Errorf("Records=%v, want the two pages gathered before the failure (partial retained)", names(res.Records))
	}
	if res.Pages != 3 {
		t.Errorf("Pages=%d, want 3 (the failed request counts)", res.Pages)
	}
}

func TestAll_FirstPageFailureIsEmptyIncomplete(t *testing.T) {
	transport := &apiclient.TransportError{}
	ex := &fakeExecutor{pages: map[string]fakePage{
		"": {err: transport},
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"})

	if res.Complete {
		t.Fatalf("Complete=true, want false on a first-page failure")
	}
	if len(res.Records) != 0 {
		t.Errorf("Records=%v, want empty on a first-page failure", res.Records)
	}
	var transErr *apiclient.TransportError
	if !errors.As(res.Stop, &transErr) {
		t.Errorf("Stop=%v, want the *TransportError", res.Stop)
	}
	if res.Pages != 1 {
		t.Errorf("Pages=%d, want 1", res.Pages)
	}
}

// --- Non-advancing cursor: BOTH variants, each with a no-loop tripwire ---

func TestAll_BlankCursorUnderHasNextDoesNotLoop(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"": {body: page(true, "", "A1")}, // has_next_page=true but blank next_cursor
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"})

	assertMalformed(t, res, 1)
	if ex.calls != 1 {
		t.Errorf("calls=%d, want exactly 1 (must not re-issue with an empty cursor)", ex.calls)
	}
	if !eq(names(res.Records), []string{"A1"}) {
		t.Errorf("Records=%v, want the records gathered so far retained", names(res.Records))
	}
}

func TestAll_RepeatedCursorUnderHasNextDoesNotLoop(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"":   {body: page(true, "c1", "A1")},
		"c1": {body: page(true, "c1", "B1")}, // returns the SAME cursor it was sent
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"})

	assertMalformed(t, res, 2)
	// The tripwire: a buggy blank-only guard would re-request cursor "c1" forever.
	if ex.calls != 2 {
		t.Errorf("calls=%d, want exactly 2 (must not re-request the repeated cursor)", ex.calls)
	}
	if !eq(names(res.Records), []string{"A1", "B1"}) {
		t.Errorf("Records=%v, want both gathered pages retained", names(res.Records))
	}
}

func assertMalformed(t *testing.T, res Result[glassfrog.Role], wantPage int) {
	t.Helper()
	if res.Complete {
		t.Fatalf("Complete=true, want false for a non-advancing cursor")
	}
	var mp *MalformedPageError
	if !errors.As(res.Stop, &mp) {
		t.Fatalf("Stop=%v, want *MalformedPageError", res.Stop)
	}
	if mp.Page != wantPage {
		t.Errorf("MalformedPageError.Page=%d, want %d", mp.Page, wantPage)
	}
}

// --- Cancellation ---

func TestAll_CancelledContextStopsWithPartialSet(t *testing.T) {
	reqCtx, cancel := context.WithCancel(context.Background())
	ex := &fakeExecutor{
		pages: map[string]fakePage{
			"":   {body: page(true, "c1", "A1")},
			"c1": {body: page(true, "c2", "B1")}, // would be served if not cancelled
		},
		cancelOnCursor: "", // cancel right after serving the first page
		cancel:         cancel,
	}

	res := All[glassfrog.Role](reqCtx, ex, apiclient.Request{Method: "GET", Path: "/me/roles"})

	if res.Complete {
		t.Fatalf("Complete=true, want false after a mid-walk cancellation")
	}
	if !errors.Is(res.Stop, context.Canceled) {
		t.Errorf("Stop=%v, want context.Canceled", res.Stop)
	}
	if !eq(names(res.Records), []string{"A1"}) {
		t.Errorf("Records=%v, want the one page gathered before cancellation", names(res.Records))
	}
}

// --- Query handling: preserve caller params, set only per_page+cursor, no mutation ---

func TestAll_PreservesCallerQueryAndSetsOnlyPagingParams(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"":   {body: page(true, "c1", "A1")},
		"c1": {body: page(false, "", "B1")},
	}}
	callerQuery := url.Values{"q": []string{"finance"}, "include": []string{"roles"}}
	req := apiclient.Request{Method: "GET", Path: "/me/roles", Query: callerQuery}

	All[glassfrog.Role](context.Background(), ex, req)

	if len(ex.sentQueries) != 2 {
		t.Fatalf("sent %d page requests, want 2", len(ex.sentQueries))
	}
	for i, q := range ex.sentQueries {
		if q.Get("q") != "finance" {
			t.Errorf("page %d: q=%q, want finance preserved", i+1, q.Get("q"))
		}
		if q.Get("include") != "roles" {
			t.Errorf("page %d: include=%q, want roles preserved", i+1, q.Get("include"))
		}
		if q.Get("per_page") != "500" {
			t.Errorf("page %d: per_page=%q, want 500 (default)", i+1, q.Get("per_page"))
		}
	}
	// First page omits cursor; the second carries the prior next_cursor.
	if ex.sentQueries[0].Get("cursor") != "" {
		t.Errorf("first request carried cursor=%q, want none", ex.sentQueries[0].Get("cursor"))
	}
	if ex.sentQueries[1].Get("cursor") != "c1" {
		t.Errorf("second request cursor=%q, want c1", ex.sentQueries[1].Get("cursor"))
	}
	// The caller's own map is never mutated.
	if callerQuery.Get("per_page") != "" || callerQuery.Get("cursor") != "" {
		t.Errorf("caller's req.Query was mutated: %v", callerQuery)
	}
	if len(callerQuery) != 2 {
		t.Errorf("caller's req.Query grew to %v, want only its original q+include", callerQuery)
	}
}

func TestAll_WithPageSizeOverridesPerPage(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"": {body: page(false, "", "A1")},
	}}

	All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"}, WithPageSize(250))

	if got := ex.sentQueries[0].Get("per_page"); got != "250" {
		t.Errorf("per_page=%q, want 250 (WithPageSize override sent as-is)", got)
	}
}

func TestAll_NilCallerQueryStillSetsPagingParams(t *testing.T) {
	ex := &fakeExecutor{pages: map[string]fakePage{
		"": {body: page(false, "", "A1")},
	}}

	res := All[glassfrog.Role](context.Background(), ex, apiclient.Request{Method: "GET", Path: "/me/roles"}) // no Query

	if !res.Complete {
		t.Fatalf("Complete=false, want true")
	}
	if ex.sentQueries[0].Get("per_page") != "500" {
		t.Errorf("per_page=%q, want 500 even with a nil caller Query", ex.sentQueries[0].Get("per_page"))
	}
}
