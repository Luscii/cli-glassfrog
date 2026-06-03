package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// RegistrationError is returned by Register — and is the value MustRegister
// panics with — when a command violates a registration rule. It names the
// offending command and the rule it broke so a failed registration is legible
// at startup. It is a named type so callers can discriminate registration
// failures from other runtime errors via errors.As.
type RegistrationError struct {
	// Command is the offending command's name (may be empty when the name
	// itself is the violation).
	Command string
	// Reason describes which rule was broken.
	Reason string
}

func (e *RegistrationError) Error() string {
	cmd := e.Command
	if strings.TrimSpace(cmd) == "" {
		cmd = "<empty>"
	}
	return fmt.Sprintf("command registration failed for %q: %s", cmd, e.Reason)
}

// Register validates child against the fail-loud registration rules and, only
// if it passes, attaches it under parent in the command tree. It is the single
// sanctioned path for building the command set — calling cobra's AddCommand
// directly bypasses these rules (see ADR-3).
//
// Rules, all surfaced before any user command runs:
//   - the name (cobra Use's first token) must be non-empty after trimming;
//   - the summary (Short) must be non-empty after trimming;
//   - a command is either a leaf (carries an action) or a group (carries at
//     least one child) — never both and never neither. A leaf without an
//     action and a group without children are the same observable state at
//     registration, so both are rejected by the one "neither" rule;
//   - the name must be unique among the parent's existing children.
//
// A group must therefore be assembled (its children registered into it) before
// it is itself registered under its parent, so the ">=1 child" rule holds at
// attach time.
func Register(parent, child *cobra.Command) error {
	if parent == nil || child == nil {
		return &RegistrationError{Reason: "parent and child must both be non-nil"}
	}

	name := strings.TrimSpace(child.Name())
	if name == "" {
		return &RegistrationError{Command: child.Name(), Reason: "name must not be empty or whitespace"}
	}
	if strings.TrimSpace(child.Short) == "" {
		return &RegistrationError{Command: name, Reason: "summary (Short) must not be empty or whitespace"}
	}

	hasChildren := len(child.Commands()) > 0
	hasAction := child.Run != nil || child.RunE != nil
	switch {
	case hasChildren && hasAction:
		return &RegistrationError{Command: name, Reason: "must be either a leaf (with an action) or a group (with children), not both"}
	case !hasChildren && !hasAction:
		return &RegistrationError{Command: name, Reason: "must be either a leaf with an action or a group with at least one child"}
	}

	for _, existing := range parent.Commands() {
		if existing.Name() == name {
			return &RegistrationError{
				Command: name,
				Reason:  fmt.Sprintf("name %q is already registered under %q", name, parent.Name()),
			}
		}
	}

	parent.AddCommand(child)
	return nil
}

// MustRegister calls Register and panics with the *RegistrationError on
// violation. The explicit wiring site (Assemble) uses it so a malformed
// registration aborts startup before cobra.Execute is ever called, leaving no
// partial command tree exposed to a user.
func MustRegister(parent, child *cobra.Command) {
	if err := Register(parent, child); err != nil {
		panic(err)
	}
}
