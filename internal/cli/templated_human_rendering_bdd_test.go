package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/glassfrog"
	"github.com/Luscii/cli-glassfrog/internal/render"
	"github.com/cucumber/godog"
)

// TestTemplatedHumanRenderingFeatures runs the executable acceptance for
// Templated Human Rendering (019): the named full/compact templates exercised
// directly through internal/render, plus the two command-level guarantees (a
// failed read is not templated; a render failure leaves stdout empty and exits
// 1) driven through the rewired read commands over fakes — every scenario runs
// offline. Its Paths name ONLY this spec's feature file, so un-@wip-ping these
// scenarios cannot disturb another suite. The three @validation scenarios stay
// @wip (held for the validate skill) and are skipped by the ~@wip filter.
//
// full is rendered directly here too: it is the standing CLI output, so the
// command-level full path is additionally pinned by the per-read godog suites
// (identity-read / me-roles / me-actions / me-projects), which stay green as the
// no-regression gate. compact is reachable from no operator surface until 020,
// so it is verified only through render here (ADR-4).
func TestTemplatedHumanRenderingFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeRenderScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unconsumable-output/templated-human-rendering.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: templated-human-rendering feature scenarios failed")
	}
}

// renderWorld is the per-scenario state for the templated-human-rendering suite.
// A scenario is either render-direct (set data + resource in a Given, render in a
// When, assert the returned text) or command-driven (run a read over fakes,
// assert the captured streams/outcome).
type renderWorld struct {
	resource render.Resource
	data     any

	// rendered holds the text a "rendered with the full/compact template" When
	// produced; wantFull is the verbatim pre-019 projection a field-equivalence
	// "And" step compares against.
	rendered string
	wantFull string

	// command-driven streams/outcome.
	stdout   string
	stderr   string
	outcome  Outcome
	exitCode int
}

func initializeRenderScenario(sc *godog.ScenarioContext) {
	w := &renderWorld{}
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = renderWorld{}
		return ctx, nil
	})
	// renderFn is a package var the "render failure" scenario overrides; always
	// restore it so one scenario's injected defect can't leak into the next.
	sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		renderFn = render.Render
		return ctx, nil
	})

	// --- Givens ---
	sc.Step(`^a "me" read returned an actor, organization, and membership$`, w.givenMeIdentity)
	sc.Step(`^a "me --include roles" read whose response carried two roles$`, w.givenMeTwoRoles)
	sc.Step(`^a "me --include roles" read returned an actor filling three roles$`, w.givenMeThreeRoles)
	sc.Step(`^a "me" read without the roles embed$`, w.givenMeNoRoles)
	sc.Step(`^a "my roles" read returned three roles$`, w.givenThreeRoles)
	sc.Step(`^a "my projects" read returned zero projects$`, w.givenZeroProjects)
	sc.Step(`^a "my actions" read had failed with a transport error$`, w.givenActionsTransportError)
	sc.Step(`^a built-in template that fails to execute for a "me" result$`, w.givenFailingTemplate)

	// --- Whens ---
	sc.Step(`^the result is rendered with the full template$`, w.whenRenderFull)
	sc.Step(`^the result is rendered with the compact template$`, w.whenRenderCompact)
	sc.Step(`^the command reports the failure$`, w.whenActionsCommandReportsFailure)
	sc.Step(`^the command renders the result$`, w.whenMeCommandRenders)

	// --- Thens ---
	sc.Step(`^stdout will show the actor's id, name, and kind, the organization's id and name, and the access level$`, w.thenFullShowsIdentity)
	sc.Step(`^the output will match the projection the command produced before this feature$`, w.thenMatchesPreFeatureProjection)
	sc.Step(`^the identity fields will be rendered$`, w.thenIdentityFieldsRendered)
	sc.Step(`^each embedded role will be listed with its id and name$`, w.thenEmbeddedRolesListed)
	sc.Step(`^the rendered output will omit the roles section$`, w.thenRolesSectionOmitted)
	sc.Step(`^no empty roles heading will be printed$`, w.thenNoEmptyRolesHeading)
	sc.Step(`^stdout will show the explicit empty line "([^"]*)"$`, w.thenShowsEmptyLine)
	sc.Step(`^no fabricated project row will appear$`, w.thenNoFabricatedProjectRow)
	sc.Step(`^each role will appear on a single line$`, w.thenOneLinePerRole)
	sc.Step(`^each line will surface the role's id$`, w.thenEachLineSurfacesRoleID)
	sc.Step(`^the actor will appear on a single line showing "([^"]*)"$`, w.thenActorOneLineShowing)
	sc.Step(`^the line will surface the actor's id$`, w.thenLineSurfacesActorID)
	sc.Step(`^the error message will appear on stderr in its cause-plus-next-step form$`, w.thenErrorOnStderr)
	sc.Step(`^nothing will be written to stdout$`, w.thenNothingOnStdout)
	sc.Step(`^the command will exit with code (\d+)$`, w.thenExitCode)
}

// --- Given implementations -------------------------------------------------

// meIdentity is the fixed identity used by the field-equivalence scenario; its
// pre-019 projection is constructed verbatim so the "matches the projection"
// step is a byte comparison, not a field spot-check.
func meIdentity() glassfrog.MeResponse {
	return glassfrog.MeResponse{
		Actor:        glassfrog.Actor{ID: "per_alice", Name: "Alice Smith", Kind: "human"},
		Organization: glassfrog.Organization{ID: "org_acme", Name: "Acme"},
		Membership:   glassfrog.Membership{AccessLevel: "admin"},
	}
}

func (w *renderWorld) givenMeIdentity() error {
	w.resource = render.ResourceMe
	w.data = meIdentity()
	w.wantFull = "actor:        Alice Smith (human) per_alice\n" +
		"organization: Acme (org_acme)\n" +
		"access:       admin\n"
	return nil
}

func (w *renderWorld) givenMeTwoRoles() error {
	me := meIdentity()
	me.Roles = []glassfrog.Role{
		{ID: "role_1", Name: "Marketing Lead"},
		{ID: "role_2", Name: "Treasurer"},
	}
	w.resource, w.data = render.ResourceMe, me
	return nil
}

func (w *renderWorld) givenMeThreeRoles() error {
	me := meIdentity()
	me.Roles = []glassfrog.Role{
		{ID: "role_1", Name: "Lead"},
		{ID: "role_2", Name: "Rep"},
		{ID: "role_3", Name: "Treasurer"},
	}
	w.resource, w.data = render.ResourceMe, me
	return nil
}

func (w *renderWorld) givenMeNoRoles() error {
	w.resource, w.data = render.ResourceMe, meIdentity()
	return nil
}

func (w *renderWorld) givenThreeRoles() error {
	w.resource = render.ResourceRoles
	w.data = glassfrog.MyRolesResponse{Data: []glassfrog.Role{
		{ID: "role_1", Name: "Lead"},
		{ID: "role_2", Name: "Rep"},
		{ID: "role_3", Name: "Treasurer"},
	}}
	return nil
}

func (w *renderWorld) givenZeroProjects() error {
	w.resource, w.data = render.ResourceProjects, glassfrog.MyProjectsResponse{}
	return nil
}

// givenActionsTransportError stages the failed `me actions` read: a fake base
// transport that returns a wire error, so runMeActions classifies it and reports
// the cause-plus-next-step message — nothing is rendered.
func (w *renderWorld) givenActionsTransportError() error {
	w.resource = render.ResourceActions
	w.data = &fakeMeSeam{ctx: validMeContext(), transport: &cannedTransport{netErr: errors.New("dial tcp: connection refused")}}
	return nil
}

// givenFailingTemplate overrides the render seam so a valid `me` result still
// fails to render — standing in for a built-in-template defect (the typed result
// structs otherwise always render, so this is the only way to reach the
// buffer-then-write failure path).
func (w *renderWorld) givenFailingTemplate() error {
	w.resource = render.ResourceMe
	w.data = &fakeMeSeam{ctx: validMeContext(), transport: &cannedTransport{status: 200, body: meBodyAlice}}
	renderFn = func(render.Resource, render.Format, any) (string, error) {
		return "", &render.RenderError{Resource: render.ResourceMe, Format: render.FormatFull, Err: errors.New("template defect")}
	}
	return nil
}

// --- When implementations --------------------------------------------------

func (w *renderWorld) whenRenderFull() error {
	out, err := render.Render(w.resource, render.FormatFull, w.data)
	if err != nil {
		return fmt.Errorf("full render failed: %v", err)
	}
	w.rendered = out
	return nil
}

func (w *renderWorld) whenRenderCompact() error {
	out, err := render.Render(w.resource, render.FormatCompact, w.data)
	if err != nil {
		return fmt.Errorf("compact render failed: %v", err)
	}
	w.rendered = out
	return nil
}

func (w *renderWorld) whenActionsCommandReportsFailure() error {
	seam := w.data.(*fakeMeSeam)
	var out, errb bytes.Buffer
	w.outcome, _ = runMeActions(meActionsConfig{
		seam:   seam,
		reqCtx: context.Background(),
		stdout: &out,
		stderr: &errb,
	})
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()
	return nil
}

func (w *renderWorld) whenMeCommandRenders() error {
	seam := w.data.(*fakeMeSeam)
	var out, errb bytes.Buffer
	w.outcome, _ = runMe(meConfig{
		seam:   seam,
		reqCtx: context.Background(),
		stdout: &out,
		stderr: &errb,
	})
	w.exitCode = ExitCode(w.outcome)
	w.stdout, w.stderr = out.String(), errb.String()
	return nil
}

// --- Then implementations --------------------------------------------------

func (w *renderWorld) thenFullShowsIdentity() error {
	for _, want := range []string{"per_alice", "Alice Smith", "(human)", "org_acme", "Acme", "admin"} {
		if !strings.Contains(w.rendered, want) {
			return fmt.Errorf("full output missing %q:\n%s", want, w.rendered)
		}
	}
	return nil
}

func (w *renderWorld) thenMatchesPreFeatureProjection() error {
	if w.rendered != w.wantFull {
		return fmt.Errorf("full output is not byte-equivalent to the pre-feature projection:\n--- got ---\n%q\n--- want ---\n%q", w.rendered, w.wantFull)
	}
	return nil
}

func (w *renderWorld) thenIdentityFieldsRendered() error {
	for _, want := range []string{"actor:", "organization:", "access:"} {
		if !strings.Contains(w.rendered, want) {
			return fmt.Errorf("identity fields missing %q:\n%s", want, w.rendered)
		}
	}
	return nil
}

func (w *renderWorld) thenEmbeddedRolesListed() error {
	if !strings.Contains(w.rendered, "roles:") {
		return fmt.Errorf("full output should enumerate the embedded roles:\n%s", w.rendered)
	}
	for _, want := range []string{"Marketing Lead", "role_1", "Treasurer", "role_2"} {
		if !strings.Contains(w.rendered, want) {
			return fmt.Errorf("embedded role missing %q (id and name):\n%s", want, w.rendered)
		}
	}
	return nil
}

func (w *renderWorld) thenRolesSectionOmitted() error {
	if strings.Contains(w.rendered, "roles:") {
		return fmt.Errorf("an absent roles embed should omit the roles section:\n%s", w.rendered)
	}
	return nil
}

func (w *renderWorld) thenNoEmptyRolesHeading() error {
	// Same guarantee as omission, phrased from the heading's side: no roles
	// heading of any form may appear when the embed is absent.
	if strings.Contains(strings.ToLower(w.rendered), "roles") {
		return fmt.Errorf("no roles heading should be printed for an absent embed:\n%s", w.rendered)
	}
	return nil
}

func (w *renderWorld) thenShowsEmptyLine(line string) error {
	if strings.TrimRight(w.rendered, "\n") != line {
		return fmt.Errorf("expected the explicit empty line %q, got:\n%q", line, w.rendered)
	}
	return nil
}

func (w *renderWorld) thenNoFabricatedProjectRow() error {
	// A fabricated row would carry a proj_ id; the empty line must not.
	if strings.Contains(w.rendered, "proj_") || strings.Contains(w.rendered, "[") {
		return fmt.Errorf("an empty result must not print a fabricated project row:\n%s", w.rendered)
	}
	return nil
}

func (w *renderWorld) thenOneLinePerRole() error {
	lines := nonEmptyLines(w.rendered)
	if len(lines) != 3 {
		return fmt.Errorf("compact should render one line per role (3), got %d:\n%s", len(lines), w.rendered)
	}
	return nil
}

func (w *renderWorld) thenEachLineSurfacesRoleID() error {
	for _, line := range nonEmptyLines(w.rendered) {
		if !strings.Contains(line, "role_") {
			return fmt.Errorf("every compact role line must surface the role id, missing on %q", line)
		}
	}
	return nil
}

func (w *renderWorld) thenActorOneLineShowing(token string) error {
	lines := nonEmptyLines(w.rendered)
	if len(lines) != 1 {
		return fmt.Errorf("compact me should be a single line, got %d:\n%s", len(lines), w.rendered)
	}
	if !strings.Contains(w.rendered, token) {
		return fmt.Errorf("compact me line should show %q (the nested collection as a count):\n%s", token, w.rendered)
	}
	return nil
}

func (w *renderWorld) thenLineSurfacesActorID() error {
	if !strings.Contains(w.rendered, "per_") && !strings.Contains(w.rendered, "agt_") {
		return fmt.Errorf("the compact me line must surface the actor id:\n%s", w.rendered)
	}
	return nil
}

func (w *renderWorld) thenErrorOnStderr() error {
	if strings.TrimSpace(w.stderr) == "" {
		return errors.New("the failure should be reported on stderr in cause-plus-next-step form")
	}
	if w.outcome == Success {
		return fmt.Errorf("a failed read must not exit successfully, got %v", w.outcome)
	}
	return nil
}

func (w *renderWorld) thenNothingOnStdout() error {
	if strings.TrimSpace(w.stdout) != "" {
		return fmt.Errorf("nothing should be written to stdout, got:\n%s", w.stdout)
	}
	return nil
}

func (w *renderWorld) thenExitCode(code int) error {
	if w.exitCode != code {
		return fmt.Errorf("expected exit code %d, got %d (outcome %v)\nstderr: %s", code, w.exitCode, w.outcome, w.stderr)
	}
	return nil
}

// nonEmptyLines splits rendered text into its non-blank lines, so a one-line-per-
// record assertion ignores the trailing newline.
func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}
