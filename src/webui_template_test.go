package src

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

// stripJSComments removes whole-line // comments, so a placeholder that only
// survives inside commented-out code is ignored rather than reported as a
// missing label.
func stripJSComments(content string) string {

	var kept []string

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		kept = append(kept, line)
	}

	return strings.Join(kept, "\n")

}

// Every placeholder the web UI files contain must resolve against the language
// file. Go templates fail hard on a missing map key, and the server would then
// serve the page half-rendered - or, for a JS file, truncated mid-function -
// which is what a missing label in html/lang/en.json actually looks like in the
// browser. Adding a `case` to the settings UI without its label is the easy way
// to trip this.
func TestWebUITemplatesResolve(t *testing.T) {

	var langBytes, err = os.ReadFile(filepath.Join("..", "html", "lang", "en.json"))
	if err != nil {
		t.Fatal(err)
	}

	var lang = jsonToMap(string(langBytes))

	var files = []string{
		"index.html",
		filepath.Join("js", "settings_ts.js"),
		filepath.Join("js", "base_ts.js"),
		filepath.Join("js", "menu_ts.js"),
	}

	for _, file := range files {

		var content, readErr = os.ReadFile(filepath.Join("..", "html", file))
		if readErr != nil {
			t.Fatal(readErr)
		}

		// Placeholders that only exist in commented-out code (dead menu entries)
		// must not fail this check.
		var tmpl, parseErr = template.New("template").Parse(stripJSComments(string(content)))
		if parseErr != nil {
			t.Errorf("%s: template did not parse: %v", file, parseErr)
			continue
		}

		var rendered bytes.Buffer
		if execErr := tmpl.Execute(&rendered, lang); execErr != nil {
			t.Errorf("%s: template did not render against en.json: %v", file, execErr)
			continue
		}

		var output = rendered.String()

		if strings.Contains(output, "{{") {
			t.Errorf("%s: a placeholder was left unresolved", file)
		}

		if strings.Contains(output, "<no value>") {
			t.Errorf("%s: a placeholder resolved to nothing", file)
		}

	}

}
