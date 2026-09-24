package src

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// backupSlots are the XEPG channel keys holding the backup stream URLs, in
// priority order.
var backupSlots = []string{"x-backup-channel-1", "x-backup-channel-2", "x-backup-channel-3"}

// autoBackupChannel is one parsed entry of the XEPG channel database, kept
// alongside its key in that database so changes can be written back.
type autoBackupChannel struct {
	id   string
	name string // normalised (lower-cased, trimmed) channel name
	url  string
	// Original URL and backup slots, so updated channels can be written back
	// into the raw map without disturbing the fields auto-fill doesn't own.
	raw     map[string]interface{}
	source  string // _file.m3u.id of the provider this channel came from
	backups [3]string
}

// normaliseChannelName makes channel-name comparisons tolerant of provider
// formatting differences: case and surrounding whitespace are ignored. Only
// that - no punctuation stripping - so "BBC One" and "BBC One HD" stay
// distinct on purpose.
func normaliseChannelName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// autoFillBackups fills each active channel's empty backup slots with the
// URLs of same-named channels from OTHER M3U providers - the typical
// multi-provider setup where several carry the same channel. Backups are
// chosen deterministically (sorted channel IDs), at most one per provider,
// never the channel's own URL or one already configured. With overwrite set,
// ALL backup slots of multi-provider channels are rebuilt from scratch.
//
// The result is persisted exactly like saveXEpgMapping and one XEPG rebuild
// runs under the same single-flight scan guard.
func autoFillBackups(request RequestStruct) (message string, err error) {

	var overwrite = len(request.Options) > 0 && request.Options[0] == "overwrite"

	xepgLock.Lock()

	// Parse the raw channel database once; remember the keys so updates can
	// be written back to the same map (and nested maps) the rest of the app
	// reads.
	var channels []*autoBackupChannel
	var byName = make(map[string][]*autoBackupChannel)

	var ids []string
	for id := range Data.XEPG.Channels {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {

		var channelBytes = []byte(mapToJSON(Data.XEPG.Channels[id]))
		var xepgChannel XEPGChannelStruct
		if jsonErr := json.Unmarshal(channelBytes, &xepgChannel); jsonErr != nil {
			continue
		}

		if xepgChannel.XActive == false || len(xepgChannel.URL) == 0 {
			continue
		}

		var name = normaliseChannelName(xepgChannel.XName)
		if len(name) == 0 {
			continue
		}

		var raw, ok = Data.XEPG.Channels[id].(map[string]interface{})
		if ok == false {
			continue
		}

		var entry = &autoBackupChannel{
			id:      id,
			name:    name,
			url:     xepgChannel.URL,
			raw:     raw,
			source:  xepgChannel.FileM3UID,
			backups: [3]string{xepgChannel.XBackupChannel1, xepgChannel.XBackupChannel2, xepgChannel.XBackupChannel3},
		}

		channels = append(channels, entry)
		byName[name] = append(byName[name], entry)

	}

	var updated int
	var filledGroups, singleProviderGroups, alreadyFull int

	for _, group := range byName {

		// A backup only makes sense when the same channel is carried by more
		// than one provider.
		var sources []string
		for _, entry := range group {
			if indexOfString(entry.source, sources) == -1 {
				sources = append(sources, entry.source)
			}
		}
		if len(sources) < 2 {
			singleProviderGroups++
			continue
		}

		filledGroups++

		for _, target := range group {

			// Collect the empty slot indexes and the URLs already configured,
			// so filling can never clobber an existing backup.
			var emptySlots []int
			var used []string

			for i, backup := range target.backups {
				if len(backup) == 0 {
					emptySlots = append(emptySlots, i)
				} else {
					used = append(used, strings.ToLower(strings.TrimSpace(backup)))
				}
			}

			if len(emptySlots) == 0 && overwrite == false {
				alreadyFull++
				continue
			}

			if overwrite == true {
				target.backups = [3]string{}
				emptySlots = []int{0, 1, 2}
				used = nil
			}

			var filled int

			for _, candidate := range group {

				if filled >= len(emptySlots) {
					break
				}

				if candidate.id == target.id || candidate.source == target.source {
					continue
				}

				if len(candidate.url) == 0 || candidate.url == target.url {
					continue
				}

				var candidateKey = strings.ToLower(strings.TrimSpace(candidate.url))
				if indexOfString(candidateKey, used) != -1 {
					continue
				}

				target.backups[emptySlots[filled]] = candidate.url
				used = append(used, candidateKey)
				filled++

			}

			if filled > 0 {

				updated++

				// Write back into the raw channel map (maps are reference
				// types, so target.raw IS the map stored in the database).
				for i, slot := range backupSlots {
					target.raw[slot] = target.backups[i]
				}

			}

		}

	}

	if updated == 0 {

		xepgLock.Unlock()

		switch {
		case filledGroups == 0 && singleProviderGroups > 0:
			message = fmt.Sprintf("No backups filled: %d channel name(s) are shared, but only one provider carries each - there is nothing to fail over to.", singleProviderGroups)
		case filledGroups == 0:
			message = "No two providers share a channel name, so there was nothing to auto-fill."
		case alreadyFull > 0:
			message = fmt.Sprintf("No empty backup slots: %d multi-provider channel(s) already have backups. Tick 'overwrite' to rebuild them.", alreadyFull)
		default:
			message = "No backup slots needed filling."
		}

		return
	}

	// Persist a deep copy of the channel database, exactly like
	// saveXEpgMapping does, then swap the parsed map in.
	var toSave = make(map[string]interface{})
	for id, value := range Data.XEPG.Channels {
		toSave[id] = value
	}

	xepgLock.Unlock()

	if err = saveMapToJSONFile(System.File.XEPG, toSave); err != nil {
		return
	}

	xepgLock.Lock()
	Data.XEPG.Channels = toSave
	xepgLock.Unlock()

	triggerXEPGRebuild()

	message = fmt.Sprintf("Backup slots filled on %d channel(s) from other providers.", updated)

	return
}

// triggerXEPGRebuild runs one XEPG rebuild now, or - when a rebuild is
// already running - queues exactly one follow-up for when it finishes.
// Extracted from saveXEpgMapping so both paths stay identical.
func triggerXEPGRebuild() {

	if tryStartScan() {

		cleanupXEPG()
		endScan()
		buildXEPG(true)

		return

	}

	// Saved while a rebuild is running: queue exactly one follow-up rebuild
	// for when it finishes, however many saves happen meanwhile.
	if atomic.CompareAndSwapInt32(&rebuildQueued, 0, 1) == false {
		return
	}

	go func() {

		defer atomic.StoreInt32(&rebuildQueued, 0)

		for tryStartScan() == false {
			time.Sleep(time.Second)
		}

		cleanupXEPG()
		endScan()
		buildXEPG(false)

	}()

}
