package resolve

import "github.com/Luscii/cli-glassfrog/internal/rcfile"

// FromFile yields the value for key from the nearest .glassfrogrc up the tree
// from startDir, then the home file, via rcfile.Resolve — the one shared
// nearest-wins file walk (DECISIONS §74); this package adds no second parser and
// owns no .glassfrogrc format knowledge. Provenance.Origin is the resolved file
// path. A missing or key-less file does not yield (the walk continues in
// Resolve). An unreadable or unparseable file errs with rcfile's typed
// *ReadError / *FormatError verbatim, which Resolve surfaces with no fall-through.
func FromFile(startDir, homeDir, key string) Source {
	return Source{
		kind: KindFile,
		eval: func() (string, string, bool, error) {
			value, path, found, err := rcfile.Resolve(startDir, homeDir, key)
			if err != nil {
				return "", "", false, err
			}
			if !found {
				return "", "", false, nil
			}
			return value, path, true, nil
		},
	}
}
