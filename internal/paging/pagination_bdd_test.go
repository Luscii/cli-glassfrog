package paging

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/cucumber/godog"
)

// TestPaginationFeatures runs the 016 driving scenarios against the All[T]
// walker, driven entirely over a fake Executor (no real network or filesystem).
//
// The suite is scoped to *only* pagination.feature: godog binds steps per-suite,
// so a directory-globbing Paths would pull in sibling suites' scenarios and fail
// with undefined steps (LEARNINGS 2026-06-04). This is internal/paging's own
// suite, pointed at its own file. The three @validation scenarios stay @wip —
// held out for the validate skill — so the "~@wip" filter excludes them here.
func TestPaginationFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializePaginationScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/silent-truncation/pagination.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// pagingWorld is the per-scenario state. A Given scripts the fake Executor's
// canned pages and the request; a When runs the walk; the Thens assert on the
// captured Result. Step helpers return errors, never panic (LEARNINGS).
type pagingWorld struct {
	ex          *fakeExecutor
	req         apiclient.Request
	reqCtx      context.Context
	opts        []Option
	wantRecords []string // the names the walk should have gathered, in API order

	res Result[glassfrog.Role]
	ran bool
}

func (w *pagingWorld) walk() error {
	w.res = All[glassfrog.Role](w.reqCtx, w.ex, w.req, w.opts...)
	w.ran = true
	return nil
}

func initializePaginationScenario(sc *godog.ScenarioContext) {
	w := &pagingWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = pagingWorld{
			ex:     &fakeExecutor{pages: map[string]fakePage{}},
			req:    apiclient.Request{Method: "GET", Path: "/me/roles"},
			reqCtx: context.Background(),
		}
		return ctx, nil
	})

	// --- Givens: complete-walk shapes ---
	sc.Step(`^a list endpoint whose first response reported no further pages$`, w.givenSinglePage)
	sc.Step(`^a list endpoint with three pages of results$`, w.givenThreePages)
	sc.Step(`^a list endpoint whose response carried no pagination block$`, w.givenNoPaginationBlock)
	sc.Step(`^a list endpoint whose first response carried no records and reported no further pages$`, w.givenEmptyComplete)
	sc.Step(`^a list request carrying a search filter and an include set$`, w.givenFilteredRequest)

	// --- Givens: cut-short shapes ---
	sc.Step(`^a walk that had already gathered two pages$`, w.givenTwoGatheredPages)
	sc.Step(`^the next page request fails with a 429 rate-limit response$`, w.givenNextPage429)
	sc.Step(`^a list endpoint whose first page request fails$`, w.givenFirstPageFails)
	sc.Step(`^a page that reported a further page but carried a blank or absent cursor$`, w.givenBlankCursor)
	sc.Step(`^a page that reported a further page but carried a cursor identical to the one just used$`, w.givenRepeatedCursor)
	sc.Step(`^a walk in progress that had gathered one page$`, w.givenOneGatheredPage)
	sc.Step(`^the request context is cancelled before the next page$`, w.givenContextCancelled)

	// --- Whens ---
	sc.Step(`^a list command walks the endpoint$`, w.walk)
	sc.Step(`^the walker requests each subsequent page$`, w.walk)
	sc.Step(`^the walker continues the walk$`, w.walk)
	sc.Step(`^the walker inspects the page$`, w.walk)

	// --- Thens: complete ---
	sc.Step(`^the walker will return a complete result carrying that page's records in API order$`, w.thenCompleteInOrder)
	sc.Step(`^it will issue no further request$`, w.thenOneCall)
	sc.Step(`^the walker will request each next page using the prior response's cursor$`, w.thenCursorThreaded)
	sc.Step(`^it will return a complete result carrying all records from the three pages in API order$`, w.thenCompleteInOrder)
	sc.Step(`^the walker will treat the single response as the complete result$`, w.thenCompleteInOrder)
	sc.Step(`^it will issue no cursor request$`, w.thenNoCursorRequest)
	sc.Step(`^the walker will return a complete result carrying no records$`, w.thenCompleteNoRecords)
	sc.Step(`^every page request will preserve the caller's search filter and include set$`, w.thenFilterPreserved)
	sc.Step(`^it will set only the page-size and cursor parameters$`, w.thenOnlyPagingParamsAdded)

	// --- Thens: cut-short ---
	sc.Step(`^it will stop and return the two pages already gathered, flagged incomplete$`, w.thenStoppedWithTwoPages)
	sc.Step(`^the result will carry the failure that stopped it$`, w.thenStopCarriesFailure)
	sc.Step(`^the partial set will not be presented as complete$`, w.thenNotComplete)
	sc.Step(`^the walker will return an empty result flagged incomplete carrying the failure$`, w.thenEmptyIncompleteWithFailure)
	sc.Step(`^it will not report success$`, w.thenNotComplete)
	sc.Step(`^it will not re-issue a request with an empty cursor and will not loop$`, w.thenOneCall)
	sc.Step(`^it will return the records gathered so far flagged incomplete, naming the malformed-paging boundary$`, w.thenMalformedBoundary)
	sc.Step(`^it will not re-issue the same cursor and will not loop$`, w.thenTwoCalls)
	sc.Step(`^it will stop and return the records gathered so far, flagged incomplete$`, w.thenStoppedWithGathered)
	sc.Step(`^the result will carry the cancellation as the failure that stopped it$`, w.thenStopIsCancellation)
}

// --- Given implementations ---

func (w *pagingWorld) givenSinglePage() error {
	w.ex.pages[""] = fakePage{body: page(false, "", "Alpha", "Bravo")}
	w.wantRecords = []string{"Alpha", "Bravo"}
	return nil
}

func (w *pagingWorld) givenThreePages() error {
	w.ex.pages[""] = fakePage{body: page(true, "c1", "A1", "A2")}
	w.ex.pages["c1"] = fakePage{body: page(true, "c2", "B1")}
	w.ex.pages["c2"] = fakePage{body: page(false, "", "C1", "C2")}
	w.wantRecords = []string{"A1", "A2", "B1", "C1", "C2"}
	return nil
}

func (w *pagingWorld) givenNoPaginationBlock() error {
	w.ex.pages[""] = fakePage{body: `{"data":[{"id":"role_1","name":"Solo"}]}`}
	w.wantRecords = []string{"Solo"}
	return nil
}

func (w *pagingWorld) givenEmptyComplete() error {
	w.ex.pages[""] = fakePage{body: `{"data":[],"meta":{"pagination":{"has_next_page":false}}}`}
	w.wantRecords = nil
	return nil
}

func (w *pagingWorld) givenFilteredRequest() error {
	w.req.Query = map[string][]string{"q": {"finance"}, "include": {"roles"}}
	w.ex.pages[""] = fakePage{body: page(true, "c1", "A1")}
	w.ex.pages["c1"] = fakePage{body: page(false, "", "B1")}
	w.wantRecords = []string{"A1", "B1"}
	return nil
}

func (w *pagingWorld) givenTwoGatheredPages() error {
	w.ex.pages[""] = fakePage{body: page(true, "c1", "A1")}
	w.ex.pages["c1"] = fakePage{body: page(true, "c2", "B1")}
	w.wantRecords = []string{"A1", "B1"}
	return nil
}

func (w *pagingWorld) givenNextPage429() error {
	w.ex.pages["c2"] = fakePage{err: &apiclient.ResponseError{StatusCode: 429}}
	return nil
}

func (w *pagingWorld) givenFirstPageFails() error {
	w.ex.pages[""] = fakePage{err: &apiclient.TransportError{}}
	w.wantRecords = nil
	return nil
}

func (w *pagingWorld) givenBlankCursor() error {
	w.ex.pages[""] = fakePage{body: page(true, "", "A1")} // has_next_page=true, blank next_cursor
	w.wantRecords = []string{"A1"}
	return nil
}

func (w *pagingWorld) givenRepeatedCursor() error {
	w.ex.pages[""] = fakePage{body: page(true, "c1", "A1")}
	w.ex.pages["c1"] = fakePage{body: page(true, "c1", "B1")} // returns the same cursor it was sent
	w.wantRecords = []string{"A1", "B1"}
	return nil
}

func (w *pagingWorld) givenOneGatheredPage() error {
	w.ex.pages[""] = fakePage{body: page(true, "c1", "A1")}
	w.ex.pages["c1"] = fakePage{body: page(true, "c2", "B1")} // served only if the walk is not cancelled
	w.wantRecords = []string{"A1"}
	return nil
}

func (w *pagingWorld) givenContextCancelled() error {
	ctx, cancel := context.WithCancel(context.Background())
	w.reqCtx = ctx
	w.ex.cancel = cancel
	w.ex.cancelOnCursor = "" // cancel right after serving the first page
	return nil
}

// --- Then implementations ---

func (w *pagingWorld) thenCompleteInOrder() error {
	if !w.res.Complete || w.res.Stop != nil {
		return fmt.Errorf("Complete=%v Stop=%v, want complete with no stop", w.res.Complete, w.res.Stop)
	}
	if !eq(names(w.res.Records), w.wantRecords) {
		return fmt.Errorf("Records=%v, want %v in API order", names(w.res.Records), w.wantRecords)
	}
	return nil
}

func (w *pagingWorld) thenOneCall() error {
	if w.ex.calls != 1 {
		return fmt.Errorf("Execute calls=%d, want exactly 1", w.ex.calls)
	}
	return nil
}

func (w *pagingWorld) thenTwoCalls() error {
	if w.ex.calls != 2 {
		return fmt.Errorf("Execute calls=%d, want exactly 2 (no re-issue of the repeated cursor)", w.ex.calls)
	}
	return nil
}

func (w *pagingWorld) thenCursorThreaded() error {
	if !eq(w.ex.sentCursors, []string{"", "c1", "c2"}) {
		return fmt.Errorf("sentCursors=%v, want [\"\" c1 c2] (each next page uses the prior response's cursor)", w.ex.sentCursors)
	}
	return nil
}

func (w *pagingWorld) thenNoCursorRequest() error {
	if w.ex.calls != 1 {
		return fmt.Errorf("calls=%d, want 1 (no cursor request issued)", w.ex.calls)
	}
	if w.ex.sentQueries[0].Get("cursor") != "" {
		return fmt.Errorf("first request carried cursor=%q, want none", w.ex.sentQueries[0].Get("cursor"))
	}
	return nil
}

func (w *pagingWorld) thenCompleteNoRecords() error {
	if !w.res.Complete || w.res.Stop != nil {
		return fmt.Errorf("Complete=%v Stop=%v, want complete", w.res.Complete, w.res.Stop)
	}
	if len(w.res.Records) != 0 {
		return fmt.Errorf("Records=%v, want none", names(w.res.Records))
	}
	return nil
}

func (w *pagingWorld) thenFilterPreserved() error {
	if len(w.ex.sentQueries) < 2 {
		return fmt.Errorf("sent %d page requests, want at least 2", len(w.ex.sentQueries))
	}
	for i, q := range w.ex.sentQueries {
		if q.Get("q") != "finance" {
			return fmt.Errorf("page %d: q=%q, want finance preserved", i+1, q.Get("q"))
		}
		if q.Get("include") != "roles" {
			return fmt.Errorf("page %d: include=%q, want roles preserved", i+1, q.Get("include"))
		}
	}
	return nil
}

func (w *pagingWorld) thenOnlyPagingParamsAdded() error {
	allowed := map[string]bool{"q": true, "include": true, "per_page": true, "cursor": true}
	for i, q := range w.ex.sentQueries {
		for key := range q {
			if !allowed[key] {
				return fmt.Errorf("page %d: unexpected param %q added — only per_page/cursor may be set atop the caller's params", i+1, key)
			}
		}
		if q.Get("per_page") == "" {
			return fmt.Errorf("page %d: per_page was not set", i+1)
		}
	}
	return nil
}

func (w *pagingWorld) thenStoppedWithTwoPages() error {
	if err := w.thenNotComplete(); err != nil {
		return err
	}
	if !eq(names(w.res.Records), []string{"A1", "B1"}) {
		return fmt.Errorf("Records=%v, want the two pages gathered before the stop", names(w.res.Records))
	}
	return nil
}

func (w *pagingWorld) thenStopCarriesFailure() error {
	if w.res.Stop == nil {
		return errors.New("Stop=nil, want the failure that stopped the walk")
	}
	return nil
}

func (w *pagingWorld) thenNotComplete() error {
	if w.res.Complete {
		return errors.New("Complete=true, want false (a partial set must never read as complete)")
	}
	if w.res.Stop == nil {
		return errors.New("Stop=nil on an incomplete walk, want a stopping cause")
	}
	return nil
}

func (w *pagingWorld) thenEmptyIncompleteWithFailure() error {
	if w.res.Complete {
		return errors.New("Complete=true, want false on a first-page failure")
	}
	if len(w.res.Records) != 0 {
		return fmt.Errorf("Records=%v, want empty on a first-page failure", names(w.res.Records))
	}
	if w.res.Stop == nil {
		return errors.New("Stop=nil, want the failure carried")
	}
	return nil
}

func (w *pagingWorld) thenMalformedBoundary() error {
	if err := w.thenNotComplete(); err != nil {
		return err
	}
	var mp *MalformedPageError
	if !errors.As(w.res.Stop, &mp) {
		return fmt.Errorf("Stop=%v, want *MalformedPageError naming the malformed-paging boundary", w.res.Stop)
	}
	if !eq(names(w.res.Records), w.wantRecords) {
		return fmt.Errorf("Records=%v, want %v gathered so far retained", names(w.res.Records), w.wantRecords)
	}
	return nil
}

func (w *pagingWorld) thenStoppedWithGathered() error {
	if err := w.thenNotComplete(); err != nil {
		return err
	}
	if !eq(names(w.res.Records), w.wantRecords) {
		return fmt.Errorf("Records=%v, want %v gathered before the stop", names(w.res.Records), w.wantRecords)
	}
	return nil
}

func (w *pagingWorld) thenStopIsCancellation() error {
	if !errors.Is(w.res.Stop, context.Canceled) {
		return fmt.Errorf("Stop=%v, want context.Canceled", w.res.Stop)
	}
	return nil
}
