package src

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
)

// backupSlots are the XEPG channel keys holding the backup stream URLs, in
// priority order.
var backupSlots = []string{"x-backup-channel-1", "x-backup-channel-2", "x-backup-channel-3"}

// autoBackupChannel is one parsed entry of the XEPG channel database, kept
// alongside its key in that database so changes can be written back.
type autoBackupChannel struct {
	id   string
	name string // normalised channel name, used for matching
	// display is the channel's name as configured, for log messages.
	display string
	url     string
	// Original URL and backup slots, so updated channels can be written back
	// into the raw map without disturbing the fields auto-fill doesn't own.
	raw     map[string]interface{}
	source  string // _file.m3u.id of the provider this channel came from
	backups [3]string
}

// channelBrackets matches a bracketed marker in a channel name: "(Backup)",
// "[FHD]", "{4K}".
var channelBrackets = regexp.MustCompile(`[\(\[\{][^\)\]\}]*[\)\]\}]`)

// channelNameNoise are the tokens providers sprinkle into the same channel's
// name without meaning a different channel: video format/quality markers and
// duplicate-copy markers. They are dropped before names are compared, so
// "BBC One HD", "BBC One FHD" and "BBC One (Backup)" all reduce to the same
// channel. Deliberately no ordinary words and no digits - dropping those
// would merge genuinely different channels ("WWE Raw" with "WWE").
var channelNameNoise = map[string]bool{
	"hd":      true,
	"fhd":     true,
	"uhd":     true,
	"qhd":     true,
	"sd":      true,
	"4k":      true,
	"8k":      true,
	"hevc":    true,
	"h264":    true,
	"h265":    true,
	"avc":     true,
	"backup":  true,
	"backup1": true,
	"backup2": true,
}

// normaliseChannelName reduces a channel name to what actually identifies the
// channel: case, surrounding whitespace, punctuation/separators, bracketed
// markers and the format tokens above are all ignored. That is what lets
// auto-fill pair the same channel across providers that spell its name
// differently ("Sky News HD" here, "sky-news" there). It stays conservative
// on purpose: only known-noise tokens and separators are dropped, never
// numbers or ordinary words, so "BBC One" and "BBC Two" can never be paired.
func normaliseChannelName(name string) string {

	var lowered = strings.ToLower(strings.TrimSpace(name))

	// Strip bracketed markers first ("(Backup)"), then split the rest on
	// anything that is neither a letter nor a digit, so "sky-news",
	// "Sky News" and "sky_news" all produce the same tokens.
	lowered = channelBrackets.ReplaceAllString(lowered, " ")

	var fields = strings.FieldsFunc(lowered, func(r rune) bool {
		return unicode.IsLetter(r) == false && unicode.IsDigit(r) == false
	})

	// Nothing name-like left at all ("(Backup)", whitespace, punctuation):
	// there is nothing to compare, so the channel takes part in no group.
	if len(fields) == 0 {
		return ""
	}

	var kept []string
	for _, field := range fields {
		if channelNameNoise[field] == false {
			kept = append(kept, field)
		}
	}

	// A name that is nothing but noise ("HD") must not collapse to "" -
	// that would match every other all-noise channel.
	if len(kept) == 0 {
		return strings.Join(fields, "")
	}

	return strings.Join(kept, "")
}

// autoFillBackups fills each active channel's empty backup slots with the
// URLs of same-named channels from OTHER M3U providers - the typical
// multi-provider setup where several carry the same channel. Channels are
// matched by normaliseChannelName, so provider spelling differences ("HD",
// "(Backup)", separators) don't stop a match. Backups are chosen
// deterministically (sorted channel IDs), at most one per provider, never the
// channel's own URL or one already configured. With the "overwrite" option,
// ALL backup slots of multi-provider channels are rebuilt from scratch.
//
// With the "dryrun" option nothing is written and no rebuild runs: the plan is
// reported instead, so the operator can see what would change before applying
// it. The result is persisted exactly like saveXEpgMapping and one XEPG
// rebuild runs under the same single-flight scan guard.
func autoFillBackups(request RequestStruct) (message string, err error) {

	var overwrite = indexOfString("overwrite", request.Options) != -1
	var dryRun = indexOfString("dryrun", request.Options) != -1

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

			// A rebuild stores channels as structs, while a freshly loaded
			// xepg.json yields maps. Convert once so there is a map to write
			// back into (and keep it, so the rest of the app reads the same
			// entry). Without this, auto-fill silently found no channel at all
			// after any guide rebuild.
			var converted map[string]interface{}
			if jsonErr := json.Unmarshal(channelBytes, &converted); jsonErr != nil || converted == nil {
				continue
			}

			raw = converted
			Data.XEPG.Channels[id] = raw

		}

		var entry = &autoBackupChannel{
			id:      id,
			name:    name,
			display: xepgChannel.XName,
			url:     xepgChannel.URL,
			raw:     raw,
			source:  xepgChannel.FileM3UID,
			backups: [3]string{xepgChannel.XBackupChannel1, xepgChannel.XBackupChannel2, xepgChannel.XBackupChannel3},
		}

		channels = append(channels, entry)
		byName[name] = append(byName[name], entry)

	}

	var updated int
	var updatedSlots int
	var samples []string
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
				updatedSlots += filled

				// A preview reports the plan (with a few examples in the log)
				// and stops here, leaving the database untouched.
				if dryRun == true {
					if len(samples) < 10 {
						samples = append(samples, fmt.Sprintf("%s <- %s", target.display, target.backups[emptySlots[0]]))
					}
					continue
				}

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

	if dryRun == true {

		xepgLock.Unlock()

		for _, sample := range samples {
			showInfo("Backup preview:" + sample)
		}

		message = fmt.Sprintf("Dry run: would fill %d backup slot(s) on %d channel(s) from other providers. Nothing was saved.", updatedSlots, updated)

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

// backupCoverage counts the active channels that have at least one backup
// configured, so the dashboard can show how much of the lineup can actually
// fail over. Handles both stored shapes (a rebuild writes structs, a loaded
// xepg.json yields maps) without a JSON round-trip, so it is cheap enough for
// a dashboard response.
func backupCoverage() (withBackups, active int) {

	xepgLock.Lock()
	defer xepgLock.Unlock()

	for _, value := range Data.XEPG.Channels {

		var hasBackup bool

		switch channel := value.(type) {

		case map[string]interface{}:

			if isActive, _ := channel["x-active"].(bool); isActive == false {
				continue
			}
			active++

			for _, slot := range backupSlots {
				if backup, _ := channel[slot].(string); len(backup) > 0 {
					hasBackup = true
					break
				}
			}

		case XEPGChannelStruct:

			if channel.XActive == false {
				continue
			}
			active++

			hasBackup = len(channel.XBackupChannel1) > 0 || len(channel.XBackupChannel2) > 0 || len(channel.XBackupChannel3) > 0

		default:
			continue

		}

		if hasBackup {
			withBackups++
		}

	}

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
