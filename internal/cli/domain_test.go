package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- domain fixtures --------------------------------------------------------

// getDomainBody is a representative GET /domains/{id} body: the {data: Domain}
// envelope for a domain bound to a controlling role, no policies embed.
const getDomainBody = `{"data":{"id":"dom_0123","type":"domain","description":"The marketing budget","role_id":"role_0123","created_at":"t","updated_at":"t"}}`

// getDomainBodyNullRole is a domain with a null role_id (unbound to any role).
const getDomainBodyNullRole = `{"data":{"id":"dom_0123","type":"domain","description":"An unbound area","role_id":null,"created_at":"t","updated_at":"t"}}`

// getDomainBodyWithPolicies is the ?include=policies body: the domain plus its
// embedded policies.
const getDomainBodyWithPolicies = `{"data":{"id":"dom_0123","type":"domain","description":"The marketing budget","role_id":"role_0123","created_at":"t","updated_at":"t","policies":[{"id":"pol_1","title":"Spend under $10k needs no approval","body":"b"}]}}`

// runDomainOver drives the pure runDomain over a fake seam, returning the outcome
// and captured stdout/stderr, and failing if the token leaks.
func runDomainOver(t *testing.T, seam domainSeam, cfg domainConfig) (Outcome, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	cfg.seam = seam
	cfg.reqCtx = context.Background()
	cfg.stdout = &out
	cfg.stderr = &errb
	if cfg.id == "" {
		cfg.id = "dom_0123"
	}
	outcome, _ := runDomain(cfg)
	if strings.Contains(out.String()+errb.String(), meSecretToken) {
		t.Fatalf("the token leaked into output: %q", out.String()+errb.String())
	}
	return outcome, out.String(), errb.String()
}

// --- single read success ----------------------------------------------------

func TestRunDomain_ReadsAndPrintsDescriptionAndRole(t *testing.T) {
	tr := &cannedTransport{status: 200, body: getDomainBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastPath; !strings.HasSuffix(got, "/domains/dom_0123") {
		t.Errorf("path = %q, want /domains/dom_0123", got)
	}
	if tr.calls != 1 {
		t.Errorf("the single read should issue exactly one request, got %d", tr.calls)
	}
	for _, want := range []string{"The marketing budget (dom_0123)", "Role: role_0123"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the domain projection should show %q:\n%s", want, stdout)
		}
	}
}

func TestRunDomain_NullRoleShowsMarker(t *testing.T) {
	tr := &cannedTransport{status: 200, body: getDomainBodyNullRole}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if !strings.Contains(stdout, "(no controlling role)") {
		t.Errorf("a null role_id must render the explicit-absence marker, never a bare/empty Role line:\n%s", stdout)
	}
}

func TestRunDomain_NullRoleShowsMarkerCompact(t *testing.T) {
	tr := &cannedTransport{status: 200, body: getDomainBodyNullRole}
	seam := &fakeMeSeam{ctx: validMeContext(), envOutput: "compact", transport: tr}

	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if !strings.Contains(stdout, "role=(no controlling role)") {
		t.Errorf("compact must render the null-role marker:\n%s", stdout)
	}
}

// --- include policies -------------------------------------------------------

func TestRunDomain_IncludePoliciesSendsParamAndEmbeds(t *testing.T) {
	tr := &cannedTransport{status: 200, body: getDomainBodyWithPolicies}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{include: []string{"policies"}})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if got := tr.lastQuery.Get("include"); got != "policies" {
		t.Errorf("include = %q, want policies", got)
	}
	for _, want := range []string{"Policies:", "Spend under $10k needs no approval"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("requested policies should embed inline; missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunDomain_RequestedButEmptyPoliciesShowsMarker(t *testing.T) {
	// --include policies requested, but the body carries none → explicit-absence
	// marker, not an omitted section.
	tr := &cannedTransport{status: 200, body: getDomainBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{include: []string{"policies"}})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	if !strings.Contains(stdout, "Policies:") || !strings.Contains(stdout, "(none)") {
		t.Errorf("requested-but-empty policies should show the explicit-absence marker:\n%s", stdout)
	}
}

func TestRunDomain_OmittedPoliciesSectionWhenNotRequested(t *testing.T) {
	tr := &cannedTransport{status: 200, body: getDomainBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, _ := runDomainOver(t, seam, domainConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if strings.Contains(stdout, "Policies:") {
		t.Errorf("an unrequested policies section must be omitted:\n%s", stdout)
	}
}

// --- include validation -----------------------------------------------------

func TestRunDomain_UnsupportedIncludeRejectedBeforeRequest(t *testing.T) {
	tr := &cannedTransport{status: 200, body: getDomainBody}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}

	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{include: []string{"nonsense"}})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError", outcome)
	}
	if tr.calls != 0 {
		t.Errorf("an unsupported --include must send no request (tripwire), got %d", tr.calls)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print, got %q", stdout)
	}
	if !strings.Contains(stderr, "nonsense") {
		t.Errorf("stderr should name the unsupported value:\n%s", stderr)
	}
	if !strings.Contains(stderr, "policies") {
		t.Errorf("stderr should name the supported set {policies}:\n%s", stderr)
	}
}

// --- list-only flags rejected on the single read ---------------------------

func TestRunDomain_ListFlagsRejectedNoRequest(t *testing.T) {
	cases := []struct {
		name string
		cfg  domainConfig
		flag string
	}{
		{"query", domainConfig{querySet: true}, "--query"},
		{"first-page", domainConfig{firstPage: true}, "--first-page"},
		{"per-page", domainConfig{perPageSet: true}, "--per-page"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr := &cannedTransport{status: 200, body: getDomainBody}
			seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
			outcome, stdout, stderr := runDomainOver(t, seam, tc.cfg)
			if outcome != UsageError {
				t.Fatalf("outcome = %v, want UsageError", outcome)
			}
			if tr.calls != 0 {
				t.Errorf("%s on the single read must send no request (tripwire), got %d", tc.flag, tr.calls)
			}
			if strings.TrimSpace(stdout) != "" {
				t.Errorf("no data should print, got %q", stdout)
			}
			if !strings.Contains(stderr, tc.flag) {
				t.Errorf("stderr should name the %s misuse:\n%s", tc.flag, stderr)
			}
		})
	}
}

// --- error classification ---------------------------------------------------

func TestRunDomain_UnknownIdSurfacesAPIStatus(t *testing.T) {
	tr := &cannedTransport{status: 404, body: `{"detail":"Domain not found"}`}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{id: "dom_ffff"})
	if outcome != APIError {
		t.Fatalf("outcome = %v, want APIError", outcome)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print on an API error, got %q", stdout)
	}
	if !strings.Contains(stderr, "404") {
		t.Errorf("stderr should name the HTTP status:\n%s", stderr)
	}
}

func TestRunDomain_TransportErrorIsNetworkUnavailable(t *testing.T) {
	tr := &cannedTransport{netErr: errors.New("dial tcp: connection refused")}
	seam := &fakeMeSeam{ctx: validMeContext(), transport: tr}
	outcome, _, stderr := runDomainOver(t, seam, domainConfig{})
	if outcome != NetworkUnavailable {
		t.Fatalf("outcome = %v, want NetworkUnavailable", outcome)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("a transport failure should be named on stderr")
	}
}

func TestRunDomain_NoTokenIsUsageError(t *testing.T) {
	seam := &fakeMeSeam{ctx: noTokenContext(), transport: &cannedTransport{status: 200, body: getDomainBody}}
	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{})
	if outcome != UsageError {
		t.Fatalf("outcome = %v, want UsageError\nstderr: %s", outcome, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("no data should print, got %q", stdout)
	}
	if !strings.Contains(strings.ToLower(stderr), "not authenticated") {
		t.Errorf("stderr should report not authenticated:\n%s", stderr)
	}
}

// --- structured output ------------------------------------------------------

func TestRunDomain_StructuredEmitsRawPayload(t *testing.T) {
	tr := &cannedTransport{status: 200, body: getDomainBody}
	seam := &fakeMeSeam{ctx: validMeContext(), envOutput: "json", transport: tr}
	outcome, stdout, stderr := runDomainOver(t, seam, domainConfig{})
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success\nstderr: %s", outcome, stderr)
	}
	for _, want := range []string{`"data"`, "dom_0123", "The marketing budget"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("structured output should emit the raw {data} payload; missing %q:\n%s", want, stdout)
		}
	}
}
