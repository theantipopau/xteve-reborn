package src

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestXEPGChannelsConcurrentAccess exercises the exact hazard xepgLock
// exists to prevent: Data.XEPG.Channels is written wholesale by some
// goroutines (mirroring saveXEpgMapping/createXEPGDatabase, triggered by a
// WS command) while others range over it or read individual entries
// (mirroring getLineup/buildM3U/setDefaultResponseData's response
// snapshotting) at the same time - which happens for real whenever an
// admin edits a mapping while the maintenance loop's scheduled provider
// refresh is rebuilding XEPG, or two browser tabs are open at once.
//
// Without a lock around every access, this reliably crashes the whole
// process with "fatal error: concurrent map read and map write" - not a
// benign, ignorable data race, an unrecoverable panic - and `go test -race`
// flags it immediately (verified by testing an unlocked version of this
// exact pattern before writing the fix). This test proves the locked
// version is actually race-free under `go test -race ./src/...`, not just
// that it compiles.
func TestXEPGChannelsConcurrentAccess(t *testing.T) {

	xepgLock.Lock()
	Data.XEPG.Channels = make(map[string]interface{})
	xepgLock.Unlock()

	var wg sync.WaitGroup

	// Writers: replace the whole map, like saveXEpgMapping/createXEPGDatabase.
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				newMap := make(map[string]interface{}, 10)
				for j := 0; j < 10; j++ {
					newMap[string(rune('a'+j))] = n*1000 + i
				}
				xepgLock.Lock()
				Data.XEPG.Channels = newMap
				xepgLock.Unlock()
			}
		}(w)
	}

	// Readers: snapshot-copy the whole map, like getLineup/buildM3U/
	// setDefaultResponseData do before handing data off to a client.
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				xepgLock.Lock()
				snapshot := make(map[string]interface{}, len(Data.XEPG.Channels))
				for k, v := range Data.XEPG.Channels {
					snapshot[k] = v
				}
				xepgLock.Unlock()
				_ = snapshot
			}
		}()
	}

	wg.Wait()
}

// New channels get the lowest free x-ID and the lowest free channel number
// from MappingFirstChannel up, skipping ones already taken in the database.
func TestCreateXEPGDatabaseAllocatesFreeIDsAndNumbers(t *testing.T) {

	var oldXEPG, oldFirst, oldActive = System.File.XEPG, Settings.MappingFirstChannel, Data.Streams.Active
	t.Cleanup(func() {
		System.File.XEPG, Settings.MappingFirstChannel, Data.Streams.Active = oldXEPG, oldFirst, oldActive
	})

	System.File.XEPG = filepath.Join(t.TempDir(), "xepg.json")
	var existing = `{"x-ID.1":{"x-channelID":"1001","_file.m3u.id":"M1","name":"Old","x-name":"Old"}}`
	if err := os.WriteFile(System.File.XEPG, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	Settings.MappingFirstChannel = 1000

	Data.Streams.Active = nil
	for _, name := range []string{"A", "B", "C"} {
		Data.Streams.Active = append(Data.Streams.Active, map[string]interface{}{
			"_file.m3u.id": "M2", "name": name, "group-title": "G", "url": "http://x/" + name,
		})
	}

	if err := createXEPGDatabase(); err != nil {
		t.Fatal(err)
	}

	var got = map[string]string{}
	for id, c := range Data.XEPG.Channels {
		var ch XEPGChannelStruct
		if err := json.Unmarshal([]byte(mapToJSON(c)), &ch); err != nil {
			t.Fatal(err)
		}
		got[id] = ch.Name + " " + ch.XChannelID
	}

	var want = map[string]string{
		"x-ID.0": "A 1000",
		"x-ID.1": "Old 1001",
		"x-ID.2": "B 1002",
		"x-ID.3": "C 1003",
	}
	if len(got) != len(want) {
		t.Fatalf("channels = %v, want %v", got, want)
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s = %q, want %q (all: %v)", id, got[id], w, got)
		}
	}
}
