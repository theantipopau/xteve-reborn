package up2date

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseChecksum(t *testing.T) {

	var checksums = "aaa111  xteve-reborn_3.0.1_linux_amd64.zip\n" +
		"bbb222 *xteve-reborn_3.0.1_windows_amd64.zip\n"

	got, err := parseChecksum(checksums, "xteve-reborn_3.0.1_windows_amd64.zip")
	if err != nil || got != "bbb222" {
		t.Errorf("got %q, %v; want bbb222", got, err)
	}

	if _, err = parseChecksum(checksums, "missing.zip"); err == nil {
		t.Error("expected an error for a file with no listed checksum")
	}
}

func writeZip(t *testing.T, entries map[string]string) string {

	var buf bytes.Buffer
	var w = zip.NewWriter(&buf)

	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte(content))
	}

	w.Close()

	var path = filepath.Join(t.TempDir(), "update.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	return path
}

// TestExtractFileOnlyTakesExactEntry: a crafted archive with a traversal
// path must not be extracted, and must not be confused with the real binary.
func TestExtractFileOnlyTakesExactEntry(t *testing.T) {

	var archive = writeZip(t, map[string]string{
		"../../evil":   "bad",
		"xteve-reborn": "good",
	})

	var out bytes.Buffer
	if err := extractFile(archive, "xteve-reborn", &out); err != nil {
		t.Fatal(err)
	}

	if out.String() != "good" {
		t.Errorf("extracted %q, want %q", out.String(), "good")
	}

	if err := extractFile(archive, "../evil", &out); err == nil {
		t.Error("expected an error for a name that isn't an exact entry")
	}
}

// TestGetLatestReleaseMatchesAssets checks platform-asset matching, that the
// checksums file is picked up, and that a release without an asset for this
// platform is skipped without carrying its checksums URL into the result.
func TestGetLatestReleaseMatchesAssets(t *testing.T) {

	var platform = runtime.GOOS + "_" + runtime.GOARCH

	var server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[
			{"tag_name": "v3.0.3", "draft": true, "assets": [
				{"name": "xteve-reborn_3.0.3_%[1]s.zip", "browser_download_url": "http://x/draft.zip"}
			]},
			{"tag_name": "v3.0.2", "draft": false, "assets": [
				{"name": "xteve-reborn_3.0.2_other_arch.zip", "browser_download_url": "http://x/other.zip"},
				{"name": "xteve-reborn_3.0.2_checksums.txt", "browser_download_url": "http://x/wrong-checksums.txt"}
			]},
			{"tag_name": "v3.0.1", "draft": false, "assets": [
				{"name": "xteve-reborn_3.0.1_%[1]s.zip", "browser_download_url": "http://x/right.zip"},
				{"name": "xteve-reborn_3.0.1_checksums.txt", "browser_download_url": "http://x/right-checksums.txt"}
			]}
		]`, platform)
	}))
	defer server.Close()

	var oldBase = apiBaseURL
	apiBaseURL = server.URL
	defer func() { apiBaseURL = oldBase }()

	release, err := GetLatestRelease("owner", "repo", "xteve-reborn")
	if err != nil {
		t.Fatal(err)
	}

	if release.Found == false || release.Tag != "v3.0.1" {
		t.Fatalf("got %+v, want v3.0.1", release)
	}

	if release.ZipURL != "http://x/right.zip" || release.ChecksumsURL != "http://x/right-checksums.txt" {
		t.Errorf("got zip %q checksums %q", release.ZipURL, release.ChecksumsURL)
	}
}

func TestDoUpdateRefusesWithoutChecksums(t *testing.T) {
	if err := DoUpdate("http://x/a.zip", "", "xteve-reborn"); err == nil {
		t.Error("expected DoUpdate to refuse an update with no checksums file")
	}
}
