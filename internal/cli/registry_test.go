package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// leaf builds a valid leaf command (name + summary + action).
func leaf(name, summary string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: summary,
		RunE:  func(*cobra.Command, []string) error { return nil },
	}
}

func TestRegister_LeafSucceeds(t *testing.T) {
	root := NewRootCommand()
	v := leaf("version", "Print the version")

	if err := Register(root, v); err != nil {
		t.Fatalf("Register returned error for a valid leaf: %v", err)
	}

	got, _, err := root.Find([]string{"version"})
	if err != nil || got.Name() != "version" {
		t.Fatalf("registered leaf not found by path: got %v, err %v", got, err)
	}
}

func TestRegister_GroupWithChildrenSucceeds(t *testing.T) {
	root := NewRootCommand()
	group := &cobra.Command{Use: "roles", Short: "Read roles"}
	// A group must be assembled before it is registered under its parent.
	if err := Register(group, leaf("list", "List roles")); err != nil {
		t.Fatalf("registering child into group failed: %v", err)
	}
	if err := Register(group, leaf("get", "Show one role")); err != nil {
		t.Fatalf("registering second child into group failed: %v", err)
	}
	if err := Register(root, group); err != nil {
		t.Fatalf("registering assembled group failed: %v", err)
	}

	if got, _, err := root.Find([]string{"roles", "list"}); err != nil || got.Name() != "list" {
		t.Fatalf("nested path 'roles list' did not resolve: got %v, err %v", got, err)
	}
	// A bare group name resolves to the group node itself.
	if got, _, err := root.Find([]string{"roles"}); err != nil || got.Name() != "roles" {
		t.Fatalf("bare group 'roles' did not resolve to the group: got %v, err %v", got, err)
	}
}

func TestRegister_DuplicateSiblingNameRejected(t *testing.T) {
	root := NewRootCommand()
	if err := Register(root, leaf("roles", "Read roles")); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err := Register(root, leaf("roles", "Read roles again"))
	assertRegistrationError(t, err, "roles")
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("duplicate error should mention the collision, got: %v", err)
	}
}

func TestRegister_EmptyNameRejected(t *testing.T) {
	root := NewRootCommand()
	for _, name := range []string{"", "   "} {
		cmd := &cobra.Command{Use: name, Short: "has a summary", RunE: func(*cobra.Command, []string) error { return nil }}
		err := Register(root, cmd)
		if err == nil {
			t.Fatalf("expected error for name %q, got nil", name)
		}
		var re *RegistrationError
		if !errors.As(err, &re) {
			t.Fatalf("expected *RegistrationError for name %q, got %T", name, err)
		}
	}
}

func TestRegister_EmptySummaryRejected(t *testing.T) {
	root := NewRootCommand()
	cmd := &cobra.Command{Use: "version", Short: "   ", RunE: func(*cobra.Command, []string) error { return nil }}
	assertRegistrationError(t, Register(root, cmd), "version")
}

func TestRegister_LeafWithoutActionRejected(t *testing.T) {
	root := NewRootCommand()
	// No children, no Run/RunE.
	cmd := &cobra.Command{Use: "version", Short: "Print the version"}
	assertRegistrationError(t, Register(root, cmd), "version")
}

func TestRegister_GroupWithoutChildrenRejected(t *testing.T) {
	root := NewRootCommand()
	// A group is identified by having children; an empty one is the same
	// observable state as an actionless leaf and is rejected.
	cmd := &cobra.Command{Use: "roles", Short: "Read roles"}
	assertRegistrationError(t, Register(root, cmd), "roles")
}

func TestRegister_SameNameUnderDifferentParentsAllowed(t *testing.T) {
	root := NewRootCommand()
	roles := &cobra.Command{Use: "roles", Short: "Read roles"}
	proposals := &cobra.Command{Use: "proposals", Short: "Manage proposals"}
	if err := Register(roles, leaf("get", "Show one role")); err != nil {
		t.Fatalf("register roles get: %v", err)
	}
	if err := Register(proposals, leaf("get", "Show one proposal")); err != nil {
		t.Fatalf("'get' under a different parent must be allowed: %v", err)
	}
	if err := Register(root, roles); err != nil {
		t.Fatalf("register roles: %v", err)
	}
	if err := Register(root, proposals); err != nil {
		t.Fatalf("register proposals: %v", err)
	}

	r, _, _ := root.Find([]string{"roles", "get"})
	p, _, _ := root.Find([]string{"proposals", "get"})
	if r == p {
		t.Fatal("'roles get' and 'proposals get' must resolve independently")
	}
}

func TestMustRegister_PanicsOnViolation(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustRegister should panic on an invalid registration")
		}
		if _, ok := r.(*RegistrationError); !ok {
			t.Fatalf("panic value should be *RegistrationError, got %T", r)
		}
	}()
	root := NewRootCommand()
	MustRegister(root, &cobra.Command{Use: "broken", Short: ""}) // empty summary
}

func TestMustRegister_DoesNotPanicOnValid(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MustRegister panicked on a valid command: %v", r)
		}
	}()
	root := NewRootCommand()
	MustRegister(root, leaf("version", "Print the version"))
}

// assertRegistrationError fails the test unless err is a *RegistrationError
// naming the expected command.
func assertRegistrationError(t *testing.T, err error, wantCommand string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a RegistrationError naming %q, got nil", wantCommand)
	}
	var re *RegistrationError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RegistrationError, got %T: %v", err, err)
	}
	if re.Command != wantCommand {
		t.Fatalf("error should name command %q, named %q (full: %v)", wantCommand, re.Command, err)
	}
}
