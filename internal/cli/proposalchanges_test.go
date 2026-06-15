package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeFileInfo is a minimal os.FileInfo for driving resolveChangesSource's
// existing-regular-file check without touching the real filesystem. mode carries the
// only bit the resolver reads (IsRegular / IsDir).
type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string       { return "fake" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any           { return nil }

// statNotFound stands in for a path that does not exist (the inline-JSON fall-through).
func statNotFound(string) (os.FileInfo, error) { return nil, fs.ErrNotExist }

// statRegular reports an existing regular file; statDir reports a directory.
func statRegular(string) (os.FileInfo, error) { return fakeFileInfo{mode: 0}, nil }
func statDir(string) (os.FileInfo, error)     { return fakeFileInfo{mode: os.ModeDir}, nil }

// statPermission stands in for a value that names a path the process cannot reach
// (e.g. a parent directory lacks traversal permission) — a non-not-exist stat error.
func statPermission(string) (os.FileInfo, error) { return nil, fs.ErrPermission }

// statNameTooLong stands in for a value too long to be a path name — a large inline
// JSON array overflows NAME_MAX and stats as ENAMETOOLONG, which is NOT a permission
// error and must fall through to the inline branch.
func statNameTooLong(string) (os.FileInfo, error) {
	return nil, &fs.PathError{Op: "stat", Err: fmt.Errorf("file name too long")}
}

// readFails always errors, standing in for an unreadable file.
func readFails(string) ([]byte, error) { return nil, errors.New("permission denied") }

// --- resolveChangesSource: stdin arm ---------------------------------------

func TestResolveChangesSource_StdinHappy(t *testing.T) {
	body := `[{"type":"CreateRole"}]`
	got, err := resolveChangesSource("stdin", statNotFound, readFails, false, strings.NewReader(body))
	if err != nil {
		t.Fatalf("piped stdin should read, got %v", err)
	}
	if string(got) != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestResolveChangesSource_StdinCaseInsensitive(t *testing.T) {
	got, err := resolveChangesSource("  STDIN  ", statRegular, readFails, false, strings.NewReader(`[{"type":"X"}]`))
	if err != nil {
		t.Fatalf("the reserved keyword is case-insensitive and beats a file, got %v", err)
	}
	if string(got) != `[{"type":"X"}]` {
		t.Errorf("reserved stdin should win over a file literally named stdin, got %q", got)
	}
}

func TestResolveChangesSource_StdinTTY(t *testing.T) {
	_, err := resolveChangesSource("stdin", statNotFound, readFails, true, strings.NewReader("ignored"))
	if err == nil {
		t.Fatal("a terminal stdin should be a usage error")
	}
	if !strings.Contains(err.Error(), "terminal") {
		t.Errorf("the error should name the terminal source: %v", err)
	}
}

func TestResolveChangesSource_StdinEmptyPipe(t *testing.T) {
	_, err := resolveChangesSource("stdin", statNotFound, readFails, false, strings.NewReader("   \n"))
	if err == nil {
		t.Fatal("an empty pipe should be a usage error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("the error should name the empty pipe: %v", err)
	}
}

// --- resolveChangesSource: file arm ----------------------------------------

func TestResolveChangesSource_ExistingFileRead(t *testing.T) {
	want := `[{"type":"CreateRole","name":"Scribe"}]`
	readFile := func(p string) ([]byte, error) {
		if p != "changes.json" {
			t.Errorf("readFile got path %q, want changes.json", p)
		}
		return []byte(want), nil
	}
	got, err := resolveChangesSource("changes.json", statRegular, readFile, false, nil)
	if err != nil {
		t.Fatalf("an existing regular file should be read, got %v", err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveChangesSource_DirectoryRejected(t *testing.T) {
	_, err := resolveChangesSource("somedir", statDir, readFails, false, nil)
	if err == nil {
		t.Fatal("a directory should be rejected, not read as a change set")
	}
	if !strings.Contains(err.Error(), "regular file") || !strings.Contains(err.Error(), "somedir") {
		t.Errorf("the error should name the source and the regular-file requirement: %v", err)
	}
}

func TestResolveChangesSource_UnreadableFile(t *testing.T) {
	_, err := resolveChangesSource("changes.json", statRegular, readFails, false, nil)
	if err == nil {
		t.Fatal("an unreadable file should be a usage error")
	}
	if !strings.Contains(err.Error(), "changes.json") {
		t.Errorf("the error should name the file: %v", err)
	}
}

// TestResolveChangesSource_PermissionErrorSurfaced pins that a permission error on
// stat (a path the operator likely meant but the process cannot reach) is surfaced as
// a named source error, NOT silently treated as inline JSON (which would mislead with
// a "must be a JSON array" parse error downstream).
func TestResolveChangesSource_PermissionErrorSurfaced(t *testing.T) {
	_, err := resolveChangesSource("/root/secret.json", statPermission, readFails, false, nil)
	if err == nil {
		t.Fatal("a permission error on stat should be surfaced, not treated as inline")
	}
	if !strings.Contains(err.Error(), "/root/secret.json") {
		t.Errorf("the error should name the source: %v", err)
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Errorf("the error should wrap the underlying permission error: %v", err)
	}
}

// --- resolveChangesSource: inline arm --------------------------------------

func TestResolveChangesSource_InlineWhenNotAFile(t *testing.T) {
	inline := `[{"type":"CreateRole"}]`
	got, err := resolveChangesSource(inline, statNotFound, readFails, false, nil)
	if err != nil {
		t.Fatalf("a value that is not an existing file is inline JSON, got %v", err)
	}
	if string(got) != inline {
		t.Errorf("inline bytes should pass through, got %q", got)
	}
}

// TestResolveChangesSource_LongInlineNotMisclassified pins that a non-permission stat
// error (here ENAMETOOLONG — a long inline JSON array overflows NAME_MAX) falls through
// to the inline branch rather than erroring. Guards against a regression where surfacing
// "any non-not-exist stat error" would break the primary inline case for large arrays.
func TestResolveChangesSource_LongInlineNotMisclassified(t *testing.T) {
	// A JSON array well over NAME_MAX (255) bytes — the realistic inline case that
	// stats as ENAMETOOLONG rather than not-exist.
	inline := "[" + strings.Repeat(`{"type":"CreateRole","name":"a very long role name to pad the bytes"},`, 10)
	inline = strings.TrimSuffix(inline, ",") + "]"
	if len(inline) <= 255 {
		t.Fatalf("test fixture should exceed NAME_MAX, got %d bytes", len(inline))
	}
	got, err := resolveChangesSource(inline, statNameTooLong, readFails, false, nil)
	if err != nil {
		t.Fatalf("a long inline value (ENAMETOOLONG on stat) should be treated as inline, got %v", err)
	}
	if string(got) != inline {
		t.Errorf("long inline bytes should pass through verbatim, got %q", got)
	}
}

// --- validateChanges -------------------------------------------------------

func TestValidateChanges_AcceptsTypedArrayVerbatim(t *testing.T) {
	raw := []byte(`[{"type":"CreateRole","name":"Scribe","keep":{"a":1}},{"type":"UpdateRole"}]`)
	changes, err := validateChanges(raw)
	if err != nil {
		t.Fatalf("a non-empty array of typed objects should pass, got %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("want 2 changes, got %d", len(changes))
	}
	// Re-marshalling the returned slice reproduces the input array exactly: every
	// command-specific key beyond `type` is preserved untouched (verbatim pass-through).
	out, err := json.Marshal(changes)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(out) != `[{"type":"CreateRole","name":"Scribe","keep":{"a":1}},{"type":"UpdateRole"}]` {
		t.Errorf("changes were reshaped, not preserved verbatim: %s", out)
	}
}

func TestValidateChanges_RejectsNonJSON(t *testing.T) {
	if _, err := validateChanges([]byte("not json")); err == nil {
		t.Fatal("non-JSON should be rejected")
	}
}

func TestValidateChanges_RejectsNonArray(t *testing.T) {
	if _, err := validateChanges([]byte(`{"type":"CreateRole"}`)); err == nil {
		t.Fatal("a JSON object (not an array) should be rejected")
	}
}

// TestValidateChanges_RejectsNull pins that JSON null — which unmarshals into a nil
// slice WITHOUT error — is rejected as "must be a JSON array", not the misleading "at
// least one change is required" (null is not an empty array).
func TestValidateChanges_RejectsNull(t *testing.T) {
	_, err := validateChanges([]byte(`null`))
	if err == nil {
		t.Fatal("JSON null should be rejected")
	}
	if !strings.Contains(err.Error(), "JSON array") {
		t.Errorf("null should be rejected as a non-array, got: %v", err)
	}
	if strings.Contains(err.Error(), "at least one change") {
		t.Errorf("null should NOT report the empty-array message, got: %v", err)
	}
}

func TestValidateChanges_RejectsEmptyArray(t *testing.T) {
	_, err := validateChanges([]byte(`[]`))
	if err == nil {
		t.Fatal("an empty array should be rejected")
	}
	if !strings.Contains(err.Error(), "at least one change") {
		t.Errorf("the error should say at least one change is required: %v", err)
	}
}

func TestValidateChanges_RejectsNonObjectElement(t *testing.T) {
	if _, err := validateChanges([]byte(`["CreateRole", 42]`)); err == nil {
		t.Fatal("a non-object element should be rejected")
	}
}

func TestValidateChanges_RejectsTypelessElement(t *testing.T) {
	_, err := validateChanges([]byte(`[{"name":"Scribe"}]`))
	if err == nil {
		t.Fatal("an element lacking a type should be rejected")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Errorf(`the error should mention "type": %v`, err)
	}
}

func TestValidateChanges_RejectsBlankType(t *testing.T) {
	if _, err := validateChanges([]byte(`[{"type":"   "}]`)); err == nil {
		t.Fatal("a blank type should be rejected")
	}
}
