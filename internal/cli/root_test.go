package cli

import (
	"bytes"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/apiclient"
	"github.com/spf13/cobra"
)

// The --base-url flag must be a persistent flag on the root, inherited by
// subcommands and retrievable from a subcommand's RunE (ADR-2). This is the
// mechanism every API command (the me command in T004, and 012–017) relies on
// to read the base URL and pass it to AssembleFromOS. Pin both the persistence
// and the inheritance: a leaf registered under the root reads the flag value its
// own RunE never declared.
func TestBaseURLIsPersistentRootFlagInheritedBySubcommands(t *testing.T) {
	root := NewRootCommand()

	// The flag is declared on the root's persistent set under the canonical name.
	if root.PersistentFlags().Lookup(apiclient.FlagBaseURL) == nil {
		t.Fatalf("--%s is not a persistent flag on the root", apiclient.FlagBaseURL)
	}

	var seen string
	leaf := &cobra.Command{
		Use:   "probe",
		Short: "a leaf that reads the inherited base-URL flag",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// A subcommand reads the inherited persistent flag from its own flag set.
			v, err := cmd.Flags().GetString(apiclient.FlagBaseURL)
			seen = v
			return err
		},
	}
	MustRegister(root, leaf)

	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	outcome, err := Run(root, []string{"probe", "--base-url", "https://example.test/api/v5"})
	if err != nil {
		t.Fatalf("running the probe leaf errored: %v", err)
	}
	if outcome != Success {
		t.Fatalf("outcome = %v, want Success", outcome)
	}
	if seen != "https://example.test/api/v5" {
		t.Fatalf("subcommand read --%s = %q, want the supplied value", apiclient.FlagBaseURL, seen)
	}
}
