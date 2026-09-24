package src

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// setXtreamProviders registers playlist sources the way the app discovers
// them, so the capacity registry knows which hosts to ask.
func setXtreamProviders(t *testing.T, sources ...string) {

	t.Helper()

	var oldSettings = Settings
	var oldFiles = Settings.Files.M3U
	t.Cleanup(func() {
		Settings = oldSettings
		Settings.Files.M3U = oldFiles
		resetProviderCapacity()
	})

	var files = make(map[string]interface{})

	for i, source := range sources {
		files[fmt.Sprintf("M%d", i+1)] = map[string]interface{}{
			"name":        fmt.Sprintf("provider %d", i+1),
			"file.source": source,
			"type":        "m3u",
		}
	}

	providerLock.Lock()
	Settings.Files.M3U = files
	providerLock.Unlock()

	resetProviderCapacity()

}

// An Xtream playlist URL identifies the provider and its player_api endpoint;
// anything else must not be called.
func TestXtreamPlayerAPI(t *testing.T) {

	var cases = []struct {
		in     string
		wanted bool
	}{
		{"http://provider.example:8080/get.php?username=u&password=p&type=m3u_plus", true},
		{"http://provider.example:8080/player_api.php?username=u&password=p", true},
		{"http://provider.example/playlist.m3u", false},
		{"http://provider.example/get.php?username=u", false},
		{"", false},
	}

	for _, c := range cases {

		var got = xtreamPlayerAPI(c.in)

		if (len(got) > 0) != c.wanted {
			t.Errorf("xtreamPlayerAPI(%q) = %q, wanted a match: %t", c.in, got, c.wanted)
		}

	}

}

// The account's connection status is read from user_info - including the
// quoted numbers Xtream sends - and turned into free slots.
func TestProviderFreeSlotsReadsXtreamUserInfo(t *testing.T) {

	var api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "user_info" {
			t.Errorf("provider was asked for %q, want action=user_info", r.URL.RawQuery)
		}
		io.WriteString(w, `{"user_info":{"username":"u","active_cons":"1","max_connections":"3","status":"Active"}}`)
	}))
	defer api.Close()

	setXtreamProviders(t, api.URL+"/get.php?username=u&password=p&type=m3u_plus")

	if free, known := providerFreeSlots(api.URL + "/live/u/p/123.ts"); known {
		t.Fatalf("free slots = %d known before any refresh, want unknown", free)
	}

	refreshProviderCapacity()

	free, known := providerFreeSlots(api.URL + "/live/u/p/123.ts")

	if known == false {
		t.Fatal("catch-up: the account status was not registered for the provider host")
	}

	if free != 2 {
		t.Errorf("free slots = %d, want 2 (max 3, 1 in use)", free)
	}

	// A host that isn't a configured Xtream provider stays unknown, so
	// selection falls back to this instance's own viewer count.
	if _, known := providerFreeSlots("http://other.example/stream/1"); known {
		t.Error("an unknown provider reported a connection limit")
	}

}

// Numbers as JSON numbers (not quoted strings) must work too.
func TestProviderFreeSlotsAcceptsNumericUserInfo(t *testing.T) {

	var api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"user_info":{"active_cons":4,"max_connections":4,"status":"Active"}}`)
	}))
	defer api.Close()

	setXtreamProviders(t, api.URL+"/get.php?username=u&password=p")

	refreshProviderCapacity()

	free, known := providerFreeSlots(api.URL + "/live/u/p/123.ts")

	if known == false || free != 0 {
		t.Errorf("free slots = %d (known %t), want 0 known true", free, known)
	}

}

// When every candidate is equally loaded here, the provider account with more
// connections to spare must win - the whole point of consulting user_info.
func TestResolveReachableStreamURLPrefersProviderWithFreeConnections(t *testing.T) {

	var userInfo = func(activeCons, maxConnections string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/player_api.php" {
				fmt.Fprintf(w, `{"user_info":{"active_cons":"%s","max_connections":"%s","status":"Active"}}`, activeCons, maxConnections)
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	}

	// provider A: the account is saturated elsewhere (2 of 2 used by other
	// apps). provider B: one of five used.
	var providerA = httptest.NewServer(userInfo("2", "2"))
	defer providerA.Close()

	var providerB = httptest.NewServer(userInfo("1", "5"))
	defer providerB.Close()

	setXtreamProviders(t,
		providerA.URL+"/get.php?username=u&password=p&type=m3u_plus",
		providerB.URL+"/get.php?username=u&password=p&type=m3u_plus")

	// Both are reachable and both are already serving one viewer here, so only
	// the provider account status can separate them.
	var deadPrimary = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer deadPrimary.Close()

	resetStreamProbeCache()
	refreshProviderCapacity()

	var streamInfo = newTestStreamInfoWithLoad(t, deadPrimary.URL,
		map[string]int{providerA.URL: 1, providerB.URL: 1},
		providerA.URL, providerB.URL)

	var got = resolveReachableStreamURL(streamInfo)

	if got != providerB.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want provider B %q (provider A has no connections left on the account)", got, providerB.URL)
	}

}

// A saturated account must not be chosen just because it is quieter here.
func TestResolveReachableStreamURLAvoidsFullProvider(t *testing.T) {

	var full = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/player_api.php" {
			io.WriteString(w, `{"user_info":{"active_cons":"2","max_connections":"2","status":"Active"}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer full.Close()

	setXtreamProviders(t, full.URL+"/get.php?username=u&password=p&type=m3u_plus")

	var deadPrimary = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer deadPrimary.Close()

	var unknownBackup = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer unknownBackup.Close()

	resetStreamProbeCache()
	refreshProviderCapacity()

	// The full provider is idle here, the unknown one is already serving two -
	// normally the idle one would win, but its account cannot take another
	// connection, so the unknown-but-serving source is the better bet.
	var streamInfo = newTestStreamInfoWithLoad(t, deadPrimary.URL,
		map[string]int{unknownBackup.URL: 2},
		full.URL, unknownBackup.URL)

	var got = resolveReachableStreamURL(streamInfo)

	if got != unknownBackup.URL {
		t.Fatalf("resolveReachableStreamURL() = %q, want %q (the other provider's account is full)", got, unknownBackup.URL)
	}

}
