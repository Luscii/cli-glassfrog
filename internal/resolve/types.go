// Package resolve is the one composable precedence walk shared by every
// configurable setting (token, base URL, output format, and future ones). A
// caller lists value Sources in precedence order — FromFlags, FromEnv, FromFile,
// FromStdin, and a trailing Default — and Resolve returns the first that yields
// along with its Provenance, leaving value validation to the caller (ADR-3).
//
// It owns no setting: flag names, env var names, the .glassfrogrc key, and the
// default value are all caller-supplied. It imports internal/rcfile for the file
// source (the one shared nearest-wins file walk — DECISIONS §74) and no domain
// package, so the 040 call sites can adopt it without an import cycle (ADR-1).
package resolve

// SourceKind classifies where a resolved value originated. It is also the
// vocabulary of Provenance.Kind. KindNone is the zero value, so a bare
// Resolution{} reports "nothing found".
type SourceKind int

const (
	// KindNone means nothing yielded and no default was present (the zero value).
	KindNone SourceKind = iota
	// KindFlag means a command flag supplied the value.
	KindFlag
	// KindEnv means an environment variable supplied the value.
	KindEnv
	// KindFile means a .glassfrogrc key supplied the value.
	KindFile
	// KindStdin means piped standard input supplied the value.
	KindStdin
	// KindDefault means the trailing default supplied the value.
	KindDefault
)

// String returns the lowercase token for each member (none/flag/env/file/stdin/
// default), for messages and tests.
func (k SourceKind) String() string {
	switch k {
	case KindNone:
		return "none"
	case KindFlag:
		return "flag"
	case KindEnv:
		return "env"
	case KindFile:
		return "file"
	case KindStdin:
		return "stdin"
	case KindDefault:
		return "default"
	default:
		return "unknown"
	}
}

// Provenance records where a resolved value came from. All fields are safe to
// display — never a secret — so a caller can phrase a validation error from them.
type Provenance struct {
	// Kind is which source kind won.
	Kind SourceKind
	// Origin is the concrete origin label: the flag name (e.g. "--output"), the
	// env var name (e.g. "GLASSFROG_BASE_URL"), or the resolved file path. Empty
	// for KindNone, KindDefault, and KindStdin.
	Origin string
}

// Resolution is the code-free output of Resolve.
type Resolution struct {
	// Value is the resolved raw value, verbatim; meaningful only when the returned
	// error is nil and Found() is true.
	Value string
	// Provenance is where Value came from.
	Provenance Provenance
}

// Found reports whether any source — including the default — yielded a value.
func (r Resolution) Found() bool { return r.Provenance.Kind != KindNone }

// Source is one origin in the precedence list. It is opaque to callers and
// constructed only via the constructors in this package (FromFlags, FromEnv,
// FromFile, FromStdin, Default). It carries its Kind — readable without
// evaluating, used by Resolve for the Stdin guard — and a lazy eval closure that
// Resolve runs only when the walk reaches it.
type Source struct {
	kind SourceKind
	// eval reports the source's outcome: yielded reports whether the source
	// supplied a value, value/origin describe it when it did, and a non-nil error
	// aborts the walk. Within a single Resolve call it is run at most once, only
	// when the walk reaches it (reusing the same Source across Resolve calls
	// re-runs it).
	eval func() (value string, origin string, yielded bool, err error)
}

// Kind returns the source's kind without evaluating it.
func (s Source) Kind() SourceKind { return s.kind }

// Flag is one flag input for FromFlags.
type Flag struct {
	// Name is the operator-facing label including dashes (e.g. "--output", "-o");
	// it becomes Provenance.Origin when this flag wins.
	Name string
	// Present reports whether the flag was supplied on the command line (cobra
	// Changed()).
	Present bool
	// Value is the flag's value.
	Value string
}
