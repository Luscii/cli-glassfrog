package output

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// decodeAny parses a rendered document with the YAML decoder (JSON is valid
// YAML), so JSON and YAML forms decode through one number-aware path and compare
// equal when they carry the same data.
func decodeAny(t *testing.T, doc []byte) any {
	t.Helper()
	var v any
	if err := yaml.Unmarshal(doc, &v); err != nil {
		t.Fatalf("decoding rendered document failed: %v\n%s", err, doc)
	}
	return v
}

func TestRenderSuccess_JSONIsValidAndCarriesPayload(t *testing.T) {
	payload := json.RawMessage(`{"id":"role_1","name":"Finance"}`)

	doc, err := RenderSuccess(JSON, payload)
	if err != nil {
		t.Fatalf("RenderSuccess(JSON) error = %v", err)
	}
	if !json.Valid(doc) {
		t.Fatalf("rendered JSON is not valid:\n%s", doc)
	}
	got := decodeAny(t, doc)
	want := decodeAny(t, payload)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rendered JSON carries different data\n got: %#v\nwant: %#v", got, want)
	}
}

func TestRenderSuccess_JSONPreservesFieldsTheProjectionDrops(t *testing.T) {
	// The tolerant typed glassfrog struct drops unmodeled fields such as
	// hypermedia links; the raw-bytes path must keep them verbatim.
	payload := json.RawMessage(`{"id":"role_1","_links":{"self":"https://glassfrog.com/api/v5/roles/role_1"}}`)

	doc, err := RenderSuccess(JSON, payload)
	if err != nil {
		t.Fatalf("RenderSuccess(JSON) error = %v", err)
	}
	if !strings.Contains(string(doc), "_links") {
		t.Errorf("rendered document dropped the _links field:\n%s", doc)
	}
	if !strings.Contains(string(doc), "https://glassfrog.com/api/v5/roles/role_1") {
		t.Errorf("rendered document dropped the hypermedia link value:\n%s", doc)
	}
}

func TestRenderSuccess_YAMLIsValidAndCarriesSameData(t *testing.T) {
	payload := json.RawMessage(`{"id":"role_1","name":"Finance","tags":["a","b"]}`)

	yamlDoc, err := RenderSuccess(YAML, payload)
	if err != nil {
		t.Fatalf("RenderSuccess(YAML) error = %v", err)
	}
	jsonDoc, err := RenderSuccess(JSON, payload)
	if err != nil {
		t.Fatalf("RenderSuccess(JSON) error = %v", err)
	}

	yamlData := decodeAny(t, yamlDoc)
	jsonData := decodeAny(t, jsonDoc)
	if !reflect.DeepEqual(yamlData, jsonData) {
		t.Fatalf("YAML and JSON forms carry different data\nyaml: %#v\njson: %#v", yamlData, jsonData)
	}
}

func TestRenderSuccess_LargeIntegerKeepsExactRepresentation(t *testing.T) {
	// 2^53 + 1 is not representable as a float64 — a generic any decode would
	// round it. The bytes path (json.Indent / JSONToYAML) keeps it exact.
	const bigInt = "9007199254740993"
	payload := json.RawMessage(`{"id":` + bigInt + `}`)

	for _, f := range []Format{JSON, YAML} {
		doc, err := RenderSuccess(f, payload)
		if err != nil {
			t.Fatalf("RenderSuccess(%s) error = %v", f, err)
		}
		if !strings.Contains(string(doc), bigInt) {
			t.Errorf("RenderSuccess(%s) lost integer precision; %q not in:\n%s", f, bigInt, doc)
		}
		if strings.Contains(string(doc), "9007199254740992") || strings.Contains(string(doc), "9.007") {
			t.Errorf("RenderSuccess(%s) rounded the integer to a float:\n%s", f, doc)
		}
	}
}

func TestRenderSuccess_Deterministic(t *testing.T) {
	payload := json.RawMessage(`{"b":2,"a":1,"nested":{"y":true,"x":false}}`)

	for _, f := range []Format{JSON, YAML} {
		first, err := RenderSuccess(f, payload)
		if err != nil {
			t.Fatalf("RenderSuccess(%s) error = %v", f, err)
		}
		second, err := RenderSuccess(f, payload)
		if err != nil {
			t.Fatalf("RenderSuccess(%s) error = %v", f, err)
		}
		if !bytes.Equal(first, second) {
			t.Errorf("RenderSuccess(%s) not deterministic:\n%s\n---\n%s", f, first, second)
		}
	}
}

func TestRenderSuccess_EmptyPayloadRendersValidNonEmptyDocument(t *testing.T) {
	for _, f := range []Format{JSON, YAML} {
		for _, payload := range []json.RawMessage{nil, json.RawMessage(""), json.RawMessage("   \n\t ")} {
			doc, err := RenderSuccess(f, payload)
			if err != nil {
				t.Fatalf("RenderSuccess(%s, %q) error = %v", f, payload, err)
			}
			if len(bytes.TrimSpace(doc)) == 0 {
				t.Errorf("RenderSuccess(%s, %q) left the channel empty", f, payload)
			}
			// The empty document must itself be a valid, parseable document.
			decodeAny(t, doc)
		}
	}
}

func TestRenderSuccess_InvalidJSONReturnsErrorAndNoDocument(t *testing.T) {
	for _, f := range []Format{JSON, YAML} {
		doc, err := RenderSuccess(f, json.RawMessage(`{"id": `))
		if err == nil {
			t.Fatalf("RenderSuccess(%s) on invalid JSON: error = nil, want a render error", f)
		}
		if doc != nil {
			t.Errorf("RenderSuccess(%s) on invalid JSON returned a partial document: %q", f, doc)
		}
	}
}
