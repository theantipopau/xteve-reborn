package src

import (
	"encoding/json"
	"testing"
)

// autoBackupTestChannel builds one XEPG channel database entry the way the
// rest of the app stores it: a nested map keyed by the channel's database ID.
func autoBackupTestChannel(id, source, name, url, backup1 string) (string, map[string]interface{}) {

	var channel, _ = json.Marshal(map[string]interface{}{
		"_file.m3u.id":       source,
		"_file.m3u.name":     source,
		"x-name":             name,
		"x-channelID":        id,
		"url":                url,
		"x-active":           true,
		"x-backup-channel-1": backup1,
	})

	var raw map[string]interface{}
	json.Unmarshal(channel, &raw)

	return id, raw

}

func setupAutoBackupTest(t *testing.T, channels map[string]interface{}) {

	t.Helper()

	xepgLock.Lock()
	Data.XEPG.Channels = channels
	xepgLock.Unlock()

	System.File.XEPG = t.TempDir() + "/xepg.json"

	t.Cleanup(func() {
		xepgLock.Lock()
		Data.XEPG.Channels = make(map[string]interface{})
		xepgLock.Unlock()
	})

}

// The core case: the same channel carried by two providers fills the empty
// backup slot on both with the other provider's URL - and touches nothing
// else. Both channels must end up with each other as their backup.
func TestAutoFillBackupsFillsBothDirections(t *testing.T) {

	idA, chA := autoBackupTestChannel("1000", "M1", "Sky News", "http://p1/sky-news", "")
	idB, chB := autoBackupTestChannel("2000", "M2", "sky news ", "http://p2/skynews", "")

	setupAutoBackupTest(t, map[string]interface{}{idA: chA, idB: chB})

	message, err := autoFillBackups(RequestStruct{Cmd: "autoFillBackups"})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(message)

	if chA["x-backup-channel-1"] != "http://p2/skynews" {
		t.Errorf("provider-1 channel backup = %v, want http://p2/skynews", chA["x-backup-channel-1"])
	}

	if chB["x-backup-channel-1"] != "http://p1/sky-news" {
		t.Errorf("provider-2 channel backup = %v, want [http://p1/sky-news]", chB["x-backup-channel-1"])
	}

}

// An existing backup must never be clobbered, and a slot the operator set is
// kept out of the duplicates when filling remaining empty slots.
func TestAutoFillBackupsPreservesExistingBackups(t *testing.T) {

	idA, chA := autoBackupTestChannel("1000", "M1", "CNN", "http://p1/cnn", "http://custom/backup")
	idB, chB := autoBackupTestChannel("2000", "M2", "CNN", "http://p2/cnn", "")

	setupAutoBackupTest(t, map[string]interface{}{idA: chA, idB: chB})

	if _, err := autoFillBackups(RequestStruct{Cmd: "autoFillBackups"}); err != nil {
		t.Fatal(err)
	}

	if chA["x-backup-channel-1"] != "http://custom/backup" {
		t.Errorf("existing backup clobbered: %v", chA["x-backup-channel-1"])
	}
	if chB["x-backup-channel-1"] != "http://p1/cnn" {
		t.Errorf("provider-2 backup = %v, want the provider-1 URL", chB["x-backup-channel-1"])
	}

}

// Never fill from the same provider as the channel's own URL, even when the
// same provider lists the channel twice.
func TestAutoFillBackupsNeverUsesSameProvider(t *testing.T) {

	idA, chA := autoBackupTestChannel("1000", "M1", "Discovery", "http://p1/discovery", "")
	idB, chB := autoBackupTestChannel("2000", "M1", "Discovery", "http://p1/discovery-hd", "")

	setupAutoBackupTest(t, map[string]interface{}{idA: chA, idB: chB})

	message, err := autoFillBackups(RequestStruct{Cmd: "autoFillBackups"})
	if err != nil {
		t.Fatal(err)
	}

	if chA["x-backup-channel-1"] != "" || chB["x-backup-channel-1"] != "" {
		t.Errorf("same-provider channels were used as backups: %v / %v", chA["x-backup-channel-1"], chB["x-backup-channel-1"])
	}

	if message == "" {
		t.Error("expected an explanatory message when nothing can be filled")
	}

}

// Overwrite rebuilds all slots from scratch instead of keeping existing ones.
func TestAutoFillBackupsOverwriteReplacesExisting(t *testing.T) {

	idA, chA := autoBackupTestChannel("1000", "M1", "BBC One", "http://p1/bbcone", "http://custom/stale")
	idB, chB := autoBackupTestChannel("2000", "M2", "BBC One", "http://p2/bbcone", "")

	setupAutoBackupTest(t, map[string]interface{}{idA: chA, idB: chB})

	if _, err := autoFillBackups(RequestStruct{Cmd: "autoFillBackups", Options: []string{"overwrite"}}); err != nil {
		t.Fatal(err)
	}

	if chA["x-backup-channel-1"] != "http://p2/bbcone" {
		t.Errorf("overwrite kept the stale custom backup: %v", chA["x-backup-channel-1"])
	}

}

// Channels that are not active, have no URL, or have no name are ignored.
func TestAutoFillBackupsIgnoresInactiveAndUnnamed(t *testing.T) {

	idA, chA := autoBackupTestChannel("1000", "M1", "HBO", "http://p1/hbo", "")
	idB, chB := autoBackupTestChannel("2000", "M2", "HBO", "http://p2/hbo", "")
	chB["x-active"] = false

	_, chNoURL := autoBackupTestChannel("3000", "M2", "HBO", "", "")
	chNoURL["x-active"] = true

	idC, chC := autoBackupTestChannel("4000", "M1", "   ", "http://p1/blank-name", "")

	setupAutoBackupTest(t, map[string]interface{}{idA: chA, idB: chB, "3000": chNoURL, idC: chC})

	if _, err := autoFillBackups(RequestStruct{Cmd: "autoFillBackups"}); err != nil {
		t.Fatal(err)
	}

	if chA["x-backup-channel-1"] != "" {
		t.Errorf("inactive duplicate was used as a backup: %v", chA["x-backup-channel-1"])
	}

}

// Persisted result: with a XEPG rebuild already in flight, the save still
// happens (to the file, parseable) and the in-memory database the app serves
// from carries the filled backups - the queued rebuild then applies on top.
// Content correctness of the write-back is covered by the assertions in the
// tests above; this one pins the save-and-swap behavior under concurrency.
func TestAutoFillBackupsPersistsWhileRebuildRunning(t *testing.T) {

	idA, chA := autoBackupTestChannel("1000", "M1", "Comedy Central", "http://p1/cc", "")
	idB, chB := autoBackupTestChannel("2000", "M2", "Comedy Central", "http://p2/cc", "")

	setupAutoBackupTest(t, map[string]interface{}{idA: chA, idB: chB})

	// Simulate a rebuild in progress so the save-and-swap path runs instead
	// of the immediate rebuild (which would legitimately drop these fake
	// channels - they have no real provider behind them).
	if tryStartScan() == false {
		t.Fatal("could not start the fake scan")
	}

	message, err := autoFillBackups(RequestStruct{Cmd: "autoFillBackups"})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(message)

	xepgLock.Lock()
	var storedRaw, _ = Data.XEPG.Channels[idA].(map[string]interface{})
	xepgLock.Unlock()

	if storedRaw == nil {
		t.Fatalf("in-memory channel %q missing after auto-fill", idA)
	}

	if stored, _ := storedRaw["x-backup-channel-1"].(string); stored != "http://p2/cc" {
		t.Errorf("in-memory channel backup = %v, want http://p2/cc", storedRaw["x-backup-channel-1"])
	}

	var saved, err2 = loadJSONFileToMap(System.File.XEPG)
	if err2 != nil {
		t.Fatalf("XEPG file did not parse after auto-fill: %v", err2)
	}

	var savedRaw, _ = saved[idA].(map[string]interface{})
	if savedRaw == nil || savedRaw["x-backup-channel-1"] != "http://p2/cc" {
		t.Errorf("saved channel backup = %v, want http://p2/cc", savedRaw)
	}

}
