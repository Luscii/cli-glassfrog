package cli

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is the CLI's build-time-injected version. It is EMPTY by default —
// an empty value is the "not injected" sentinel that tells resolveVersion to
// fall back to Go's recorded module build info. A release build (and any
// GoReleaser build) injects it via
// -ldflags "-X github.com/Luscii/cli-glassfrog/internal/cli.version=vX.Y.Z"
// (the .goreleaser.yaml builds.ldflags seam, spec 023). Plain `go build` /
// `go install` apply no such flag, so version stays empty and the build-info
// fallback supplies the value.
var version string

// placeholderVersion is the last-resort value resolveVersion returns when
// neither an injected version nor usable module build info is available. It is
// non-empty and recognizably not a release, so version output is never blank
// and a development build is never mistaken for a published one (spec 023,
// ADR-3).
const placeholderVersion = "0.0.0-dev"

// resolveVersion computes the single version string the CLI reports, by a fixed
// three-tier precedence (spec 023, ADR-1):
//
//  1. an injected build-time version wins whenever present;
//  2. otherwise the version Go recorded in the binary's module build info is
//     used VERBATIM — a real tag (vX.Y.Z), a pseudo-version
//     (v0.0.0-<timestamp>-<commit>), or Go's "(devel)" marker all pass through
//     unchanged, never normalized;
//  3. otherwise the development placeholder.
//
// It is pure: it performs no I/O (no network, no VCS) and never returns an
// empty string. info/ok are the result of runtime/debug.ReadBuildInfo, threaded
// in so the precedence is unit-testable offline with crafted build info.
func resolveVersion(injected string, info *debug.BuildInfo, ok bool) string {
	if injected != "" {
		return injected
	}
	if ok && info != nil && info.Main.Version != "" {
		return info.Main.Version
	}
	return placeholderVersion
}

// resolvedVersion is the production wrapper: it reads the binary's module build
// info once and delegates to resolveVersion. It is the single value both the
// --version flag and the `version` command report, preserving Help & Version's
// (003) version-unify property. The build info is embedded by the toolchain at
// build time, so this still performs no runtime network or VCS lookup.
func resolvedVersion() string {
	info, ok := debug.ReadBuildInfo()
	return resolveVersion(version, info, ok)
}

// newVersionCommand returns the `version` leaf. It prints the resolved version
// (resolvedVersion) — the same value --version reports — keeping the two
// request forms byte-identical (003 ADR-3). Rendering (the bare-string form)
// stays Help & Version's concern; this command supplies the value only.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the glassfrog version",
		// version takes no positional arguments; reject any so unexpected input
		// is a usage error rather than silently ignored (dispatch's Invalid-input
		// accord, 002).
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), resolvedVersion())
			return nil
		},
	}
}
