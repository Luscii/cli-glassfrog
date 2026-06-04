package cli

import (
	"strings"
	"testing"
)

func TestReadBoundedStdin_WithinLimit(t *testing.T) {
	got, err := readBoundedStdin(strings.NewReader("gf_token\n"))
	if err != nil {
		t.Fatalf("readBoundedStdin: %v", err)
	}
	if got != "gf_token\n" {
		t.Fatalf("got %q, want %q", got, "gf_token\n")
	}
}

func TestReadBoundedStdin_OverLimit_Errors(t *testing.T) {
	big := strings.Repeat("a", maxPipedTokenBytes+10)
	_, err := readBoundedStdin(strings.NewReader(big))
	if err == nil {
		t.Fatal("expected an error for input exceeding the limit, got nil")
	}
}

func TestReadBoundedStdin_AtLimit_OK(t *testing.T) {
	exact := strings.Repeat("a", maxPipedTokenBytes)
	got, err := readBoundedStdin(strings.NewReader(exact))
	if err != nil {
		t.Fatalf("input exactly at the limit should be accepted, got %v", err)
	}
	if len(got) != maxPipedTokenBytes {
		t.Fatalf("got %d bytes, want %d", len(got), maxPipedTokenBytes)
	}
}

// gatherInputsFrom is the pure core of the production input seam; these tests
// pin its precedence-shaping decisions (when stdin is and is not read) without
// touching the real stdin/TTY/env. The terminal-bound production paths
// (term.ReadPassword, confirm, choose) are exercised through the BDD seam.

func TestGatherInputsFrom_ArgumentSkipsStdinRead(t *testing.T) {
	read := func() (string, error) {
		t.Fatal("stdin must not be read when an argument is supplied")
		return "", nil
	}
	in, err := gatherInputsFrom([]string{"arg_tok"}, "env_tok", false, read)
	if err != nil {
		t.Fatalf("gatherInputsFrom: %v", err)
	}
	if !in.argGiven || in.arg != "arg_tok" {
		t.Fatalf("argument not captured: %+v", in)
	}
	if in.stdinPiped {
		t.Fatal("stdin should not be marked piped when an argument is present")
	}
	if in.env != "env_tok" {
		t.Fatalf("env not captured: %+v", in)
	}
}

func TestGatherInputsFrom_TTYSkipsStdinRead(t *testing.T) {
	read := func() (string, error) {
		t.Fatal("stdin must not be read when stdin is a terminal")
		return "", nil
	}
	in, err := gatherInputsFrom(nil, "", true, read)
	if err != nil {
		t.Fatalf("gatherInputsFrom: %v", err)
	}
	if in.stdinPiped {
		t.Fatal("a TTY session must not read piped stdin")
	}
	if !in.isTTY {
		t.Fatal("isTTY should be propagated")
	}
}

func TestGatherInputsFrom_NonTTYNoArgReadsStdin(t *testing.T) {
	read := func() (string, error) { return "piped_tok\n", nil }
	in, err := gatherInputsFrom(nil, "", false, read)
	if err != nil {
		t.Fatalf("gatherInputsFrom: %v", err)
	}
	if !in.stdinPiped || in.stdin != "piped_tok\n" {
		t.Fatalf("piped stdin not captured: %+v", in)
	}
}
