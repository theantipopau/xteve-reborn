package src

import (
	"fmt"
	"math/rand"
	"time"
)

// InitMaintenance : Wartungsprozess initialisieren
func InitMaintenance() (err error) {

	rand.Seed(time.Now().Unix())
	System.TimeForAutoUpdate = fmt.Sprintf("0%d%d", randomTime(0, 2), randomTime(10, 59))

	go maintenance()

	return
}

func maintenance() {

	for {

		var t = time.Now()

		// Scheduled full refresh of every playlist/XMLTV (Settings.Update).
		// Run in its own goroutine so a slow download can't stall this loop
		// past the next scheduled minute.
		for _, schedule := range Settings.Update {
			if schedule == t.Format("1504") {
				go scheduledRefresh(schedule)
			}
		}

		// Between scheduled refreshes, check each source for changes and
		// refresh just the ones that changed.
		if scanInProgress() == false && sourceCheckDue(t) {
			go runSourceCheck()
		}

		// Keep the Xtream account status (max_connections / active_cons) warm, so
		// source selection has it before the first play of a channel rather than
		// only after one. No-op while the cached answer is still fresh.
		providerCapacityMaybeStale()

		// Update xTeVe (Binary)
		if System.TimeForAutoUpdate == t.Format("1504") {
			go BinaryUpdate()
		}

		time.Sleep(60 * time.Second)

	}
}

func scheduledRefresh(schedule string) {

	refreshLock.Lock()
	defer refreshLock.Unlock()

	waitForScan()

	showInfo("Update:" + schedule)

	// Backup erstellen
	err := xTeVeAutoBackup()
	if err != nil {
		ShowError(err, 000)
	}

	// Playlist und XMLTV Dateien aktualisieren
	getProviderData("m3u", "")
	getProviderData("hdhr", "")

	if Settings.EpgSource == "XEPG" {
		getProviderData("xmltv", "")
	}

	// Datenbank für DVR erstellen
	err = buildDatabaseDVR()
	if err != nil {
		ShowError(err, 000)
	}

	if Settings.CacheImages == false && imageCachingInProgress() == false {
		removeChildItems(System.Folder.ImagesCache)
	}

	// XEPG Dateien erstellen
	xepgLock.Lock()
	Data.Cache.XMLTV = make(map[string]XMLTV)
	xepgLock.Unlock()
	buildXEPG(false)

}

func randomTime(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
