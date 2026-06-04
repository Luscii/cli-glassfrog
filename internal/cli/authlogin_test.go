package cli

import (
	"path/filepath"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// --- resolveTokenSource: precedence (argument → stdin → env → prompt) ---

func TestResolveTokenSource_Precedence(t *testing.T) {
	cases := []struct {
		name       string
		in         tokenInputs
		wantRaw    string
		wantSource tokenSource
	}{
		{
			name:       "argument beats everything",
			in:         tokenInputs{arg: "arg_tok", argGiven: true, stdin: "stdin_tok", stdinPiped: true, env: "env_tok", isTTY: false},
			wantRaw:    "arg_tok",
			wantSource: tokenFromArg,
		},
		{
			name:       "piped stdin beats env and prompt",
			in:         tokenInputs{stdin: "stdin_tok", stdinPiped: true, env: "env_tok", isTTY: true},
			wantRaw:    "stdin_tok",
			wantSource: tokenFromStdin,
		},
		{
			name:       "env beats prompt",
			in:         tokenInputs{env: "env_tok", isTTY: true},
			wantRaw:    "env_tok",
			wantSource: tokenFromEnv,
		},
		{
			name:       "prompt when interactive and no other source",
			in:         tokenInputs{isTTY: true},
			wantRaw:    "",
			wantSource: tokenNeedsPrompt,
		},
		{
			name:       "no token when non-interactive and no source",
			in:         tokenInputs{isTTY: false},
			wantRaw:    "",
			wantSource: tokenNone,
		},
		{
			name:       "empty pipe falls through to env",
			in:         tokenInputs{stdin: "   ", stdinPiped: true, env: "env_tok"},
			wantRaw:    "env_tok",
			wantSource: tokenFromEnv,
		},
		{
			name:       "empty env falls through to prompt",
			in:         tokenInputs{env: "   ", isTTY: true},
			wantRaw:    "",
			wantSource: tokenNeedsPrompt,
		},
		{
			name:       "blank argument is still the selected source (rejected downstream)",
			in:         tokenInputs{arg: "   ", argGiven: true, env: "env_tok"},
			wantRaw:    "   ",
			wantSource: tokenFromArg,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, src := resolveTokenSource(tc.in)
			if raw != tc.wantRaw || src != tc.wantSource {
				t.Fatalf("resolveTokenSource = (%q, %v), want (%q, %v)", raw, src, tc.wantRaw, tc.wantSource)
			}
		})
	}
}

// --- usableToken: blank rejection ---

func TestUsableToken(t *testing.T) {
	cases := []struct {
		raw      string
		wantTok  string
		wantUsed bool
	}{
		{"gf_tok", "gf_tok", true},
		{"  gf_padded  ", "gf_padded", true},
		{"", "", false},
		{"   ", "", false},
		{"\t\n ", "", false},
		// Embedded line breaks are rejected — writing one would split into
		// multiple .glassfrogrc lines and could inject extra keys.
		{"gf_a\ngf_b", "gf_a\ngf_b", false},
		{"gf_a\r\nother=x", "gf_a\r\nother=x", false},
		{"line1\rline2", "line1\rline2", false},
	}
	for _, tc := range cases {
		tok, ok := usableToken(tc.raw)
		if tok != tc.wantTok || ok != tc.wantUsed {
			t.Fatalf("usableToken(%q) = (%q, %v), want (%q, %v)", tc.raw, tok, ok, tc.wantTok, tc.wantUsed)
		}
	}
}

// --- targetPath: home default, current directory under --cwd ---

func TestTargetPath(t *testing.T) {
	home := filepath.Join("home", "user")
	start := filepath.Join("work", "project")

	if got, want := targetPath(home, start, false), filepath.Join(home, auth.CredentialsFileName); got != want {
		t.Fatalf("default target = %q, want %q", got, want)
	}
	if got, want := targetPath(home, start, true), filepath.Join(start, auth.CredentialsFileName); got != want {
		t.Fatalf("--cwd target = %q, want %q", got, want)
	}
}

// --- existingTokenGuard: overwrite / interactivity branches ---

func TestExistingTokenGuard(t *testing.T) {
	cases := []struct {
		name        string
		hasExisting bool
		isTTY       bool
		overwrite   bool
		want        guardDecision
	}{
		{"no existing token proceeds", false, false, false, guardProceed},
		{"no existing token proceeds even interactive", false, true, false, guardProceed},
		{"non-interactive existing token without overwrite is blocked", true, false, false, guardBlocked},
		{"non-interactive existing token with overwrite proceeds", true, false, true, guardProceed},
		{"interactive existing token defers to confirm/choose", true, true, false, guardInteractive},
		{"interactive existing token ignores overwrite flag", true, true, true, guardInteractive},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := existingTokenGuard(tc.hasExisting, tc.isTTY, tc.overwrite); got != tc.want {
				t.Fatalf("existingTokenGuard(%v,%v,%v) = %v, want %v", tc.hasExisting, tc.isTTY, tc.overwrite, got, tc.want)
			}
		})
	}
}
