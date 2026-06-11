package resolve

import "strings"

// FromFlags yields the value of the first Present flag, in argument order. A
// Present flag yields even when its Value is empty — presence-based, matching
// cobra's Changed() (so an explicit --flag= counts). Provenance.Origin is that
// flag's Name. Walking several flags supports aliases (e.g. --output and -o); the
// winner's Name identifies which alias was supplied.
func FromFlags(flags ...Flag) Source {
	return Source{
		kind: KindFlag,
		eval: func() (string, string, bool, error) {
			for _, f := range flags {
				if f.Present {
					return f.Value, f.Name, true, nil
				}
			}
			return "", "", false, nil
		},
	}
}

// FromEnv yields the first non-empty value among names, looked up via lookup, in
// argument order. A value that is empty after trimming does not yield (the walk
// continues). The yielded value is returned verbatim (untrimmed); only the
// presence check trims. Provenance.Origin is the name that yielded.
func FromEnv(lookup func(string) string, names ...string) Source {
	return Source{
		kind: KindEnv,
		eval: func() (string, string, bool, error) {
			for _, name := range names {
				value := lookup(name)
				if strings.TrimSpace(value) != "" {
					return value, name, true, nil
				}
			}
			return "", "", false, nil
		},
	}
}

// Default always yields value, with Provenance{Kind: KindDefault} and an empty
// Origin. Place it last in the precedence list to backstop an otherwise-empty
// chain; omit it for an optional setting, where a KindNone result is the normal
// "nothing found" outcome.
func Default(value string) Source {
	return Source{
		kind: KindDefault,
		eval: func() (string, string, bool, error) {
			return value, "", true, nil
		},
	}
}
