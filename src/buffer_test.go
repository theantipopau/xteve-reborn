package src

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// setActiveConnections simulates viewers already connected to the given
// upstream URLs by populating the buffer's client registry the same way a real
// streaming session does (keys are playlist ID + URL hash, so the suffix match
// in activeClientConnections finds them).
func setActiveConnections(t *testing.T, busy map[string]int) {

	t.Helper()

	var keys []string

	for streamURL, connections := range busy {

		var key = "testplaylist" + getMD5(streamURL)
		keys = append(keys, key)

		BufferClients.Store(key, ClientConnection{Connection: connections})

	}

	t.Cleanup(func() {
		for _, key := range keys {
			BufferClients.Delete(key)
		}
	})

}

func TestResolveReachableStreamURLPrefersWorkingPrimary(t *testing.T) {

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer primary.Close()

	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backup.Close()

	streamInfo := StreamInfo{URL: primary.URL, BackupURL1: backup.URL}

	got := resolveReachableStreamURL(streamInfo)
	if got != primary.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the primary URL %q (it was reachable and idle, no need to fail over)", got, primary.URL)
	}
}

func TestResolveReachableStreamURLFallsBackToFirstWorkingBackup(t *testing.T) {

	deadPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer deadPrimary.Close()

	deadBackup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer deadBackup.Close()

	workingBackup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer workingBackup.Close()

	streamInfo := StreamInfo{
		URL:        deadPrimary.URL,
		BackupURL1: deadBackup.URL,
		BackupURL2: workingBackup.URL,
		BackupURL3: "http://127.0.0.1:1/unreachable",
	}

	got := resolveReachableStreamURL(streamInfo)
	if got != workingBackup.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the first working backup %q (it is reachable and idle)", got, workingBackup.URL)
	}
}

func TestResolveReachableStreamURLFallsBackToPrimaryWhenNothingWorks(t *testing.T) {

	deadPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer deadPrimary.Close()

	streamInfo := StreamInfo{
		URL:        deadPrimary.URL,
		BackupURL1: "http://127.0.0.1:1/unreachable",
	}

	// Nothing is reachable, so a channel with no working backup must behave
	// exactly as it did before backup channels existed: use the primary and
	// let the existing downstream error handling take over.
	got := resolveReachableStreamURL(streamInfo)
	if got != deadPrimary.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the primary URL %q unchanged when nothing is reachable", got, deadPrimary.URL)
	}
}

func TestIsStreamURLReachable(t *testing.T) {

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()

	notFound := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFound.Close()

	if !isStreamURLReachable(ok.URL) {
		t.Error("isStreamURLReachable() = false for a 200 OK server, want true")
	}
	if isStreamURLReachable(notFound.URL) {
		t.Error("isStreamURLReachable() = true for a 404 server, want false")
	}
	if isStreamURLReachable("") {
		t.Error("isStreamURLReachable() = true for an empty URL, want false")
	}
	if isStreamURLReachable("http://127.0.0.1:1/unreachable") {
		t.Error("isStreamURLReachable() = true for an unreachable address, want false")
	}
}

// newTestStreamInfoWithLoad builds a StreamInfo from a primary URL and optional
// backup URLs, and simulates connection counts for sources that are already
// serving streams (sources absent from the map count as idle).
func newTestStreamInfoWithLoad(t *testing.T, primary string, busy map[string]int, backups ...string) StreamInfo {

	t.Helper()

	setActiveConnections(t, busy)

	var streamInfo = StreamInfo{URL: primary}

	if len(backups) > 0 {
		streamInfo.BackupURL1 = backups[0]
	}
	if len(backups) > 1 {
		streamInfo.BackupURL2 = backups[1]
	}
	if len(backups) > 2 {
		streamInfo.BackupURL3 = backups[2]
	}

	return streamInfo
}

// Load-aware selection, ported from the idea in c0y0t3d3n's iptv project
// ("pick the source with the most free connections"): when the primary is
// reachable but already serving streams and an idle backup exists, the idle
// backup wins even though the primary answers first.
func TestResolveReachableStreamURLPrefersIdleBackupOverBusyPrimary(t *testing.T) {

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer primary.Close()

	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backup.Close()

	streamInfo := newTestStreamInfoWithLoad(t, primary.URL, map[string]int{primary.URL: 2}, backup.URL)

	got := resolveReachableStreamURL(streamInfo)
	if got != backup.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the idle backup %q (the primary is already serving 2 streams)", got, backup.URL)
	}
}

// A reachable source with no active viewers is preferred over every source
// that already has viewers, whichever URL it was configured on.
func TestResolveReachableStreamURLPrefersIdleSourceWhereverConfigured(t *testing.T) {

	deadPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer deadPrimary.Close()

	busyBackup1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer busyBackup1.Close()

	idleBackup2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer idleBackup2.Close()

	streamInfo := newTestStreamInfoWithLoad(t, deadPrimary.URL,
		map[string]int{busyBackup1.URL: 1},
		busyBackup1.URL, idleBackup2.URL)

	got := resolveReachableStreamURL(streamInfo)
	if got != idleBackup2.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the idle backup %q (the other backup is already serving 1 stream)", got, idleBackup2.URL)
	}
}

// When every reachable source is already busy, the one with the fewest active
// viewers wins - spreading viewers across providers instead of stacking them
// on one account until the provider cuts it off for too many connections.
func TestResolveReachableStreamURLPicksLeastLoadedSourceWhenAllBusy(t *testing.T) {

	deadPrimary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer deadPrimary.Close()

	loadedBackup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer loadedBackup.Close()

	quietBackup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer quietBackup.Close()

	streamInfo := newTestStreamInfoWithLoad(t, deadPrimary.URL,
		map[string]int{loadedBackup.URL: 3, quietBackup.URL: 1},
		loadedBackup.URL, quietBackup.URL)

	got := resolveReachableStreamURL(streamInfo)
	if got != quietBackup.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want the least-loaded backup %q (1 active stream vs 3)", got, quietBackup.URL)
	}
}
