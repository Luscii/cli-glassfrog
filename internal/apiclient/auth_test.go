package apiclient

import (
	"errors"
	"strings"
	"testing"

	"github.com/Luscii/cli-glassfrog/internal/auth"
)

// secretToken is the credential used across the mapping tests. Every assertion
// that inspects an AuthError checks this value never surfaces.
const secretToken = "gf_super_secret_token"

func TestAuthorize_EnvironmentSourceReturnsToken(t *testing.T) {
	res := auth.Resolution{Token: secretToken, Source: auth.SourceEnvironment}

	token, authErr := authorize(res, nil)

	if authErr != nil {
		t.Fatalf("authorize returned an error for an Environment source: %v", authErr)
	}
	if token != secretToken {
		t.Fatalf("token = %q, want the resolved token", token)
	}
}

func TestAuthorize_FileSourceReturnsToken(t *testing.T) {
	res := auth.Resolution{Token: secretToken, Source: auth.SourceFile, Path: "/home/dev/.glassfrogrc"}

	token, authErr := authorize(res, nil)

	if authErr != nil {
		t.Fatalf("authorize returned an error for a File source: %v", authErr)
	}
	if token != secretToken {
		t.Fatalf("token = %q, want the resolved token", token)
	}
}

func TestAuthorize_NoneSourceIsNoCredentials(t *testing.T) {
	res := auth.Resolution{Source: auth.SourceNone}

	token, authErr := authorize(res, nil)

	if token != "" {
		t.Fatalf("token = %q, want empty for a None source", token)
	}
	if authErr == nil {
		t.Fatal("authorize returned no error for a None source, want NoCredentials")
	}
	if authErr.Kind != NoCredentials {
		t.Fatalf("Kind = %v, want NoCredentials", authErr.Kind)
	}
}

func TestAuthorize_ResolverErrorIsCredentialError(t *testing.T) {
	cause := &auth.FormatError{Path: "/home/dev/.glassfrogrc"}

	// Even if a token is somehow present on the resolution, an error wins and no
	// token is handed back.
	res := auth.Resolution{Token: secretToken}
	token, authErr := authorize(res, cause)

	if token != "" {
		t.Fatalf("token = %q, want empty when the resolver errored", token)
	}
	if authErr == nil {
		t.Fatal("authorize returned no error despite a resolver error, want CredentialError")
	}
	if authErr.Kind != CredentialError {
		t.Fatalf("Kind = %v, want CredentialError", authErr.Kind)
	}

	var fe *auth.FormatError
	if !errors.As(authErr, &fe) {
		t.Fatalf("AuthError does not unwrap to the original *auth.FormatError; got %v", authErr)
	}
	if fe.Path != cause.Path {
		t.Fatalf("unwrapped path = %q, want %q", fe.Path, cause.Path)
	}
}

func TestAuthError_KindsAreDistinguishable(t *testing.T) {
	none := &AuthError{Kind: NoCredentials}
	broken := &AuthError{Kind: CredentialError, cause: &auth.FormatError{Path: "/x/.glassfrogrc"}}

	if none.Kind == broken.Kind {
		t.Fatal("NoCredentials and CredentialError share a Kind; they must be distinct")
	}
}

func TestAuthError_NeverContainsTheToken(t *testing.T) {
	// A CredentialError wrapping a path-only cause, plus a NoCredentials error:
	// neither Error() string may contain the secret. The cause names only the
	// path (the auth package guarantees this), and authorize never copies the
	// token into the AuthError.
	cause := &auth.ReadError{Path: "/home/dev/.glassfrogrc", Err: errors.New("permission denied")}
	res := auth.Resolution{Token: secretToken}

	_, credErr := authorize(res, cause)
	_, noneErr := authorize(auth.Resolution{Source: auth.SourceNone}, nil)

	for _, e := range []*AuthError{credErr, noneErr} {
		if strings.Contains(e.Error(), secretToken) {
			t.Fatalf("AuthError.Error() leaked the token: %q", e.Error())
		}
	}
	// The credential error must still name the offending path so the operator
	// can find the broken file.
	if !strings.Contains(credErr.Error(), "/home/dev/.glassfrogrc") {
		t.Fatalf("CredentialError.Error() = %q, want it to name the path", credErr.Error())
	}
}

func TestAuthHeaderNameIsXAuthToken(t *testing.T) {
	if AuthHeaderName != "X-Auth-Token" {
		t.Fatalf("AuthHeaderName = %q, want X-Auth-Token", AuthHeaderName)
	}
}
