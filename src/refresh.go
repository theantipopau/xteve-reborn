package src

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// providerLock serializes writes to the provider maps in Settings.Files
// (getProviderData updates download counters/timestamps in them) with the
// JSON snapshot of Settings sent in WS responses, so a background refresh
// can't mutate those maps while a response is being encoded.
var providerLock sync.Mutex

// refreshLock serializes whole provider-refresh sequences (download, then
// rebuild) so the scheduled update, the source-change check and a manual
// "Update" click never interleave.
var refreshLock sync.Mutex

// sourceCheckRunning is set while a source-change check is in progress.
var sourceCheckRunning int32

// refreshStateLock guards everything below it.
var refreshStateLock sync.Mutex

var lastRefresh time.Time
var nextSourceCheck time.Time

// sourceValidators remembers each remote source's ETag/Last-Modified so the
// next check can be a conditional request: a server that supports them
// answers "304 Not Modified" without resending the whole file.
var sourceValidators = make(map[string]sourceValidator)

type sourceValidator struct {
	etag         string
	lastModified string
}

type providerRef struct {
	fileType string
	id       string
}

// markRefreshed records that the guide/lineup was just rebuilt, for the
// dashboard's "Guide refreshed" display.
func markRefreshed() {
	refreshStateLock.Lock()
	lastRefresh = time.Now()
	refreshStateLock.Unlock()
}

// refreshTimes returns the dashboard's "Guide refreshed" and "Next source
// check" values, already formatted for display.
func refreshTimes() (last, next string) {

	refreshStateLock.Lock()
	var l, n = lastRefresh, nextSourceCheck
	refreshStateLock.Unlock()

	last = formatRefreshTime(l)

	switch {
	case Settings.SourceCheckInterval <= 0:
		next = "off"
	case n.IsZero():
		next = "soon"
	default:
		next = formatRefreshTime(n)
	}

	return
}

func formatRefreshTime(t time.Time) string {

	if t.IsZero() {
		return "never"
	}

	var now = time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}

	return t.Format("2 Jan 15:04")
}

// snapshotSettings returns a deep copy of Settings for a WS response, taken
// under providerLock so the nested provider maps can't change mid-copy.
func snapshotSettings() (settings SettingsStruct) {

	providerLock.Lock()
	defer providerLock.Unlock()

	json.Unmarshal([]byte(mapToJSON(Settings)), &settings)
	return
}

// waitForScan blocks until any running rebuild has finished, so a refresh
// that's about to rebuild doesn't get silently skipped by buildXEPG's
// "already running" guard. Gives up after ten minutes rather than hang.
func waitForScan() {

	for i := 0; i < 600 && scanInProgress(); i++ {
		time.Sleep(time.Second)
	}
}

// refreshProviders re-downloads the given provider files and rebuilds the
// lineup and guide once for all of them.
func refreshProviders(items []providerRef) (err error) {

	refreshLock.Lock()
	defer refreshLock.Unlock()

	waitForScan()

	var ok = 0

	for _, item := range items {

		if errNew := getProviderData(item.fileType, item.id); errNew != nil {
			err = errNew
			continue
		}

		ok++
	}

	if ok == 0 {
		return
	}

	if errNew := buildDatabaseDVR(); errNew != nil {
		return errNew
	}

	buildXEPG(false)

	return
}

// sourceCheckDue reports whether it's time for the next periodic check.
func sourceCheckDue(now time.Time) bool {

	if Settings.SourceCheckInterval <= 0 {
		return false
	}

	refreshStateLock.Lock()
	defer refreshStateLock.Unlock()

	if nextSourceCheck.IsZero() {
		// First check happens one interval after startup - startup itself
		// already refreshes every source when "Update at startup" is on.
		nextSourceCheck = now.Add(time.Duration(Settings.SourceCheckInterval) * time.Minute)
		return false
	}

	return now.After(nextSourceCheck)
}

// runSourceCheck checks every M3U (and, with XEPG, XMLTV) source for changes
// and refreshes just the ones whose content actually differs from the local
// copy. Returns how many changed.
func runSourceCheck() (changed int, err error) {

	if atomic.CompareAndSwapInt32(&sourceCheckRunning, 0, 1) == false {
		return 0, errors.New("a source check is already running")
	}
	defer atomic.StoreInt32(&sourceCheckRunning, 0)

	defer func() {
		refreshStateLock.Lock()
		if Settings.SourceCheckInterval > 0 {
			nextSourceCheck = time.Now().Add(time.Duration(Settings.SourceCheckInterval) * time.Minute)
		}
		refreshStateLock.Unlock()
	}()

	var items = changedSources()
	if len(items) == 0 {
		showDebug("Source Check:No changes", 1)
		return 0, nil
	}

	showInfo(fmt.Sprintf("Source Check:%d source(s) changed, refreshing", len(items)))

	err = refreshProviders(items)
	return len(items), err
}

// changedSources returns the provider files whose source content differs
// from the local copy last downloaded.
func changedSources() (items []providerRef) {

	var fileTypes = []string{"m3u"}
	if Settings.EpgSource == "XEPG" {
		fileTypes = append(fileTypes, "xmltv")
	}

	type source struct {
		ref    providerRef
		source string
		name   string
	}

	var sources []source

	providerLock.Lock()
	for _, fileType := range fileTypes {

		var files = Settings.Files.M3U
		if fileType == "xmltv" {
			files = Settings.Files.XMLTV
		}

		for id, d := range files {
			if data, ok := d.(map[string]interface{}); ok {
				if s, ok := data["file.source"].(string); ok && len(s) > 0 {
					var name, _ = data["name"].(string)
					sources = append(sources, source{providerRef{fileType, id}, s, name})
				}
			}
		}

	}
	providerLock.Unlock()

	for _, s := range sources {

		changed, err := sourceChanged(s.ref, s.source)
		if err != nil {
			showInfo(fmt.Sprintf("Source Check:Could not check %s (%s)", s.name, err))
			continue
		}

		if changed {
			items = append(items, s.ref)
		}

	}

	return
}

// sourceChanged fetches a source and compares it with the local copy
// getProviderData last saved. Remote sources are requested conditionally
// when the server supplied an ETag/Last-Modified before.
func sourceChanged(ref providerRef, source string) (changed bool, err error) {

	var extension = ".m3u"
	if ref.fileType == "xmltv" {
		extension = ".xml"
	}

	local, err := os.ReadFile(System.Folder.Data + ref.id + extension)
	if err != nil {
		// No local copy (e.g. deleted by hand): refresh it.
		return true, nil
	}

	var body []byte
	var validator sourceValidator
	var remote = strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")

	if remote {

		req, err := http.NewRequest("GET", source, nil)
		if err != nil {
			return false, err
		}
		req.Header.Set("User-Agent", Settings.UserAgent)

		refreshStateLock.Lock()
		var previous = sourceValidators[source]
		refreshStateLock.Unlock()

		if len(previous.etag) > 0 {
			req.Header.Set("If-None-Match", previous.etag)
		}
		if len(previous.lastModified) > 0 {
			req.Header.Set("If-Modified-Since", previous.lastModified)
		}

		resp, err := fileDownloadHTTPClient.Do(req)
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotModified {
			return false, nil
		}

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}

		validator = sourceValidator{etag: resp.Header.Get("ETag"), lastModified: resp.Header.Get("Last-Modified")}

	} else {

		body, err = os.ReadFile(source)
		if err != nil {
			return false, err
		}

	}

	body, err = extractGZIP(body, source)
	if err != nil {
		return false, err
	}

	changed = bytes.Equal(body, local) == false

	// Only trust the validators once the local copy is known to match what
	// they describe. Storing them for a changed source would make the next
	// check get a 304 even if this refresh fails, and the change would be
	// missed for good.
	if remote && changed == false {
		refreshStateLock.Lock()
		sourceValidators[source] = validator
		refreshStateLock.Unlock()
	}

	return
}
