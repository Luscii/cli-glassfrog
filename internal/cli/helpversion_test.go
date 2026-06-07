package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/spf13/cobra"
)

// runAssembled executes args against a freshly assembled root with the package
// version var pinned to ver and output captured. It returns the combined
// stdout+stderr and the dispatch outcome. Help and version are success
// outputs; the outcome lets a test assert built-ins do not resolve.
func runAssembled(t *testing.T, ver string, args ...string) (string, Outcome) {
	t.Helper()
	saved := version
	version = ver
	t.Cleanup(func() { version = saved })

	root := Assemble()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	outcome, _ := Run(root, args)
	return buf.String(), outcome
}

// availableCommands extracts the command names listed under cobra's
// "Available Commands:" section of a help listing, in the order they appear.
// This is the listing the spec governs — distinct from the word "help" that
// appears in the flags section.
func availableCommands(helpOutput string) []string {
	var cmds []string
	inSection := false
	for _, ln := range strings.Split(helpOutput, "\n") {
		if strings.HasPrefix(ln, "Available Commands:") {
			inSection = true
			continue
		}
		if inSection {
			if strings.TrimSpace(ln) == "" {
				break
			}
			if fields := strings.Fields(ln); len(fields) > 0 {
				cmds = append(cmds, fields[0])
			}
		}
	}
	return cmds
}

// ADR-3: the --version flag and the version command emit byte-identical output
// — the bare version value, never cobra's default "glassfrog version X" form.
func TestVersionFlagAndCommandParity(t *testing.T) {
	flagOut, _ := runAssembled(t, "1.2.0", "--version")
	cmdOut, _ := runAssembled(t, "1.2.0", "version")

	if flagOut != cmdOut {
		t.Fatalf("--version (%q) and `version` command (%q) must be byte-identical", flagOut, cmdOut)
	}
	if !strings.Contains(flagOut, "1.2.0") {
		t.Fatalf("version output %q does not contain the build version", flagOut)
	}
	if strings.Contains(flagOut, "glassfrog version") {
		t.Fatalf("version output %q carries cobra's default Name-version prefix", flagOut)
	}
}

// The version-unset default is a clear, non-empty placeholder — never empty.
func TestVersionDefaultPlaceholder(t *testing.T) {
	if strings.TrimSpace(version) == "" {
		t.Fatal("default version must be a non-empty placeholder")
	}
	if version != "0.0.0-dev" {
		t.Fatalf("default version = %q, want 0.0.0-dev", version)
	}
}

// ADR-1/ADR-2: --help renders the listing; the listing shows only the
// guard-registered commands (no built-in help/completion) in alphabetical
// order.
func TestListingExcludesBuiltinsAlphabetically(t *testing.T) {
	out, _ := runAssembled(t, "0.0.0-dev", "--help")
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("--help did not render usage/listing: %q", out)
	}
	cmds := availableCommands(out)
	// Assert the invariants the spec cares about, not the exact command set, so
	// adding a legitimate command later does not break this test:
	//   - the framework built-ins are absent from the listing (ADR-2), and
	//   - known commands appear in alphabetical relative order (ADR-1).
	for _, builtin := range []string{"help", "completion"} {
		if containsString(cmds, builtin) {
			t.Fatalf("listing must not include the built-in %q command; got %v", builtin, cmds)
		}
	}
	ri, vi := indexOf(cmds, "roles"), indexOf(cmds, "version")
	if ri < 0 || vi < 0 {
		t.Fatalf("expected `roles` and `version` listed, got %v", cmds)
	}
	if ri >= vi {
		t.Fatalf("`roles` should be listed before `version` (alphabetical), got %v", cmds)
	}
}

// ADR-2: the built-in help and completion tokens must not resolve as commands —
// hiding alone is insufficient; `glassfrog help` / `glassfrog completion` are
// unknown commands.
func TestBuiltinCommandsDoNotResolve(t *testing.T) {
	for _, tok := range []string{"help", "completion"} {
		_, outcome := runAssembled(t, "0.0.0-dev", tok)
		if outcome != UsageError {
			t.Fatalf("glassfrog %s: outcome = %v, want UsageError (built-in must not resolve)", tok, outcome)
		}
	}
}

// The --help flag must still render even though the help command is suppressed.
func TestHelpFlagStillRenders(t *testing.T) {
	out, outcome := runAssembled(t, "0.0.0-dev", "--help")
	if outcome != Success {
		t.Fatalf("--help outcome = %v, want Success", outcome)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("--help did not render usage: %q", out)
	}
}

// Risk pin: alphabetical ordering depends on a cobra package-global that
// defaults true. A regression test fails loudly if any future code flips it.
func TestCommandSortingEnabled(t *testing.T) {
	if !cobra.EnableCommandSorting {
		t.Fatal("cobra.EnableCommandSorting must stay true for alphabetical listings (ADR-1)")
	}
}

// Identity Read (011, ADR-2) adds the persistent --base-url flag to the root, so
// root help now carries a global-flags section naming it. 003's narrowed
// non-behavior permits this: the flag is optional documentation data, not new
// required data. Pinning its presence keeps the ADR-2 decision honest — removing
// the global flag (or renaming it away from apiclient.FlagBaseURL) fails here.
func TestRootHelpShowsBaseURLGlobalFlag(t *testing.T) {
	out, outcome := runAssembled(t, "0.0.0-dev", "--help")
	if outcome != Success {
		t.Fatalf("--help outcome = %v, want Success", outcome)
	}
	if !strings.Contains(out, "--"+apiclient.FlagBaseURL) {
		t.Fatalf("root help should document the global --%s flag, got:\n%s", apiclient.FlagBaseURL, out)
	}
}

// Precedence: when both --help and --version are supplied, help wins.
func TestHelpPrecedesVersion(t *testing.T) {
	out, _ := runAssembled(t, "1.2.0", "--help", "--version")
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("--help --version did not produce help output: %q", out)
	}
	if strings.TrimSpace(out) == "1.2.0" {
		t.Fatalf("--help --version produced version output, not help: %q", out)
	}
}
