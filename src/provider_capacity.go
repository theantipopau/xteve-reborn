package src

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Xtream Codes providers publish the account's connection limit and current
// usage through user_info. That is the only way to know how loaded a provider
// is outside this instance: several apps (or several DVRs) can share one
// account, so "no viewer here" does not mean "a free connection there". For
// every other kind of provider the answer stays unknown, and source selection
// falls back to this instance's own viewer count.

const (
	// providerCapacityTTL is how long a user_info answer is trusted.
	providerCapacityTTL = 2 * time.Minute
	// providerCapacityMinInterval is the shortest gap between two checks, so
	// a stream request storm can't turn into a user_info storm.
	providerCapacityMinInterval = 30 * time.Second
	// providerCapacityTimeout bounds one user_info request.
	providerCapacityTimeout = 3 * time.Second
	// providerCapacityMaxHosts bounds how much of the memory this map keeps.
	providerCapacityMaxHosts = 256
)

type providerCapacity struct {
	maxConnections int
	activeCons     int
	checked        time.Time
	err            string
}

var providerCapacityLock sync.Mutex
var providerCapacityByHost = make(map[string]providerCapacity)
var providerCapacityRunning bool
var providerCapacityLastAttempt time.Time

// xtreamPlayerAPI reports whether a playlist URL is an Xtream Codes one and
// returns the player_api.php endpoint that reports the account status, or ""
// when the URL isn't Xtream (no username/password in it).
func xtreamPlayerAPI(sourceURL string) string {

	if len(sourceURL) == 0 {
		return ""
	}

	var parsed, err = url.Parse(sourceURL)
	if err != nil || len(parsed.Host) == 0 {
		return ""
	}

	var scheme = parsed.Scheme
	if len(scheme) == 0 {
		scheme = "http"
	}

	var query = parsed.Query()
	var username, password = query.Get("username"), query.Get("password")

	if len(username) == 0 || len(password) == 0 {
		return ""
	}

	return fmt.Sprintf("%s://%s/player_api.php?username=%s&password=%s&action=user_info",
		scheme, parsed.Host, url.QueryEscape(username), url.QueryEscape(password))

}

// xtreamProviders maps the host of every configured Xtream provider to its
// player_api.php endpoint.
func xtreamProviders() (providers map[string]string) {

	providers = make(map[string]string)

	providerLock.Lock()
	defer providerLock.Unlock()

	for _, d := range Settings.Files.M3U {

		var data, ok = d.(map[string]interface{})
		if ok == false {
			continue
		}

		var source, _ = data["file.source"].(string)
		var endpoint = xtreamPlayerAPI(source)
		if len(endpoint) == 0 {
			continue
		}

		if parsed, err := url.Parse(endpoint); err == nil {
			providers[parsed.Host] = endpoint
		}

	}

	return
}

// refreshProviderCapacity queries every configured Xtream provider's account
// status. Synchronous - the caller decides whether that is a background job -
// and single-flight, so a burst of stream requests can't turn into a burst of
// user_info calls.
func refreshProviderCapacity() {

	providerCapacityLock.Lock()

	if providerCapacityRunning || time.Since(providerCapacityLastAttempt) < providerCapacityMinInterval {
		providerCapacityLock.Unlock()
		return
	}

	providerCapacityRunning = true
	providerCapacityLastAttempt = time.Now()
	providerCapacityLock.Unlock()

	defer func() {
		providerCapacityLock.Lock()
		providerCapacityRunning = false
		providerCapacityLock.Unlock()
	}()

	for host, endpoint := range xtreamProviders() {

		var capacity, err = fetchProviderCapacity(endpoint)

		providerCapacityLock.Lock()
		if err != nil {
			// Keep the last numbers (they are still the best estimate) but
			// remember that they are stale and why.
			var previous = providerCapacityByHost[host]
			previous.err = err.Error()
			previous.checked = time.Now()
			providerCapacityByHost[host] = previous
		} else {
			providerCapacityByHost[host] = capacity
		}
		providerCapacityLock.Unlock()

	}

}

// fetchProviderCapacity reads one Xtream account's connection status.
func fetchProviderCapacity(endpoint string) (capacity providerCapacity, err error) {

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", Settings.UserAgent)

	var client = &http.Client{Timeout: providerCapacityTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("user_info: %s", resp.Status)
		return
	}

	var body struct {
		UserInfo map[string]interface{} `json:"user_info"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return
	}

	capacity.maxConnections = intFromJSON(body.UserInfo["max_connections"])
	capacity.activeCons = intFromJSON(body.UserInfo["active_cons"])
	capacity.checked = time.Now()

	if status, ok := body.UserInfo["status"].(string); ok && strings.EqualFold(status, "Active") == false {
		capacity.err = "account status: " + status
	}

	return
}

// providerFreeSlots reports how many connections the provider behind a stream
// URL still has free, and whether that is known at all. Only Xtream providers
// expose it.
func providerFreeSlots(streamURL string) (free int, known bool) {

	if len(streamURL) == 0 {
		return 0, false
	}

	var parsed, err = url.Parse(streamURL)
	if err != nil || len(parsed.Host) == 0 {
		return 0, false
	}

	providerCapacityLock.Lock()
	var capacity, ok = providerCapacityByHost[parsed.Host]
	providerCapacityLock.Unlock()

	if ok == false || capacity.maxConnections <= 0 || capacity.checked.IsZero() {
		return 0, false
	}

	return capacity.maxConnections - capacity.activeCons, true
}

// providerCapacityMaybeStale starts a background refresh when the cached
// account status is missing or older than its TTL. Called from the stream
// request path, so it must not block: it only inspects the cache and, at
// most, starts one goroutine.
func providerCapacityMaybeStale() {

	providerCapacityLock.Lock()

	var stale = true
	for _, capacity := range providerCapacityByHost {
		if capacity.err == "" && time.Since(capacity.checked) < providerCapacityTTL {
			stale = false
			break
		}
	}

	providerCapacityLock.Unlock()

	if stale == false {
		return
	}

	go refreshProviderCapacity()

}

// intFromJSON reads an integer Xtream may send either as a JSON number or as a
// quoted string ("max_connections":"2" is the common form).
func intFromJSON(value interface{}) int {

	switch v := value.(type) {
	case float64:
		return int(v)
	case string:
		var n, err = strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return n
		}
	}

	return 0
}

// resetProviderCapacity drops every cached account status and the rate limit
// bookkeeping. Used by the tests.
func resetProviderCapacity() {

	providerCapacityLock.Lock()
	providerCapacityByHost = make(map[string]providerCapacity)
	providerCapacityRunning = false
	providerCapacityLastAttempt = time.Time{}
	providerCapacityLock.Unlock()

}
