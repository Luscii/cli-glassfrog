package build

import "testing"

// TestParseOrientationManifest_RequiredFields pins the manifest-validity contract
// (PR #148 review): name, version, and description are required, so a
// syntactically-valid manifest that omits any of them is rejected rather than
// silently passing the build-side checks.
func TestParseOrientationManifest_RequiredFields(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"complete", `{"name":"glassfrog","version":"0.1.0","description":"d"}`, false},
		{"missing name", `{"version":"0.1.0","description":"d"}`, true},
		{"empty name", `{"name":"","version":"0.1.0","description":"d"}`, true},
		{"missing version", `{"name":"glassfrog","description":"d"}`, true},
		{"missing description", `{"name":"glassfrog","version":"0.1.0"}`, true},
		{"malformed json", `{"name":"glassfrog",`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseOrientationManifest([]byte(c.raw))
			if (err != nil) != c.wantErr {
				t.Fatalf("ParseOrientationManifest(%s) err=%v, wantErr=%v", c.raw, err, c.wantErr)
			}
		})
	}
}

// TestManifestDemandsNoSetup_ForbiddenKeys pins the directory-discovery contract
// (PR #148 review): a manifest carrying a `skills` array (or any other forbidden
// capability/setup key) is rejected, since skills are discovered from skills/.
func TestManifestDemandsNoSetup_ForbiddenKeys(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"clean", `{"name":"glassfrog","version":"0.1.0","description":"d"}`, true},
		{"declares skills", `{"name":"glassfrog","skills":["x"]}`, false},
		{"declares commands", `{"name":"glassfrog","commands":["x"]}`, false},
		{"declares hooks", `{"name":"glassfrog","hooks":{}}`, false},
		{"declares mcpServers", `{"name":"glassfrog","mcpServers":{}}`, false},
		{"declares agents", `{"name":"glassfrog","agents":[]}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ManifestDemandsNoSetup([]byte(c.raw)); got != c.want {
				t.Fatalf("ManifestDemandsNoSetup(%s) = %v, want %v", c.raw, got, c.want)
			}
		})
	}
}
