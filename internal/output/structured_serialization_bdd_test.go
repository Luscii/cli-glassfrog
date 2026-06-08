package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"sigs.k8s.io/yaml"
)

// TestStructuredSerializationFeatures runs the 018 driving scenarios as
// component-level acceptance, driving RenderSuccess / RenderError directly with
// raw-byte and hand-built-envelope fixtures. There is no CLI surface until 020,
// so the steps exercise the package functions — there is no command to invoke.
//
// The suite is scoped to *only* structured-serialization.feature: godog binds
// steps per-suite, and the sibling templated-human-rendering.feature (019) lives
// in the same directory, so a directory-globbing Paths would pull in 019's
// scenarios and fail with undefined steps (LEARNINGS 2026-06-04). The five
// @validation scenarios stay @wip — held for the validate skill — so the
// "~@wip" filter excludes them here.
func TestStructuredSerializationFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: initializeStructuredSerializationScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/unconsumable-output/structured-serialization.feature"},
			Tags:     "~@wip",
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status: feature scenarios failed")
	}
}

// tokenFixture is a stand-in secret. The token is a request header, never a
// response field (011 ADR-1), so no fixture below carries it — the token-absence
// scenario asserts the renderer introduces none.
const tokenFixture = "tok_supersecret_value"

// serWorld is the per-scenario state. A Given scripts the format and the
// success payload or hand-built envelope; a When drives RenderSuccess/RenderError;
// the Thens assert on the captured document and error. Step helpers return
// errors, never panic (LEARNINGS) — and no os.Stdout/Stderr capture is needed,
// since the renderers return their bytes directly.
type serWorld struct {
	format Format

	payload  json.RawMessage // success input
	envelope ErrorEnvelope   // failure input

	wantFields []string // fields that must survive verbatim
	bigInt     string   // the large integer that must keep its exact form

	doc []byte
	err error
}

// parseDoc decodes a rendered document with the YAML decoder (JSON is valid
// YAML), so JSON and YAML forms decode through one number-aware path.
func parseDoc(doc []byte) (any, error) {
	var v any
	if err := yaml.Unmarshal(doc, &v); err != nil {
		return nil, fmt.Errorf("decoding rendered document failed: %w\n%s", err, doc)
	}
	return v, nil
}

// errorDetail returns the single nested error object of a rendered envelope.
func errorDetail(doc []byte) (map[string]any, error) {
	v, err := parseDoc(doc)
	if err != nil {
		return nil, err
	}
	root, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("rendered error document is not an object:\n%s", doc)
	}
	if len(root) != 1 {
		return nil, fmt.Errorf("error document has %d top-level keys, want exactly 1 (error):\n%s", len(root), doc)
	}
	d, ok := root["error"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("error document has no nested error object:\n%s", doc)
	}
	return d, nil
}

func initializeStructuredSerializationScenario(sc *godog.ScenarioContext) {
	w := &serWorld{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		*w = serWorld{format: JSON}
		return ctx, nil
	})

	// --- Givens: format ---
	sc.Step(`^the JSON format was active$`, w.givenJSONActive)
	sc.Step(`^the YAML format was active$`, w.givenYAMLActive)
	sc.Step(`^a structured format was active$`, w.givenStructuredActive)

	// --- Givens: success content ---
	sc.Step(`^a command had produced a successful API payload$`, w.givenSuccessPayload)
	sc.Step(`^a successful payload contained fields the human projection omits, such as hypermedia links$`, w.givenPayloadWithDroppedFields)
	sc.Step(`^a successful response carried an empty collection$`, w.givenEmptyCollection)
	sc.Step(`^a successful payload contained a large integer identifier$`, w.givenLargeInteger)
	sc.Step(`^a 2xx body that was not valid JSON$`, w.givenInvalidBody)

	// --- Givens: failure content ---
	sc.Step(`^the API had returned a non-2xx response carrying an error body$`, w.givenNon2xxWithBody)
	sc.Step(`^a transport failure had occurred with no API body$`, w.givenTransportFailureNoBody)

	// --- Whens ---
	sc.Step(`^the result is rendered$`, w.whenResultRendered)
	sc.Step(`^the failure is rendered$`, w.whenFailureRendered)
	sc.Step(`^the renderer produces any document, on success or failure$`, w.whenAnyDocumentProduced)

	// --- Thens: success ---
	sc.Step(`^the renderer will emit a single valid JSON document carrying the raw payload$`, w.thenValidJSONCarryingPayload)
	sc.Step(`^the document will be the only content on the output channel$`, w.thenOnlyContentOnChannel)
	sc.Step(`^the rendered document will contain those fields verbatim$`, w.thenContainsFieldsVerbatim)
	sc.Step(`^the renderer will emit a valid document representing the empty result$`, w.thenValidEmptyResultDocument)
	sc.Step(`^the output channel will not be left empty$`, w.thenChannelNotEmpty)
	sc.Step(`^the token value will never appear in the document$`, w.thenTokenNeverAppears)
	sc.Step(`^the rendered value will keep its exact representation$`, w.thenValueKeepsExactRepresentation)
	sc.Step(`^it will not be rounded to a floating-point approximation$`, w.thenNotRoundedToFloat)
	sc.Step(`^the renderer will emit a single valid YAML document carrying the same data$`, w.thenValidYAMLSameData)
	sc.Step(`^no field will be added or dropped relative to the JSON form$`, w.thenNoFieldDelta)

	// --- Thens: failure ---
	sc.Step(`^the renderer will emit a unified error envelope as valid JSON$`, w.thenUnifiedEnvelopeValidJSON)
	sc.Step(`^the envelope will carry the raw error body verbatim$`, w.thenEnvelopeCarriesBodyVerbatim)
	sc.Step(`^it will not classify or interpret the error$`, w.thenNotClassifyOrInterpret)
	sc.Step(`^the renderer will emit the same unified error envelope as valid JSON$`, w.thenUnifiedEnvelopeValidJSON)
	sc.Step(`^the envelope will carry the available failure facts without a raw body$`, w.thenFailureFactsWithoutBody)
	sc.Step(`^the renderer will surface a render error$`, w.thenSurfaceRenderError)
	sc.Step(`^it will emit no partial or invalid document$`, w.thenNoPartialDocument)
}

// --- Given implementations ---

func (w *serWorld) givenJSONActive() error       { w.format = JSON; return nil }
func (w *serWorld) givenYAMLActive() error       { w.format = YAML; return nil }
func (w *serWorld) givenStructuredActive() error { w.format = JSON; return nil }

func (w *serWorld) givenSuccessPayload() error {
	w.payload = json.RawMessage(`{"id":"role_1","name":"Finance","tags":["a","b"]}`)
	return nil
}

func (w *serWorld) givenPayloadWithDroppedFields() error {
	w.payload = json.RawMessage(`{"id":"role_1","_links":{"self":"https://glassfrog.com/api/v5/roles/role_1"}}`)
	w.wantFields = []string{"_links", "https://glassfrog.com/api/v5/roles/role_1"}
	return nil
}

func (w *serWorld) givenEmptyCollection() error {
	w.payload = json.RawMessage(`{"data":[]}`)
	return nil
}

func (w *serWorld) givenLargeInteger() error {
	w.bigInt = "9007199254740993" // 2^53 + 1 — not representable as a float64
	w.payload = json.RawMessage(`{"id":` + w.bigInt + `}`)
	return nil
}

func (w *serWorld) givenInvalidBody() error {
	w.payload = json.RawMessage(`{"id": `) // truncated, not valid JSON
	return nil
}

func (w *serWorld) givenNon2xxWithBody() error {
	w.envelope = ErrorEnvelope{Error: ErrorDetail{
		Message: "the API returned a non-2xx response: status 404",
		Kind:    "api",
		Status:  404,
		Body:    json.RawMessage(`{"type":"about:blank","title":"Not Found","status":404,"detail":"No such role"}`),
	}}
	return nil
}

func (w *serWorld) givenTransportFailureNoBody() error {
	w.envelope = ErrorEnvelope{Error: ErrorDetail{
		Message: "request failed: connection refused",
		Kind:    "network",
	}}
	return nil
}

// --- When implementations ---

func (w *serWorld) whenResultRendered() error {
	w.doc, w.err = RenderSuccess(w.format, w.payload)
	return nil
}

func (w *serWorld) whenFailureRendered() error {
	w.doc, w.err = RenderError(w.format, w.envelope)
	return nil
}

func (w *serWorld) whenAnyDocumentProduced() error {
	success, err := RenderSuccess(w.format, json.RawMessage(`{"id":"role_1","name":"Finance"}`))
	if err != nil {
		return fmt.Errorf("rendering the success document failed: %w", err)
	}
	failure, err := RenderError(w.format, ErrorEnvelope{Error: ErrorDetail{
		Message: "the API returned a non-2xx response: status 500",
		Kind:    "api",
		Status:  500,
		Body:    json.RawMessage(`{"detail":"internal error"}`),
	}})
	if err != nil {
		return fmt.Errorf("rendering the failure document failed: %w", err)
	}
	w.doc = append(success, failure...)
	return nil
}

// --- Then implementations ---

func (w *serWorld) thenValidJSONCarryingPayload() error {
	if w.err != nil {
		return fmt.Errorf("render error = %v, want success", w.err)
	}
	if !json.Valid(w.doc) {
		return fmt.Errorf("rendered document is not valid JSON:\n%s", w.doc)
	}
	got, err := parseDoc(w.doc)
	if err != nil {
		return err
	}
	want, err := parseDoc(w.payload)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("document does not carry the raw payload\n got: %#v\nwant: %#v", got, want)
	}
	return nil
}

func (w *serWorld) thenOnlyContentOnChannel() error {
	if w.err != nil {
		return fmt.Errorf("render error = %v", w.err)
	}
	if len(bytes.TrimSpace(w.doc)) == 0 {
		return fmt.Errorf("output channel is empty")
	}
	if !json.Valid(w.doc) {
		return fmt.Errorf("channel carries more than one document or trailing junk:\n%s", w.doc)
	}
	return nil
}

func (w *serWorld) thenContainsFieldsVerbatim() error {
	if w.err != nil {
		return fmt.Errorf("render error = %v", w.err)
	}
	for _, field := range w.wantFields {
		if !strings.Contains(string(w.doc), field) {
			return fmt.Errorf("rendered document dropped %q:\n%s", field, w.doc)
		}
	}
	return nil
}

func (w *serWorld) thenValidEmptyResultDocument() error {
	if w.err != nil {
		return fmt.Errorf("render error = %v", w.err)
	}
	got, err := parseDoc(w.doc)
	if err != nil {
		return err
	}
	want, err := parseDoc(w.payload)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("document does not represent the empty result\n got: %#v\nwant: %#v", got, want)
	}
	return nil
}

func (w *serWorld) thenChannelNotEmpty() error {
	if len(bytes.TrimSpace(w.doc)) == 0 {
		return fmt.Errorf("output channel was left empty")
	}
	return nil
}

func (w *serWorld) thenTokenNeverAppears() error {
	if strings.Contains(string(w.doc), tokenFixture) {
		return fmt.Errorf("token leaked into the rendered document:\n%s", w.doc)
	}
	return nil
}

func (w *serWorld) thenValueKeepsExactRepresentation() error {
	if w.err != nil {
		return fmt.Errorf("render error = %v", w.err)
	}
	if !strings.Contains(string(w.doc), w.bigInt) {
		return fmt.Errorf("large integer %q lost its exact form:\n%s", w.bigInt, w.doc)
	}
	return nil
}

func (w *serWorld) thenNotRoundedToFloat() error {
	doc := string(w.doc)
	if strings.Contains(doc, "9007199254740992") || strings.Contains(doc, "9.007") {
		return fmt.Errorf("large integer was rounded to a float approximation:\n%s", doc)
	}
	return nil
}

func (w *serWorld) compareYAMLToJSON() error {
	if w.err != nil {
		return fmt.Errorf("render error = %v", w.err)
	}
	jsonDoc, err := RenderSuccess(JSON, w.payload)
	if err != nil {
		return fmt.Errorf("rendering the JSON comparison form failed: %w", err)
	}
	yamlData, err := parseDoc(w.doc)
	if err != nil {
		return err
	}
	jsonData, err := parseDoc(jsonDoc)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(yamlData, jsonData) {
		return fmt.Errorf("YAML and JSON forms carry different data\nyaml: %#v\njson: %#v", yamlData, jsonData)
	}
	return nil
}

func (w *serWorld) thenValidYAMLSameData() error { return w.compareYAMLToJSON() }
func (w *serWorld) thenNoFieldDelta() error      { return w.compareYAMLToJSON() }

func (w *serWorld) thenUnifiedEnvelopeValidJSON() error {
	if w.err != nil {
		return fmt.Errorf("render error = %v, want success", w.err)
	}
	if !json.Valid(w.doc) {
		return fmt.Errorf("rendered envelope is not valid JSON:\n%s", w.doc)
	}
	if _, err := errorDetail(w.doc); err != nil {
		return err
	}
	return nil
}

func (w *serWorld) thenEnvelopeCarriesBodyVerbatim() error {
	detail, err := errorDetail(w.doc)
	if err != nil {
		return err
	}
	gotBody, ok := detail["body"]
	if !ok {
		return fmt.Errorf("error.body missing:\n%s", w.doc)
	}
	wantBody, err := parseDoc(w.envelope.Error.Body)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(gotBody, wantBody) {
		return fmt.Errorf("error.body not carried verbatim\n got: %#v\nwant: %#v", gotBody, wantBody)
	}
	return nil
}

func (w *serWorld) thenNotClassifyOrInterpret() error {
	detail, err := errorDetail(w.doc)
	if err != nil {
		return err
	}
	// The renderer reflects the facts it was handed — no reclassification.
	if got := detail["kind"]; got != w.envelope.Error.Kind {
		return fmt.Errorf("error.kind = %v, want %q exactly as supplied (renderer must not classify)", got, w.envelope.Error.Kind)
	}
	if got := fmt.Sprint(detail["status"]); got != fmt.Sprint(w.envelope.Error.Status) {
		return fmt.Errorf("error.status = %v, want %d exactly as supplied", detail["status"], w.envelope.Error.Status)
	}
	return nil
}

func (w *serWorld) thenFailureFactsWithoutBody() error {
	detail, err := errorDetail(w.doc)
	if err != nil {
		return err
	}
	if detail["message"] == nil || detail["kind"] == nil {
		return fmt.Errorf("error.message/kind missing — always-present facts:\n%s", w.doc)
	}
	if _, present := detail["body"]; present {
		return fmt.Errorf("error.body present on a bodiless failure (want absent):\n%s", w.doc)
	}
	if _, present := detail["status"]; present {
		return fmt.Errorf("error.status present on a bodiless failure (want absent):\n%s", w.doc)
	}
	return nil
}

func (w *serWorld) thenSurfaceRenderError() error {
	if w.err == nil {
		return fmt.Errorf("render error = nil, want a render error")
	}
	return nil
}

func (w *serWorld) thenNoPartialDocument() error {
	if w.doc != nil {
		return fmt.Errorf("renderer emitted a partial document on failure: %q", w.doc)
	}
	return nil
}
