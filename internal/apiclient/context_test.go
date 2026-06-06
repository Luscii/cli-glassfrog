package apiclient

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
	"github.com/Luscii/cli-glassfrog/internal/rcfile"
)

// completeContext is a small helper building a ConnectionContext whose both
// halves resolved cleanly — a file-sourced base URL and a file-sourced token.
func completeContext(token string) ConnectionContext {
	return ConnectionContext{
		BaseURL: BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceFile, Path: "/home/me/.glassfrogrc"},
		Cred:    auth.Resolution{Token: token, Source: auth.SourceFile, Path: "/home/me/.glassfrogrc"},
	}
}

func TestConnectionContextComplete(t *testing.T) {
	tests := []struct {
		name string
		ctx  ConnectionContext
		want bool
	}{
		{
			name: "complete: usable base URL and present token, no errors",
			ctx:  completeContext("gf_live_secret123"),
			want: true,
		},
		{
			name: "credential absent: base URL present, Source None",
			ctx: ConnectionContext{
				BaseURL: BaseURL{Value: DefaultBaseURL, Source: SourceDefault},
				Cred:    auth.Resolution{Source: auth.SourceNone},
			},
			want: false,
		},
		{
			name: "base-URL error carried, token present",
			ctx: ConnectionContext{
				BaseURLErr: &BaseURLError{Source: "--" + FlagBaseURL},
				Cred:       auth.Resolution{Token: "gf_x", Source: auth.SourceEnvironment},
			},
			want: false,
		},
		{
			name: "credential error carried, base URL present",
			ctx: ConnectionContext{
				BaseURL: BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceFile, Path: "/etc/.glassfrogrc"},
				CredErr: &rcfile.ReadError{Path: "/etc/.glassfrogrc", Err: errors.New("permission denied")},
			},
			want: false,
		},
		{
			name: "both errored",
			ctx: ConnectionContext{
				BaseURLErr: &BaseURLError{Source: "--" + FlagBaseURL},
				CredErr:    &rcfile.ReadError{Path: "/etc/.glassfrogrc", Err: errors.New("permission denied")},
			},
			want: false,
		},
		{
			name: "credential error even with a token set is incomplete",
			ctx: ConnectionContext{
				BaseURL: BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceDefault},
				Cred:    auth.Resolution{Token: "gf_x", Source: auth.SourceFile, Path: "/etc/.glassfrogrc"},
				CredErr: &rcfile.FormatError{Path: "/etc/.glassfrogrc"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ctx.Complete(); got != tt.want {
				t.Errorf("Complete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnectionContextProblems(t *testing.T) {
	t.Run("empty when complete", func(t *testing.T) {
		if probs := completeContext("gf_live_secret123").Problems(); len(probs) != 0 {
			t.Errorf("Problems() = %v, want empty", probs)
		}
	})

	t.Run("credential absent names the missing credential", func(t *testing.T) {
		ctx := ConnectionContext{
			BaseURL: BaseURL{Value: DefaultBaseURL, Source: SourceDefault},
			Cred:    auth.Resolution{Source: auth.SourceNone},
		}
		probs := ctx.Problems()
		if len(probs) != 1 {
			t.Fatalf("Problems() = %v, want exactly one entry", probs)
		}
		if !strings.Contains(strings.ToLower(probs[0]), "credential") {
			t.Errorf("Problems()[0] = %q, want it to name the credential", probs[0])
		}
	})

	t.Run("base-URL error names the base-URL part", func(t *testing.T) {
		ctx := ConnectionContext{
			BaseURLErr: &BaseURLError{Source: "--" + FlagBaseURL},
			Cred:       auth.Resolution{Token: "gf_x", Source: auth.SourceEnvironment},
		}
		probs := ctx.Problems()
		if len(probs) != 1 {
			t.Fatalf("Problems() = %v, want exactly one entry", probs)
		}
		if !strings.Contains(strings.ToLower(probs[0]), "base url") {
			t.Errorf("Problems()[0] = %q, want it to name the base URL", probs[0])
		}
	})

	t.Run("credential error names the file path", func(t *testing.T) {
		ctx := ConnectionContext{
			BaseURL: BaseURL{Value: "https://glassfrog.com/api/v5", Source: SourceDefault},
			CredErr: &rcfile.ReadError{Path: "/etc/.glassfrogrc", Err: errors.New("permission denied")},
		}
		probs := ctx.Problems()
		if len(probs) != 1 {
			t.Fatalf("Problems() = %v, want exactly one entry", probs)
		}
		if !strings.Contains(probs[0], "/etc/.glassfrogrc") {
			t.Errorf("Problems()[0] = %q, want it to name the file path", probs[0])
		}
	})

	t.Run("both problems in stable order: base URL first, then credential", func(t *testing.T) {
		ctx := ConnectionContext{
			BaseURLErr: &BaseURLError{Source: "--" + FlagBaseURL},
			Cred:       auth.Resolution{Source: auth.SourceNone},
		}
		probs := ctx.Problems()
		if len(probs) != 2 {
			t.Fatalf("Problems() = %v, want two entries", probs)
		}
		if !strings.Contains(strings.ToLower(probs[0]), "base url") {
			t.Errorf("Problems()[0] = %q, want the base-URL part first", probs[0])
		}
		if !strings.Contains(strings.ToLower(probs[1]), "credential") {
			t.Errorf("Problems()[1] = %q, want the credential part second", probs[1])
		}
	})
}

// TestConnectionContextRedaction is the load-bearing secret-hygiene test: a real
// token value must never appear in %+v, %v, %s, String(), or any Problems() entry
// (mirrors auth.Resolution.String()).
func TestConnectionContextRedaction(t *testing.T) {
	const token = "gf_live_secret123"
	ctx := completeContext(token)

	renderings := map[string]string{
		"String()": ctx.String(),
		"%v":       fmt.Sprintf("%v", ctx),
		"%+v":      fmt.Sprintf("%+v", ctx),
		"%s":       fmt.Sprintf("%s", ctx),
	}
	for verb, out := range renderings {
		if strings.Contains(out, token) {
			t.Errorf("%s leaked the token: %q", verb, out)
		}
	}

	// String() shows the safe parts and reports the token as present.
	if s := ctx.String(); !strings.Contains(s, "file") {
		t.Errorf("String() = %q, want it to show the credential/base-URL source", s)
	}

	// An absent token is reported distinctly, still never verbatim.
	absent := ConnectionContext{
		BaseURL: BaseURL{Value: DefaultBaseURL, Source: SourceDefault},
		Cred:    auth.Resolution{Source: auth.SourceNone},
	}
	if strings.Contains(absent.String(), token) {
		t.Errorf("absent-context String() leaked the token: %q", absent.String())
	}

	// No Problems() entry contains the token even when a token is present but the
	// context is otherwise incomplete.
	withToken := ConnectionContext{
		BaseURLErr: &BaseURLError{Source: "--" + FlagBaseURL},
		Cred:       auth.Resolution{Token: token, Source: auth.SourceEnvironment},
	}
	for _, p := range withToken.Problems() {
		if strings.Contains(p, token) {
			t.Errorf("Problems() leaked the token: %q", p)
		}
	}
}
