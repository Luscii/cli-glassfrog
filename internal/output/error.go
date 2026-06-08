package output

import (
	"bytes"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

// ErrorEnvelope is the unified failure document (ADR-4) — one shape for every
// failure origin (API error, transport failure, fail-safe refusal), wrapping a
// single ErrorDetail under the `error` key. 018 owns the envelope *shape* and its
// encoder only; the typed-error→envelope mapping (kind from classifyClientError,
// status/body from *ResponseError) lives at the cli/020 boundary, so this package
// performs no classification and imports no transport.
type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail carries the failure facts. Status and Body are omitempty so a
// bodiless failure shares the exact top-level shape of an API error — the fields
// that do not apply are absent, never null-keyed or renamed.
type ErrorDetail struct {
	// Message is a human-readable, token-free description (always present).
	Message string `json:"message"`
	// Kind is the lowercased taxonomy term (always present): usage / runtime /
	// network / api (plus the 015-widened permission / rate-limit).
	Kind string `json:"kind"`
	// Status is the HTTP status, present only for a non-2xx response.
	Status int `json:"status,omitempty"`
	// Body is the raw API error body verbatim, present only when the API returned
	// one. It nests as structured data (object/array), not a quoted JSON string.
	Body json.RawMessage `json:"body,omitempty"`
}

// RenderError renders env in f with a complete document or a render error, never
// a fragment.
//
//   - JSON: env marshaled with stable, declaration-ordered, indented fields.
//   - YAML: the same envelope marshaled to JSON, then transformed via JSONToYAML,
//     so Body nests as structured YAML and integers keep their representation.
//
// The whole document is built in memory before it is returned; if marshaling
// fails (e.g. Body is not valid JSON) it returns an error and no document. The
// renderer adds nothing to the values it is handed — a secret-free envelope stays
// secret-free.
func RenderError(f Format, env ErrorEnvelope) ([]byte, error) {
	switch f {
	case JSON:
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", jsonIndent)
		// Keep the raw body verbatim and consistent with RenderSuccess's
		// json.Indent, which does not HTML-escape <, >, or &.
		enc.SetEscapeHTML(false)
		if err := enc.Encode(env); err != nil {
			return nil, fmt.Errorf("rendering json error envelope failed: %w", err)
		}
		// json.Encoder.Encode already appends a trailing newline.
		return buf.Bytes(), nil
	case YAML:
		raw, err := json.Marshal(env)
		if err != nil {
			return nil, fmt.Errorf("rendering yaml error envelope failed: %w", err)
		}
		doc, err := yaml.JSONToYAML(raw)
		if err != nil {
			return nil, fmt.Errorf("rendering yaml error envelope failed: %w", err)
		}
		return doc, nil
	default:
		return nil, fmt.Errorf("rendering error envelope failed: unsupported format %q", f)
	}
}
