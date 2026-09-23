package src

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// fakeSource serves body with an ETag and honours If-None-Match, counting
// full (200) responses so tests can tell a conditional hit from a re-download.
type fakeSource struct {
	body      []byte
	etag      string
	fullSends int32
}

func (f *fakeSource) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("If-None-Match") == f.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	atomic.AddInt32(&f.fullSends, 1)
	w.Header().Set("ETag", f.etag)
	w.Write(f.body)
}

func withDataFolder(t *testing.T) string {

	var dir = t.TempDir() + string(os.PathSeparator)
	var old = System.Folder.Data

	System.Folder.Data = dir
	t.Cleanup(func() { System.Folder.Data = old })

	refreshStateLock.Lock()
	sourceValidators = make(map[string]sourceValidator)
	refreshStateLock.Unlock()

	return dir
}

func TestSourceUnchangedThenConditional(t *testing.T) {

	var dir = withDataFolder(t)
	var src = &fakeSource{body: []byte("#EXTM3U\nchannel"), etag: `"v1"`}
	var server = httptest.NewServer(src)
	defer server.Close()

	os.WriteFile(filepath.Join(dir, "M1.m3u"), src.body, 0644)
	var ref = providerRef{"m3u", "M1"}

	changed, err := sourceChanged(ref, server.URL)
	if err != nil || changed {
		t.Fatalf("first check: changed=%v err=%v, want unchanged", changed, err)
	}

	// Second check should be answered with 304 - no full re-download.
	changed, err = sourceChanged(ref, server.URL)
	if err != nil || changed {
		t.Fatalf("second check: changed=%v err=%v, want unchanged", changed, err)
	}

	if n := atomic.LoadInt32(&src.fullSends); n != 1 {
		t.Errorf("source sent the full file %d times, want 1 (second check should be a 304)", n)
	}
}

// TestSourceChangedIsDetectedAndNotCached: a changed source must report
// changed, and must not store its validators - otherwise a failed refresh
// would be followed by 304s and the change never picked up.
func TestSourceChangedIsDetectedAndNotCached(t *testing.T) {

	var dir = withDataFolder(t)
	var src = &fakeSource{body: []byte("<tv>new schedule</tv>"), etag: `"v2"`}
	var server = httptest.NewServer(src)
	defer server.Close()

	os.WriteFile(filepath.Join(dir, "X1.xml"), []byte("<tv>old schedule</tv>"), 0644)
	var ref = providerRef{"xmltv", "X1"}

	for i := 0; i < 2; i++ {
		changed, err := sourceChanged(ref, server.URL)
		if err != nil || changed == false {
			t.Fatalf("check %d: changed=%v err=%v, want changed", i, changed, err)
		}
	}

	if n := atomic.LoadInt32(&src.fullSends); n != 2 {
		t.Errorf("source sent the full file %d times, want 2 (no 304 while unrefreshed)", n)
	}
}

// TestGzippedSourceComparesDecompressed: getProviderData saves the
// decompressed file, so a gzipped source must be compared after decompression.
func TestGzippedSourceComparesDecompressed(t *testing.T) {

	var dir = withDataFolder(t)
	var plain = []byte("<tv>same</tv>")

	var gz bytes.Buffer
	var w = gzip.NewWriter(&gz)
	w.Write(plain)
	w.Close()

	var server = httptest.NewServer(&fakeSource{body: gz.Bytes(), etag: `"gz"`})
	defer server.Close()

	os.WriteFile(filepath.Join(dir, "X2.xml"), plain, 0644)

	changed, err := sourceChanged(providerRef{"xmltv", "X2"}, server.URL)
	if err != nil || changed {
		t.Errorf("changed=%v err=%v, want unchanged", changed, err)
	}
}

func TestLocalFileSourceAndMissingCopy(t *testing.T) {

	var dir = withDataFolder(t)

	var source = filepath.Join(t.TempDir(), "list.m3u")
	os.WriteFile(source, []byte("#EXTM3U\na"), 0644)
	os.WriteFile(filepath.Join(dir, "M2.m3u"), []byte("#EXTM3U\na"), 0644)

	if changed, err := sourceChanged(providerRef{"m3u", "M2"}, source); err != nil || changed {
		t.Errorf("identical local file: changed=%v err=%v", changed, err)
	}

	os.WriteFile(source, []byte("#EXTM3U\nb"), 0644)
	if changed, _ := sourceChanged(providerRef{"m3u", "M2"}, source); changed == false {
		t.Error("edited local file not detected as changed")
	}

	if changed, _ := sourceChanged(providerRef{"m3u", "missing"}, source); changed == false {
		t.Error("missing local copy should count as changed")
	}
}
