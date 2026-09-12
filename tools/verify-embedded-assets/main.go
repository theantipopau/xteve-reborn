// Command verify-embedded-assets regenerates src/webUI.go from the html/
// folder, the same way `xteve -dev` does at startup, without needing to
// boot the full server (network binding, SSDP, etc). CI runs this and then
// diffs the result against the committed file to catch a stale bundle
// before it ships - see .github/workflows/ci.yml.
package main

import (
	"fmt"
	"go/format"
	"os"

	"xteve-reborn/src"
)

func main() {
	var goFile = "src" + string(os.PathSeparator) + "webUI.go"

	src.HTMLInit("webUI", "src", "html"+string(os.PathSeparator), goFile)

	if err := src.BuildGoFile(); err != nil {
		fmt.Fprintln(os.Stderr, "regeneration failed:", err)
		os.Exit(1)
	}

	// BuildGoFile emits unformatted source (two-space indents); gofmt it so
	// the result matches how the committed file is always saved, otherwise
	// every regeneration would show a spurious indentation-only diff.
	raw, err := os.ReadFile(goFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read failed:", err)
		os.Exit(1)
	}

	formatted, err := format.Source(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gofmt failed:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(goFile, formatted, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write failed:", err)
		os.Exit(1)
	}
}
