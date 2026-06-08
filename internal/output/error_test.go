package output

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// topLevelKeys decodes a rendered error document and returns the keys of the
// single nested error object, so a test can assert which fields are present or
// absent (not null-keyed).
func topLevelKeys(t *testing.T, doc []byte) (errKeys map[string]any) {
	t.Helper()
	root, ok := decodeAny(t, doc).(map[string]any)
	if !ok {
		t.Fatalf("rendered error document is not an object:\n%s", doc)
	}
	if len(root) != 1 {
		t.Fatalf("error document has %d top-level keys, want exactly 1 (error):\n%s", len(root), doc)
	}
	detail, ok := root["error"].(map[string]any)
	if !ok {
		t.Fatalf("error document has no nested error object:\n%s", doc)
	}
	return detail
}

func TestRenderError_APIErrorCarriesRawBodyVerbatim(t *testing.T) {
	body := json.RawMessage(`{"type":"about:blank","title":"Not Found","status":404,"detail":"No such role"}`)
	env := ErrorEnvelope{Error: ErrorDetail{
		Message: "the API returned a non-2xx response: status 404",
		Kind:    "api",
		Status:  404,
		Body:    body,
	}}

	doc, err := RenderError(JSON, env)
	if err != nil {
		t.Fatalf("RenderError(JSON) error = %v", err)
	}
	if !json.Valid(doc) {
		t.Fatalf("rendered error is not valid JSON:\n%s", doc)
	}

	detail := topLevelKeys(t, doc)
	gotBody, ok := detail["body"].(map[string]any)
	if !ok {
		t.Fatalf("error.body is not a nested object (must nest verbatim, not a quoted string):\n%s", doc)
	}
	wantBody := decodeAny(t, body)
	if !reflect.DeepEqual(gotBody, wantBody) {
		t.Errorf("error.body not carried verbatim\n got: %#v\nwant: %#v", gotBody, wantBody)
	}
	if detail["status"] == nil {
		t.Errorf("error.status missing for an API error:\n%s", doc)
	}
}

func TestRenderError_BodilessFailureOmitsStatusAndBody(t *testing.T) {
	env := ErrorEnvelope{Error: ErrorDetail{
		Message: "request failed: connection refused",
		Kind:    "network",
	}}

	doc, err := RenderError(JSON, env)
	if err != nil {
		t.Fatalf("RenderError(JSON) error = %v", err)
	}

	detail := topLevelKeys(t, doc)
	if _, present := detail["status"]; present {
		t.Errorf("error.status present on a bodiless failure (want absent, not null-keyed):\n%s", doc)
	}
	if _, present := detail["body"]; present {
		t.Errorf("error.body present on a bodiless failure (want absent, not null-keyed):\n%s", doc)
	}
	if detail["message"] == nil || detail["kind"] == nil {
		t.Errorf("error.message/kind missing — always-present fields:\n%s", doc)
	}
}

func TestRenderError_AllFailuresShareTopLevelShape(t *testing.T) {
	apiDoc, err := RenderError(JSON, ErrorEnvelope{Error: ErrorDetail{
		Message: "status 404", Kind: "api", Status: 404, Body: json.RawMessage(`{"detail":"x"}`),
	}})
	if err != nil {
		t.Fatalf("RenderError(api) error = %v", err)
	}
	netDoc, err := RenderError(JSON, ErrorEnvelope{Error: ErrorDetail{
		Message: "connection refused", Kind: "network",
	}})
	if err != nil {
		t.Fatalf("RenderError(network) error = %v", err)
	}
	failsafeDoc, err := RenderError(JSON, ErrorEnvelope{Error: ErrorDetail{
		Message: "refusing to send a request without credentials", Kind: "runtime",
	}})
	if err != nil {
		t.Fatalf("RenderError(runtime) error = %v", err)
	}

	for _, doc := range [][]byte{apiDoc, netDoc, failsafeDoc} {
		root, ok := decodeAny(t, doc).(map[string]any)
		if !ok || len(root) != 1 {
			t.Fatalf("failure document does not share the single-key error shape:\n%s", doc)
		}
		if _, ok := root["error"]; !ok {
			t.Fatalf("failure document is not wrapped under `error`:\n%s", doc)
		}
	}
}

func TestRenderError_JSONAndYAMLEncodeIdenticalData(t *testing.T) {
	env := ErrorEnvelope{Error: ErrorDetail{
		Message: "status 422",
		Kind:    "api",
		Status:  422,
		Body:    json.RawMessage(`{"errors":[{"field":"name"}],"id":9007199254740993}`),
	}}

	jsonDoc, err := RenderError(JSON, env)
	if err != nil {
		t.Fatalf("RenderError(JSON) error = %v", err)
	}
	yamlDoc, err := RenderError(YAML, env)
	if err != nil {
		t.Fatalf("RenderError(YAML) error = %v", err)
	}
	if !reflect.DeepEqual(decodeAny(t, jsonDoc), decodeAny(t, yamlDoc)) {
		t.Fatalf("JSON and YAML envelopes differ\njson:\n%s\nyaml:\n%s", jsonDoc, yamlDoc)
	}
	// Number fidelity survives the YAML path too.
	if !strings.Contains(string(yamlDoc), "9007199254740993") {
		t.Errorf("YAML envelope lost integer precision:\n%s", yamlDoc)
	}
}

func TestRenderError_EachKindRendersWithoutError(t *testing.T) {
	// The renderer is taxonomy-agnostic — every kind term renders. Covers the
	// four 018 categories plus the 015-widened permission / rate-limit.
	for _, kind := range []string{"api", "network", "usage", "runtime", "permission", "rate-limit"} {
		for _, f := range []Format{JSON, YAML} {
			doc, err := RenderError(f, ErrorEnvelope{Error: ErrorDetail{Message: "x", Kind: kind}})
			if err != nil {
				t.Errorf("RenderError(%s, kind=%s) error = %v", f, kind, err)
			}
			if len(doc) == 0 {
				t.Errorf("RenderError(%s, kind=%s) produced no document", f, kind)
			}
		}
	}
}

func TestRenderError_RendererIntroducesNoToken(t *testing.T) {
	const token = "secret-token-value"
	// A secret-free envelope must stay secret-free — the renderer adds nothing.
	env := ErrorEnvelope{Error: ErrorDetail{
		Message: "the API returned a non-2xx response: status 500",
		Kind:    "api",
		Status:  500,
		Body:    json.RawMessage(`{"detail":"internal error"}`),
	}}
	for _, f := range []Format{JSON, YAML} {
		doc, err := RenderError(f, env)
		if err != nil {
			t.Fatalf("RenderError(%s) error = %v", f, err)
		}
		if strings.Contains(string(doc), token) {
			t.Errorf("RenderError(%s) leaked a token into the document:\n%s", f, doc)
		}
	}
}

func TestRenderError_InvalidBodyReturnsErrorAndNoDocument(t *testing.T) {
	env := ErrorEnvelope{Error: ErrorDetail{
		Message: "x", Kind: "api", Body: json.RawMessage(`{"unterminated":`),
	}}
	for _, f := range []Format{JSON, YAML} {
		doc, err := RenderError(f, env)
		if err == nil {
			t.Fatalf("RenderError(%s) with invalid body: error = nil, want a render error", f)
		}
		if doc != nil {
			t.Errorf("RenderError(%s) returned a partial document on failure: %q", f, doc)
		}
	}
}
