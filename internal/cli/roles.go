package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newRolesCommand assembles the `roles` group and its subcommands, then returns
// it for registration under the root. The group is fully assembled (its
// children registered through the guard) before it is itself registered, so
// the guard's ">=1 child" rule holds when the group attaches to its parent.
//
// The subcommand actions are honest stubs: this feature exercises command
// registration, not the Glassfrog API. The real governance-read behavior is
// the Governance Reads capability's concern (a later spec), so the stubs make
// no claim about API operations (Constitution I/VIII).
func newRolesCommand() *cobra.Command {
	roles := &cobra.Command{
		Use:   "roles",
		Short: "Read roles and the governance around them",
	}
	MustRegister(roles, newRolesListCommand())
	MustRegister(roles, newRolesGetCommand())
	return roles
}

func newRolesListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List roles",
		// No positional arguments today; reject unexpected input as a usage
		// error rather than ignoring it (dispatch's Invalid-input accord, 002).
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "roles list is not yet implemented")
			return nil
		},
	}
}

func newRolesGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show one role",
		// No positional arguments today; reject unexpected input as a usage
		// error rather than ignoring it (dispatch's Invalid-input accord, 002).
		// The role-id argument arrives with the Governance Reads capability.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "roles get is not yet implemented")
			return nil
		},
	}
}
