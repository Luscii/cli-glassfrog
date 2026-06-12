package cli

import (
	"strings"
	"testing"
)

// --- validateTensionStatus (pure, T002) ------------------------------------
//
// A NEW validator over the tension status set (unprocessed/processed/archived),
// distinct from the action/project validateStatus set — reusing that would accept
// invalid tension statuses and reject valid ones (plan ADR-3).

func TestValidateTensionStatus(t *testing.T) {
	if err := validateTensionStatus(""); err != nil {
		t.Errorf("an absent --status should be valid (no filter), got %v", err)
	}
	for _, ok := range []string{"unprocessed", "processed", "archived"} {
		if err := validateTensionStatus(ok); err != nil {
			t.Errorf("%q should be valid, got %v", ok, err)
		}
	}
	err := validateTensionStatus("open")
	if err == nil {
		t.Fatal("an unsupported --status should be rejected")
	}
	if !strings.Contains(err.Error(), "open") {
		t.Errorf("the error should name the unsupported value:\n%v", err)
	}
	for _, want := range []string{"archived", "processed", "unprocessed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error should list the supported set (missing %q):\n%v", want, err)
		}
	}
}

// TestValidateTensionStatus_RejectsActionVocabulary pins that the action/project
// statuses are NOT accepted here — the new set is distinct (plan ADR-3).
func TestValidateTensionStatus_RejectsActionVocabulary(t *testing.T) {
	for _, wrong := range []string{"current", "completed"} {
		if err := validateTensionStatus(wrong); err == nil {
			t.Errorf("%q is an action/project status and must be rejected for tensions", wrong)
		}
	}
}

// TestSupportedTensionStatusNames_Sorted pins the deterministic sorted order used
// in the usage message.
func TestSupportedTensionStatusNames_Sorted(t *testing.T) {
	got := supportedTensionStatusNames()
	want := []string{"archived", "processed", "unprocessed"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (sorted)", got, want)
		}
	}
}
