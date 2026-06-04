package cli

import "testing"

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
