// Package output is the machine-rendering branch of the Output Formatting
// solution (018). It turns a command's result into a JSON or YAML document and
// renders failures as one unified error envelope in the same format. It is a
// pure leaf: no cobra, no transport, no domain types — imported one-directionally
// by internal/cli the way internal/glassfrog (schema) and internal/apiclient
// (transport) are, with no import cycle. It owns no flag and no command.
//
// The defining constraint is fidelity. Execute decodes a 2xx body into the
// tolerant typed glassfrog structs (011 ADR-1 — unknown fields ignored) and
// discards the raw bytes, so re-encoding a struct would silently drop fields and
// risk number-precision loss. RenderSuccess therefore serializes the raw response
// bytes (ADR-2): JSON via stdlib normalization, YAML via sigs.k8s.io/yaml's
// JSONToYAML (ADR-3) — both operate on bytes, so no JSON number is coerced to
// float64 and no field is dropped. "JSON ≡ YAML" holds by construction because
// the YAML is a transform of the same JSON bytes.
//
// Selection of the format, the decode-target choice, and the typed-error→envelope
// mapping happen at the internal/cli / 020 boundary; this package's renderers are
// pure encoders of the values they are handed. Templated Human Rendering (019)
// extends this package with template rendering; Output Format Selection (020)
// adds the --output flag and its router.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

// Format is the set of machine formats 018 supports. 020 extends the *selection*
// vocabulary with the human full/compact formats (019); these two identifiers are
// 018's.
type Format int

const (
	// JSON renders the raw bytes normalized into a single valid, indented JSON
	// document.
	JSON Format = iota
	// YAML renders the raw JSON bytes transformed into YAML via JSONToYAML.
	YAML
)

// String returns the lowercase format identifier, used in render-error messages.
func (f Format) String() string {
	switch f {
	case JSON:
		return "json"
	case YAML:
		return "yaml"
	default:
		return "unknown"
	}
}

// jsonIndent is the per-level indentation for normalized JSON output — a single
// fixed value so the same input renders an identical document each time.
const jsonIndent = "  "

// emptyDocument is the valid empty document rendered for an empty/whitespace-only
// payload (e.g. a 204 No Content). null is valid in both JSON and YAML, so the
// output channel is never left empty regardless of the active format.
var emptyDocument = []byte("null\n")

// RenderSuccess renders the raw 2xx body verbatim in f.
//
//   - JSON: the bytes validated and normalized into a single valid, consistently
//     indented document — numbers kept as their literal token (no float64
//     coercion) because json.Indent operates on bytes, not a decoded value.
//   - YAML: yaml.JSONToYAML(payload), which threads the JSON through go-yaml's
//     number-aware decode so integers keep their exact representation.
//
// An empty/whitespace-only payload renders as a valid empty document (never an
// empty channel). Bytes that are not valid JSON (a 2xx contract violation) return
// a render error and no document — the whole document is built in memory before
// it is returned, so a render failure never yields a partial or invalid fragment.
func RenderSuccess(f Format, payload json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return append([]byte(nil), emptyDocument...), nil
	}
	if !json.Valid(payload) {
		return nil, fmt.Errorf("rendering %s output failed: payload is not valid JSON", f)
	}

	switch f {
	case JSON:
		var buf bytes.Buffer
		if err := json.Indent(&buf, payload, "", jsonIndent); err != nil {
			return nil, fmt.Errorf("rendering json output failed: %w", err)
		}
		buf.WriteByte('\n')
		return buf.Bytes(), nil
	case YAML:
		doc, err := yaml.JSONToYAML(payload)
		if err != nil {
			return nil, fmt.Errorf("rendering yaml output failed: %w", err)
		}
		return doc, nil
	default:
		return nil, fmt.Errorf("rendering output failed: unsupported format %q", f)
	}
}
