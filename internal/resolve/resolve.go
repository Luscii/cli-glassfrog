package resolve

// Resolve walks sources in order (index 0 = highest precedence) and returns the
// first that yields. Evaluation is lazy: once a source yields, no lower source is
// evaluated. A source that errs aborts the walk and returns that error verbatim
// (no fall-through). When no source yields and no Default is present, it returns
// Resolution{Provenance: {Kind: KindNone}} with a nil error — a valid empty
// outcome, not a failure.
//
// It PANICS if more than one Stdin source is supplied: STDIN is a single
// consumable stream, so composing two readers is a wiring bug, not a runtime
// input condition (ADR-5, consistent with the nil-seam fail-fast convention). The
// panic fires before any source is evaluated, so the stream is never drained.
//
// Resolve runs no validator and emits no diagnostics: it returns the raw winning
// value (which, for the token setting, is a secret) for the caller to validate
// and never formats Value into a message (ADR-3, secret hygiene).
func Resolve(sources ...Source) (Resolution, error) {
	// Inspect sources before walking so both wiring guards fire before any eval —
	// never draining the stream for a would-be first reader.
	stdinSources := 0
	for _, s := range sources {
		// A zero-value Source has a nil eval — it can only arise from a caller
		// bypassing the constructors (the fields are unexported). Fail fast with a
		// clear message rather than a cryptic nil-pointer panic in the walk below,
		// consistent with the nil-seam fail-fast convention.
		if s.eval == nil {
			panic("resolve.Resolve: zero-value Source — construct sources via FromFlags/FromEnv/FromFile/FromStdin/Default")
		}
		if s.kind == KindStdin {
			stdinSources++
		}
	}
	if stdinSources > 1 {
		panic("resolve.Resolve: at most one Stdin source per resolution")
	}

	for _, s := range sources {
		value, origin, yielded, err := s.eval()
		if err != nil {
			return Resolution{}, err // resolution error → abort, no fall-through
		}
		if yielded {
			return Resolution{
				Value:      value,
				Provenance: Provenance{Kind: s.kind, Origin: origin},
			}, nil
		}
		// Empty source → continue to the next, lower-precedence one.
	}

	return Resolution{Provenance: Provenance{Kind: KindNone}}, nil
}
