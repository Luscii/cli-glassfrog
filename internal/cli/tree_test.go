package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// noTokenContext is a complete-but-unauthenticated context: a parseable base URL
// but no resolved credential, so NewClient fails fast as a usage error (not
// authenticated) before any send — the shape the tree/subroles no-token tests
// drive.
func noTokenContext() apiclient.ConnectionContext {
	return apiclient.ConnectionContext{
		BaseURL: apiclient.BaseURL{Value: "https://example.test/api/v5", Source: apiclient.SourceFlag},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
}

// --- tree fixtures ---------------------------------------------------------

// orgTreeBody is a {data: TreeNode} whole-org tree: an anchor with one child that
// itself has a grandchild (recursion to three levels), the anchor carrying a null
// parent_role_id and a structural flag.
const orgTreeBody = `{
  "data": {
    "id": "role_anchor", "type": "circle", "name": "General Company Circle",
    "purpose": "Run the company", "parent_role_id": null, "has_subroles": true, "flags": ["structural"],
    "children": [
      {"id": "role_mkt", "type": "circle", "name": "Marketing", "purpose": null,
       "parent_role_id": "role_anchor", "has_subroles": true, "flags": [],
       "children": [
         {"id": "role_press", "type": "role", "name": "Press Officer", "purpose": "Press that lands",
          "parent_role_id": "role_mkt", "has_subroles": false, "flags": [], "children": []}
       ]}
    ]
  }
}`

// rootedTreeBody is a {data: TreeNode} rooted at role_0123 with one child.
const rootedTreeBody = `{
  "data": {
    "id": "role_0123", "type": "circle", "name": "Marketing", "purpose": "Markets that know us",
    "parent_role_id": "role_anchor", "has_subroles": true, "flags": [],
    "children": [
      {"id": "role_press", "type": "role", "name": "Press Officer", "purpose": "p",
       "parent_role_id": "role_0123", "has_subroles": false, "flags": [], "children": []}
    ]
  }
}`

// leafTreeBody is a single-node tree (a leaf root: no children, has_subroles false).
const leafTreeBody = `{"data": {"id": "role_0123", "type": "role", "name": "Solo Role", "purpose": "p", "parent_role_id": "role_anchor", "has_subroles": false, "flags": [], "children": []}}`

// cappedTreeBody is a --depth 1 cut: the anchor and its two direct children, one
// of which still has subroles below the cut (children absent), the other a true leaf.
const cappedTreeBody = `{
  "data": {
    "id": "role_anchor", "type": "circle", "name": "Anchor", "purpose": "p",
    "parent_role_id": null, "has_subroles": true, "flags": [],
    "children": [
      {"id": "role_0456", "type": "circle", "name": "Deep Branch", "purpose": "p",
       "parent_role_id": "role_anchor", "has_subroles": true, "flags": []},
      {"id": "role_leaf", "type": "role", "name": "True Leaf", "purpose": "p",
       "parent_role_id": "role_anchor", "has_subroles": false, "flags": []}
    ]
  }
}`

// treeWithIncludesBody carries per-node accountabilities/domains/fillers.
const treeWithIncludesBody = `{
  "data": {
    "id": "role_0123", "type": "circle", "name": "Marketing", "purpose": "p",
    "parent_role_id": null, "has_subroles": false, "flags": [],
    "accountabilities": [{"id": "acc_1", "description": "Defining the campaign"}],
    "domains": [{"id": "dom_1", "description": "The marketing budget"}],
    "fillers": [{"id": "per_x", "name": "Alice Smith", "kind": "human"}],
    "children": []
  }
}`

// runTreeOver drives the pure runTree over a fake seam, returning the outcome and
// captured stdout/stderr, and failing if the token leaks.
func runTreeOver(t *testing.T, seam treeSeam, cfg treeConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	outcome, _ := runTree(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- whole-org + rooted reads ----------------------------------------------

func TestRunTree_WholeOrgWalksNoPagesAndNests(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTreeOver(t, seam, treeConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if tr.calls != 1 {
		t.Errorf("the tree is unpaginated — want exactly 1 request, got %d", tr.calls)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/tree") || strings.Contains(got, "/roles/") {
		t.Errorf("path = %q, want it to target /tree (whole org)", got)
	}
	// Nesting: deeper nodes indent further.
	for _, want := range []string{
		"General Company Circle (role_anchor) [structural]",
		"  Marketing (role_mkt)",
		"    Press Officer (role_press)",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("nested projection missing %q:\n%s", want, stdout)
		}
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("a clean tree read writes nothing to stderr, got %q", stderr)
	}
}

func TestRunTree_RootedSubtreeTargetsRoleEndpoint(t *testing.T) {
	tr := &cannedTransport{status: 200, body: rootedTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runTreeOver(t, seam, treeConfig{args: []string{"role_0123"}})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/roles/role_0123/tree") {
		t.Errorf("path = %q, want /roles/role_0123/tree", got)
	}
	if !strings.Contains(stdout, "Marketing (role_0123)") {
		t.Errorf("the subtree should be rooted at role_0123:\n%s", stdout)
	}
}

func TestRunTree_LeafRootRendersSingleNode(t *testing.T) {
	tr := &cannedTransport{status: 200, body: leafTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runTreeOver(t, seam, treeConfig{args: []string{"role_0123"}})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "Solo Role (role_0123)") {
		t.Errorf("a leaf root should print its single node:\n%s", stdout)
	}
	// No indented child lines (a leaf has nothing nested) and no depth marker.
	if strings.Contains(stdout, "\n  ") && !strings.Contains(stdout, "\n  Purpose:") {
		t.Errorf("a leaf root should have no nested children:\n%s", stdout)
	}
	if strings.Contains(stdout, "(+ subroles below depth)") {
		t.Errorf("a true leaf must not carry the depth-boundary marker:\n%s", stdout)
	}
}

// --- depth ------------------------------------------------------------------

func TestRunTree_DepthSentOnlyWhenSet(t *testing.T) {
	// --depth 1 sends depth=1.
	tr := &cannedTransport{status: 200, body: cappedTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runTreeOver(t, seam, treeConfig{depth: 1, depthSet: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("depth"); got != "1" {
		t.Errorf("depth query = %q, want 1", got)
	}

	// Omitting --depth sends no depth.
	tr2 := &cannedTransport{status: 200, body: orgTreeBody}
	seam2 := &fakeMeSeam{ctx: validMeContext(), transport: tr2}
	if _, _, _ = runTreeOver(t, seam2, treeConfig{}); tr2.lastQuery.Has("depth") {
		t.Errorf("omitting --depth must send no depth param, got %q", tr2.lastQuery.Get("depth"))
	}

	// --depth 0 (root only) is distinct from omitted: it sends depth=0.
	tr3 := &cannedTransport{status: 200, body: leafTreeBody}
	seam3 := &fakeMeSeam{ctx: validMeContext(), transport: tr3}
	if _, _, _ = runTreeOver(t, seam3, treeConfig{depth: 0, depthSet: true}); tr3.lastQuery.Get("depth") != "0" {
		t.Errorf("--depth 0 must send depth=0, got %q", tr3.lastQuery.Get("depth"))
	}
}

func TestRunTree_NegativeDepthIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, stderr := runTreeOver(t, seam, treeConfig{depth: -1, depthSet: true})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("a negative --depth must send no request (tripwire), got %d calls", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no tree data should print on a usage error, got %q", stdout)
	}
	if !strings.Contains(stderr, "--depth") {
		t.Errorf("stderr should name the --depth misuse:\n%s", stderr)
	}
}

func TestRunTree_DepthCappedNodeMarkedDistinctFromLeaf(t *testing.T) {
	tr := &cannedTransport{status: 200, body: cappedTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, _ := runTreeOver(t, seam, treeConfig{depth: 1, depthSet: true})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if !strings.Contains(stdout, "Deep Branch (role_0456)  (+ subroles below depth)") {
		t.Errorf("a depth-capped node must carry the boundary marker:\n%s", stdout)
	}
	if strings.Contains(stdout, "True Leaf (role_leaf)  (+ subroles below depth)") {
		t.Errorf("a true leaf must not carry the marker:\n%s", stdout)
	}
}

// --- include ----------------------------------------------------------------

func TestRunTree_IncludeSendsParamAndEmbedsPerNode(t *testing.T) {
	tr := &cannedTransport{status: 200, body: treeWithIncludesBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, stderr := runTreeOver(t, seam, treeConfig{
		args:    []string{"role_0123"},
		include: []string{"accountabilities", "domains"},
	})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("include"); got != "accountabilities,domains" {
		t.Errorf("include = %q, want accountabilities,domains", got)
	}
	for _, want := range []string{"Accountabilities:", "Defining the campaign", "Domains:", "The marketing budget"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("requested include should embed per node; missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunTree_UnsupportedIncludeRejectedBeforeRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, stderr := runTreeOver(t, seam, treeConfig{
		args:    []string{"role_0123"},
		include: []string{"nonsense"},
	})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --include must send no request (tripwire), got %d calls", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no tree data should print, got %q", stdout)
	}
	if !strings.Contains(stderr, "nonsense") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	// names the tree's supported set (not the subroles set)
	for _, want := range []string{"accountabilities", "domains", "members"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should name the tree include set; missing %q:\n%s", want, stderr)
		}
	}
	if strings.Contains(stderr, "assignments") {
		t.Errorf("stderr must name the TREE set, not the subroles set:\n%s", stderr)
	}
}

// --- pagination flags rejected on tree -------------------------------------

func TestRunTree_PaginationFlagsRejectedNoRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  treeConfig
	}{
		{"first-page", treeConfig{firstPage: true}},
		{"per-page", treeConfig{perPageSet: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := &cannedTransport{status: 200, body: orgTreeBody}
			seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
			outcome, _, stderr := runTreeOver(t, seam, tc.cfg)
			if outcome != UsageError {
				t.Fatalf("outcome = %v, want UsageError", outcome)
			}
			if tr.calls != 0 {
				t.Errorf("a pagination flag on tree must send no request (tripwire), got %d", tr.calls)
			}
			if strings.TrimSpace(stderr) == "" {
				t.Errorf("stderr should name the misuse")
			}
		})
	}
}

// --- error classification ---------------------------------------------------

func TestRunTree_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Role not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, stderr := runTreeOver(t, seam, treeConfig{args: []string{"role_ffff"}})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no tree data should print on an API error, got %q", stdout)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status:\n%s", stderr)
	}
}

func TestRunTree_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runTreeOver(t, seam, treeConfig{})
	if outcome != NetworkUnavailable {
		t.Fatalf("outcome = %v, want NetworkUnavailable", outcome)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a transport failure should be named on stderr")
	}
}

func TestRunTree_NoTokenIsUsageError(t *testing.T) {
	// A no-credential context: NewClient over a context with SourceNone fails fast
	// as a usage error (not authenticated), before any send.
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: &cannedTransport{status: 200, body: orgTreeBody}}
	outcome, stdout, stderr := runTreeOver(t, seam, treeConfig{})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError\nstderr: %s", outcome, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no tree data should print, got %q", stdout)
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
}

// --- structured output ------------------------------------------------------

func TestRunTree_StructuredEmitsRawNestedPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), envOutput: "json", transport: tr}
	outcome, stdout, stderr := runTreeOver(t, seam, treeConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	// The raw nested payload is emitted verbatim (data wrapper + recursion present),
	// not the human projection.
	for _, want := range []string{`"data"`, `"children"`, "role_press"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured output should emit the raw nested payload; missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunTree_InvalidOutputIsUsageErrorNoRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: orgTreeBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, _ := runTreeOver(t, seam, treeConfig{outputFlag: "bogus"})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an invalid --output must send no request, got %d calls", tr.calls)
	}
}
