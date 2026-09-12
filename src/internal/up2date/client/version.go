package up2date

import (
	"strconv"
	"strings"
)

// IsNewer reports whether latest represents a newer release than current.
// Both are expected in this fork's tag format, "vMAJOR.MINOR.PATCH" with an
// optional "-prerelease" suffix (e.g. "v3.0.0-pre.2"). A plain string
// comparison doesn't work here - lexically "v3.0.0-pre.10" < "v3.0.0-pre.2"
// even though pre.10 is the newer build - so the numeric core and the
// prerelease identifiers are compared piece by piece, following normal
// semver precedence: a release with no prerelease suffix outranks any
// prerelease of the same core version, and where both have one, its
// dot-separated components are compared in turn (numerically if both
// sides parse as numbers, lexically otherwise).
func IsNewer(current, latest string) bool {

	if current == latest {
		return false
	}

	var currentCore, currentPre = splitVersion(current)
	var latestCore, latestPre = splitVersion(latest)

	switch compareCores(currentCore, latestCore) {
	case -1:
		return true
	case 1:
		return false
	}

	// Same core version: no prerelease beats any prerelease.
	if currentPre == "" && latestPre == "" {
		return false
	}
	if currentPre == "" {
		return false
	}
	if latestPre == "" {
		return true
	}

	return comparePrerelease(currentPre, latestPre) < 0
}

// splitVersion strips a leading "v" and separates the "MAJOR.MINOR.PATCH"
// core from an optional "-prerelease" suffix.
func splitVersion(version string) (core, prerelease string) {

	version = strings.TrimPrefix(version, "v")

	if i := strings.Index(version, "-"); i >= 0 {
		return version[:i], version[i+1:]
	}

	return version, ""
}

// compareCores compares two "MAJOR.MINOR.PATCH" strings component by
// component, returning -1/0/1 the way strings.Compare does. A missing or
// non-numeric component is treated as 0, so a malformed core never panics -
// just compares as if it were absent.
func compareCores(a, b string) int {

	var partsA = strings.Split(a, ".")
	var partsB = strings.Split(b, ".")

	for i := 0; i < 3; i++ {

		var na, nb int

		if i < len(partsA) {
			na, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			nb, _ = strconv.Atoi(partsB[i])
		}

		if na != nb {
			if na < nb {
				return -1
			}
			return 1
		}

	}

	return 0
}

// comparePrerelease compares two prerelease strings ("pre", "pre.2") by
// dot-separated component, numerically where both sides of a component
// parse as numbers (so "pre.2" < "pre.10") and lexically otherwise (so
// "beta" < "pre" the same way semver orders identifier strings). A
// prerelease with more components than the other, up to the point they
// agree, is considered newer - matching semver's rule that
// "1.0.0-pre" < "1.0.0-pre.0".
func comparePrerelease(a, b string) int {

	var partsA = strings.Split(a, ".")
	var partsB = strings.Split(b, ".")

	for i := 0; i < len(partsA) || i < len(partsB); i++ {

		if i >= len(partsA) {
			return -1
		}
		if i >= len(partsB) {
			return 1
		}

		var pa, pb = partsA[i], partsB[i]

		na, errA := strconv.Atoi(pa)
		nb, errB := strconv.Atoi(pb)

		if errA == nil && errB == nil {
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
			continue
		}

		if pa != pb {
			if pa < pb {
				return -1
			}
			return 1
		}

	}

	return 0
}
