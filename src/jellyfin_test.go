package src

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xteve-reborn/src/internal/imgcache"
)

// Jellyfin has no bespoke integration here: it is served by emulating the
// HDHomeRun tuner protocol and feeding it an XMLTV guide, exactly like Plex
// and Emby. That means "Jellyfin support" is really "these specific endpoint,
// field and format expectations hold", so they're worth pinning down in tests
// that run on every push - the alternative is finding out from a Jellyfin
// user's issue report that a field Jellyfin requires went missing.
//
// The container-based end-to-end check lives in .github/workflows/jellyfin.yml;
// these tests cover the contract itself, deterministically and without Docker.

// TestJellyfinDiscoveryContract checks the fields Jellyfin's HDHomeRun client
// reads when it probes a tuner: /discover.json, /device.xml (UPnP) and
// /lineup_status.json. A missing or empty one of these is what makes Jellyfin
// refuse to add the device, or add it and then show no guide.
func TestJellyfinDiscoveryContract(t *testing.T) {

	setupTunerTest(t)

	// friendlyName, firmwareVersion and modelNumber below come from these (at
	// runtime config.go fills them from the built-in Name/Version constants,
	// which live in package main and so aren't reachable from here). The
	// values are deliberately arbitrary: the assertions only care that they
	// reach /discover.json, so a version bump doesn't need a test change.
	var oldName, oldVersion = System.Name, System.Version
	t.Cleanup(func() { System.Name, System.Version = oldName, oldVersion })
	System.Name = "Test Tuner Name"
	System.Version = "9.9.9"

	// --- /discover.json ---
	var discover Discover
	if err := json.Unmarshal(get(t, Index, "/discover.json").Body.Bytes(), &discover); err != nil {
		t.Fatalf("discover.json is not valid JSON: %v", err)
	}

	var required = map[string]string{
		"BaseURL":         discover.BaseURL,
		"DeviceID":        discover.DeviceID,
		"FirmwareName":    discover.FirmwareName,
		"FirmwareVersion": discover.FirmwareVersion,
		"FriendlyName":    discover.FriendlyName,
		"LineupURL":       discover.LineupURL,
		"Manufacturer":    discover.Manufacturer,
		"ModelNumber":     discover.ModelNumber,
	}
	for field, value := range required {
		if value == "" {
			t.Errorf("discover.json: %s is empty, Jellyfin needs it", field)
		}
	}

	// Jellyfin uses LineupURL to fetch channels, so it has to point at the
	// same host the client reached us on - a hardcoded or stale host here
	// produces a tuner that adds fine and then finds zero channels.
	if !strings.HasSuffix(discover.LineupURL, "/lineup.json") {
		t.Errorf("LineupURL = %q, want it to end in /lineup.json", discover.LineupURL)
	}
	if discover.DeviceID != System.DeviceID {
		t.Errorf("DeviceID = %q, want the app's own device id %q", discover.DeviceID, System.DeviceID)
	}
	if discover.TunerCount != Settings.Tuner {
		t.Errorf("TunerCount = %d, want Settings.Tuner (%d)", discover.TunerCount, Settings.Tuner)
	}

	// --- /device.xml (UPnP) ---
	var device = get(t, Index, "/device.xml").Body.String()
	for _, want := range []string{
		"urn:schemas-upnp-org:device-1-0",
		"<manufacturer>Silicondust</manufacturer>",
		"<modelName>HDTC-2US</modelName>",
		"uuid:" + System.DeviceID,
	} {
		if !strings.Contains(device, want) {
			t.Errorf("device.xml is missing %q", want)
		}
	}

	// It must also be well-formed XML the client can actually parse.
	var root struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal([]byte(device), &root); err != nil {
		t.Errorf("device.xml is not well-formed XML: %v", err)
	} else if root.XMLName.Local != "root" {
		t.Errorf("device.xml root element = <%s>, want <root>", root.XMLName.Local)
	}

	// --- /lineup_status.json ---
	var status LineupStatus
	if err := json.Unmarshal(get(t, Index, "/lineup_status.json").Body.Bytes(), &status); err != nil {
		t.Fatalf("lineup_status.json is not valid JSON: %v", err)
	}
	if status.ScanPossible != 0 {
		t.Errorf("ScanPossible = %d, want 0: there is no tuner here to scan with, "+
			"and claiming otherwise makes clients wait on a scan that never finishes", status.ScanPossible)
	}
	if status.Source == "" || len(status.SourceList) == 0 {
		t.Errorf("Source=%q SourceList=%v, both must be populated", status.Source, status.SourceList)
	}
}

// TestJellyfinLineupShape checks that lineups come back as the JSON array of
// {GuideNumber, GuideName, URL} objects the HDHomeRun protocol specifies -
// what Jellyfin's tuner parser expects field-for-field.
func TestJellyfinLineupShape(t *testing.T) {

	setupTunerTest(t)

	// XEPG is the default mode: the lineup is built from the active channels
	// in the mapping database, so it needs no pms.json (unlike PMS mode,
	// where channel numbers come from Plex/Emby's own guide database).
	Settings.EpgSource = "XEPG"

	setChannels(map[string]interface{}{
		"1": channel("Seed Channel", "1", "http://provider.example/live/1.ts", true, ""),
	})

	var body = get(t, Index, "/lineup.json").Body.Bytes()

	var lineup []LineupStream
	if err := json.Unmarshal(body, &lineup); err != nil {
		t.Fatalf("lineup.json is not a JSON array of streams: %v\n%s", err, body)
	}
	if len(lineup) == 0 {
		t.Fatalf("lineup.json is empty for an active channel; Jellyfin would find no channels")
	}

	if lineup[0].GuideName != "Seed Channel" {
		t.Errorf("GuideName = %q, want %q", lineup[0].GuideName, "Seed Channel")
	}
	// The guide number is what ties a lineup entry to its <channel id> in the
	// XMLTV guide, so an empty one means no guide data in Jellyfin.
	if lineup[0].GuideNumber != "1" {
		t.Errorf("GuideNumber = %q, want the channel's x-channelID %q", lineup[0].GuideNumber, "1")
	}
	if !strings.Contains(lineup[0].URL, "/stream/") {
		t.Errorf("lineup URL = %q, want one of our own /stream/ URLs, not the provider's raw URL",
			lineup[0].URL)
	}
}

// TestXMLTVGuideContract generates a guide through the real code path and
// checks the structure Jellyfin's XMLTV parser consumes. Jellyfin is stricter
// than Plex about the guide, and a guide that parses but has programs attached
// to the wrong channel id shows up as "no guide data" rather than an error, so
// the shape matters as much as well-formedness.
func TestXMLTVGuideContract(t *testing.T) {

	var oldSettings = Settings
	var oldSystem = System
	var oldData = Data
	t.Cleanup(func() {
		Settings = oldSettings
		System = oldSystem
		Data = oldData
	})

	var dir = t.TempDir()
	var dataDir = filepath.Join(dir, "data")
	var imagesDir = filepath.Join(dir, "images")
	for _, d := range []string{dataDir, imagesDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Only needs to be non-empty - the guide records generator/source from
	// these, and the assertions check the structure, not the exact strings.
	System.Name = "Test Tuner Name"
	System.Version = "9.9.9"
	System.Branch = "master"
	System.Folder.Data = dataDir + string(os.PathSeparator)
	System.Folder.ImagesCache = imagesDir + string(os.PathSeparator)
	System.File.XML = filepath.Join(dir, "xteve-reborn.xml")
	System.Compressed.GZxml = filepath.Join(dir, "xteve-reborn.xml.gz")

	// Caching off, so GetURL() passes the logo straight through instead of
	// trying to fetch anything over the network.
	var images, err = imgcache.New(imagesDir, "", false)
	if err != nil {
		t.Fatal(err)
	}
	Data.Cache.Images = images
	Data.Cache.XMLTV = make(map[string]XMLTV)
	Data.XMLTV.Files = []string{"seed.xml"}
	Data.Streams.Active = []interface{}{}

	// A miniature upstream XMLTV, as a provider would deliver it.
	var source = `<?xml version="1.0" encoding="UTF-8"?>
<tv generator-info-name="seed provider">
  <channel id="seed.1"><display-name>Seed Channel</display-name></channel>
  <programme channel="seed.1" start="20260924060000 +0000" stop="20260924070000 +0000">
    <title lang="en">Seed Programme</title>
    <desc lang="en">A programme.</desc>
  </programme>
</tv>`
	if err := os.WriteFile(System.Folder.Data+"seed.xml", []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	// One active channel, mapped onto the source channel id ("x-mapping" must
	// match the upstream <programme channel="...">).
	xepgLock.Lock()
	Data.XEPG.Channels = map[string]interface{}{
		"1": map[string]interface{}{
			"_file.m3u.id": "M1",
			"name":         "Seed Channel",
			"x-name":       "Seed Channel",
			"x-channelID":  "1",
			"x-active":     true,
			"x-mapping":    "seed.1",
			"x-xmltv-file": "seed.xml",
			"tvg-logo":     "",
			"url":          "http://provider.example/live/1.ts",
		},
	}
	xepgLock.Unlock()

	if err := createXMLTVFile(); err != nil {
		t.Fatalf("createXMLTVFile: %v", err)
	}

	var content, readErr = os.ReadFile(System.File.XML)
	if readErr != nil {
		t.Fatalf("guide was not written: %v", readErr)
	}

	// It has to round-trip through the same structs the app writes it with -
	// i.e. be valid XMLTV, not merely valid XML.
	var guide XMLTV
	if err := xml.Unmarshal(content, &guide); err != nil {
		t.Fatalf("generated guide is not valid XMLTV: %v\n%s", err, content)
	}

	if guide.XMLName.Local != "tv" {
		t.Errorf("root element = <%s>, want <tv>", guide.XMLName.Local)
	}
	if guide.Generator == "" {
		t.Error("generator-info-name is empty")
	}

	if len(guide.Channel) != 1 {
		t.Fatalf("guide has %d channels, want 1", len(guide.Channel))
	}
	if guide.Channel[0].ID != "1" {
		t.Errorf("channel id = %q, want the lineup GuideNumber %q", guide.Channel[0].ID, "1")
	}
	if len(guide.Channel[0].DisplayName) == 0 || guide.Channel[0].DisplayName[0].Value != "Seed Channel" {
		t.Errorf("channel display-name = %+v, want the channel name", guide.Channel[0].DisplayName)
	}

	if len(guide.Program) != 1 {
		t.Fatalf("guide has %d programmes, want 1", len(guide.Program))
	}
	var program = guide.Program[0]
	// The programme must be attached to the *lineup* channel id, not the
	// upstream one - a mismatch here is silent, and leaves Jellyfin showing
	// the channel with no guide data at all.
	if program.Channel != "1" {
		t.Errorf("programme channel = %q, want it matched to channel id %q", program.Channel, "1")
	}
	if program.Start != "20260924060000 +0000" || program.Stop != "20260924070000 +0000" {
		t.Errorf("programme start/stop = %q/%q, want the upstream times passed through",
			program.Start, program.Stop)
	}
	if len(program.Title) == 0 || program.Title[0].Value != "Seed Programme" {
		t.Errorf("programme title = %+v, want it carried over from the source", program.Title)
	}
}
