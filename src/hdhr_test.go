package src

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTunerTest puts the globals the HDHomeRun/stream handlers depend on
// into a known state and restores them afterwards.
func setupTunerTest(t *testing.T) {

	var oldSettings = Settings
	var oldProtocol = System.ServerProtocol
	var oldURLS = System.File.URLS
	var oldDeviceID = System.DeviceID

	System.ServerProtocol.DVR = "http"
	System.ServerProtocol.WEB = "http"
	System.ServerProtocol.M3U = "http"
	System.ServerProtocol.XML = "http"
	System.DeviceID = "test-device"
	System.File.URLS = filepath.Join(t.TempDir(), "urls.json")
	os.WriteFile(System.File.URLS, []byte("{}"), 0644)

	Settings.Tuner = 3
	Settings.EpgSource = "XEPG"
	Settings.Buffer = "-"
	Settings.AuthenticationPMS = false

	streamingURLsLock.Lock()
	Data.Cache.StreamingURLS = make(map[string]StreamInfo)
	streamingURLsLock.Unlock()

	t.Cleanup(func() {
		Settings = oldSettings
		System.ServerProtocol = oldProtocol
		System.File.URLS = oldURLS
		System.DeviceID = oldDeviceID
		endScan()
	})
}

func get(t *testing.T, handler http.HandlerFunc, path string) *httptest.ResponseRecorder {

	var r = httptest.NewRequest("GET", "http://tuner.local:34400"+path, nil)
	var w = httptest.NewRecorder()
	handler(w, r)

	return w
}

func TestDiscoverJSON(t *testing.T) {

	setupTunerTest(t)

	var w = get(t, Index, "/discover.json")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}

	var discover Discover
	if err := json.Unmarshal(w.Body.Bytes(), &discover); err != nil {
		t.Fatal(err)
	}

	if discover.LineupURL != "http://tuner.local:34400/lineup.json" {
		t.Errorf("LineupURL = %q", discover.LineupURL)
	}

	if discover.TunerCount != 3 || discover.DeviceID != "test-device" {
		t.Errorf("TunerCount=%d DeviceID=%q", discover.TunerCount, discover.DeviceID)
	}
}

func TestLineupStatusReflectsScan(t *testing.T) {

	setupTunerTest(t)

	var status LineupStatus
	json.Unmarshal(get(t, Index, "/lineup_status.json").Body.Bytes(), &status)
	if status.ScanInProgress != 0 {
		t.Errorf("idle: ScanInProgress = %d, want 0", status.ScanInProgress)
	}

	tryStartScan()
	json.Unmarshal(get(t, Index, "/lineup_status.json").Body.Bytes(), &status)
	if status.ScanInProgress != 1 {
		t.Errorf("scanning: ScanInProgress = %d, want 1", status.ScanInProgress)
	}
}

func setChannels(channels map[string]interface{}) {
	xepgLock.Lock()
	Data.XEPG.Channels = channels
	xepgLock.Unlock()
}

func channel(name, number, url string, active bool, backup string) map[string]interface{} {
	return map[string]interface{}{
		"_file.m3u.id":       "M1",
		"x-name":             name,
		"x-channelID":        number,
		"url":                url,
		"x-active":           active,
		"x-backup-channel-1": backup,
	}
}

// TestLineupToStreamRedirect walks the path Plex takes: fetch lineup.json,
// then request one channel's stream URL, which (with no buffer) must
// redirect to the provider's real URL. Inactive channels must not appear.
func TestLineupToStreamRedirect(t *testing.T) {

	setupTunerTest(t)

	setChannels(map[string]interface{}{
		"x-ID.0": channel("News", "1000", "http://provider.example/news", true, ""),
		"x-ID.1": channel("Hidden", "1001", "http://provider.example/hidden", false, ""),
	})

	var lineup []LineupStream
	if err := json.Unmarshal(get(t, Index, "/lineup.json").Body.Bytes(), &lineup); err != nil {
		t.Fatal(err)
	}

	if len(lineup) != 1 || lineup[0].GuideName != "News" || lineup[0].GuideNumber != "1000" {
		t.Fatalf("lineup = %+v, want only the active News channel", lineup)
	}

	var streamPath = strings.TrimPrefix(lineup[0].URL, "http://tuner.local:34400")
	if strings.HasPrefix(streamPath, "/stream/") == false {
		t.Fatalf("stream URL %q not under /stream/", lineup[0].URL)
	}

	var w = get(t, Stream, streamPath)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "http://provider.example/news" {
		t.Errorf("stream: status %d Location %q, want 302 to the provider URL", w.Code, w.Header().Get("Location"))
	}

	// A client appending a query string must still resolve the channel.
	w = get(t, Stream, streamPath+"?session=abc")
	if w.Code != http.StatusFound {
		t.Errorf("stream with query string: status %d, want 302", w.Code)
	}
}

func TestStreamUnknownIDIs404(t *testing.T) {

	setupTunerTest(t)

	if w := get(t, Stream, "/stream/does-not-exist"); w.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", w.Code)
	}
}

// TestStreamFailsOverToBackup: a channel whose primary provider is down
// should be redirected to its first reachable backup.
func TestStreamFailsOverToBackup(t *testing.T) {

	setupTunerTest(t)

	var backup = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer backup.Close()

	var dead = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	var deadURL = dead.URL + "/stream"
	dead.Close()

	setChannels(map[string]interface{}{
		"x-ID.0": channel("Sport", "1000", deadURL, true, backup.URL+"/stream"),
	})

	var lineup []LineupStream
	json.Unmarshal(get(t, Index, "/lineup.json").Body.Bytes(), &lineup)
	if len(lineup) != 1 {
		t.Fatalf("lineup = %+v", lineup)
	}

	var w = get(t, Stream, strings.TrimPrefix(lineup[0].URL, "http://tuner.local:34400"))
	if w.Header().Get("Location") != backup.URL+"/stream" {
		t.Errorf("redirected to %q, want the backup %q", w.Header().Get("Location"), backup.URL+"/stream")
	}
}
