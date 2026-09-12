package up2date

import "testing"

func TestIsNewer(t *testing.T) {

	var cases = []struct {
		current string
		latest  string
		want    bool
	}{
		// Identical versions are never "newer".
		{"v3.0.0", "v3.0.0", false},
		{"v3.0.0-pre", "v3.0.0-pre", false},

		// Plain core version bumps.
		{"v3.0.0", "v3.0.1", true},
		{"v3.0.1", "v3.0.0", false},
		{"v3.0.0", "v3.1.0", true},
		{"v3.0.0", "v4.0.0", true},
		{"v4.0.0", "v3.9.9", false},

		// The bug a naive string comparison gets wrong: pre.10 is newer
		// than pre.2, but "v3.0.0-pre.10" < "v3.0.0-pre.2" lexically.
		{"v3.0.0-pre.2", "v3.0.0-pre.10", true},
		{"v3.0.0-pre.10", "v3.0.0-pre.2", false},

		// A stable release always outranks a prerelease of the same core
		// version, in either direction.
		{"v3.0.0-pre.2", "v3.0.0", true},
		{"v3.0.0", "v3.0.0-pre.2", false},

		// A bare "-pre" is older than any numbered "-pre.N" of the same
		// core version (semver: 1.0.0-pre < 1.0.0-pre.0).
		{"v3.0.0-pre", "v3.0.0-pre.2", true},
		{"v3.0.0-pre.2", "v3.0.0-pre", false},

		// A newer core version wins even if the current build has no
		// prerelease suffix and the candidate does.
		{"v3.0.0", "v3.1.0-pre", true},

		// No leading "v" should still work.
		{"3.0.0", "3.0.1", true},
	}

	for _, c := range cases {
		if got := IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}
