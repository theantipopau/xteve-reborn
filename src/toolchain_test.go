package src

import (
	"os"
	"path/filepath"
	"testing"
)

// writeByteToFile replaces the file through a rename and leaves no temp files.
func TestWriteByteToFileReplaces(t *testing.T) {

	var dir = t.TempDir()
	var file = filepath.Join(dir, "settings.json")

	for _, content := range []string{"first, longer content", "second"} {
		if err := writeByteToFile(file, []byte(content)); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(file)
		if err != nil || string(got) != content {
			t.Fatalf("read back %q, %v; want %q", got, err, content)
		}
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("folder has %d entries, want only settings.json", len(entries))
	}
}
