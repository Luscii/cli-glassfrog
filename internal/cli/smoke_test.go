package cli

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// smokeModeEnv selects what the test binary does when re-executed as a
// subprocess child by TestSmoke_SubprocessExitCodes (below). It is unset for a
// normal `go test` run, so the package's tests run as usual.
const smokeModeEnv = "GLASSFROG_SMOKE_MODE"

// TestMain lets the test binary re-execute itself as a subprocess child so a
// test can observe a real process exit status — something in-process tests
// cannot see because os.Exit would terminate the test runner. The child
// reconstructs the same os.Exit(<exit code>) composition that main.go performs;
// it does not run main.go itself (the test binary has no access to it).
//
//   - mode "real": reproduce main.go's one line, os.Exit(Main()), against the
//     assembled tree and the forwarded args — the closest faithful stand-in.
//   - mode "fixture": the real tree has no failing or panicking command, so the
//     RuntimeError→1 and panic→1 paths are exercised through the same production
//     core (os.Exit(runToExitCode(...))) against the codeTree fixture.
//
// With the env unset, TestMain just runs the package's tests.
func TestMain(m *testing.M) {
	switch os.Getenv(smokeModeEnv) {
	case "":
		os.Exit(m.Run())
	case "real":
		os.Exit(Main())
	case "fixture":
		os.Exit(runToExitCode(codeTree(), os.Args[1:]))
	default:
		os.Stderr.WriteString("unknown " + smokeModeEnv + "\n")
		os.Exit(127)
	}
}

// TestSmoke_SubprocessExitCodes confirms the os.Exit(Main()) composition exits
// with the mapped code across a real process boundary, including the panic→1
// path: a panic must surface as exit 1, never Go's default panic status 2
// (which would collide with the usage code). The subprocess is the test binary
// re-executed via TestMain, not the compiled main.go, but it runs the identical
// os.Exit(Main()) call ("real" cases, exercising Main()/Assemble()); the failure
// cases run a fixture tree through the same recover+map core because the
// assembled tree has no command that fails or panics.
func TestSmoke_SubprocessExitCodes(t *testing.T) {
	cases := []struct {
		name string
		mode string
		args []string
		want int
	}{
		{"successful command", "real", []string{"version"}, 0},
		{"bare group help", "real", []string{"roles"}, 0},
		{"unknown command", "real", []string{"nope"}, 2},
		{"runtime failure", "fixture", []string{"boom"}, 1},
		{"panic recovers to 1", "fixture", []string{"panicker"}, 1},
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locate test binary: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(exe, tc.args...)
			cmd.Env = append(os.Environ(), smokeModeEnv+"="+tc.mode)
			// Discard the child's stdout/stderr (command output and any panic
			// diagnostic) — the assertion is purely on the process exit status.
			cmd.Stdout, cmd.Stderr = nil, nil

			got := 0
			if runErr := cmd.Run(); runErr != nil {
				var exitErr *exec.ExitError
				if !errors.As(runErr, &exitErr) {
					t.Fatalf("running child: %v", runErr)
				}
				got = exitErr.ExitCode()
			}
			if got != tc.want {
				t.Fatalf("child exit code = %d, want %d", got, tc.want)
			}
		})
	}
}
