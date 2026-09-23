package src

import (
	"errors"
	"fmt"
	"os"

	up2date "xteve-reborn/src/internal/up2date/client"

	"reflect"
)

// checkForRelease queries GitHub for the newest release and reports
// whether it's actually newer than the running binary. Shared by the
// automatic startup/scheduled check and the manual "Check for Updates" UI
// action so both apply the exact same logic. Caches the verdict on
// System.UpdateAvailable/UpdateVersion so the dashboard can display it
// without making its own GitHub API call on every WS response.
func checkForRelease() (release up2date.Release, isUpdate bool, err error) {

	if len(System.ReleaseTag) == 0 {
		err = errors.New("not a release build (no version to compare against)")
		return
	}

	release, err = up2date.GetLatestRelease(System.GitHub.User, System.GitHub.Repo, System.AppName)
	if err != nil {
		return
	}

	if release.Found == false {
		err = fmt.Errorf("no release asset found for %s/%s", System.OS, System.ARCH)
		return
	}

	isUpdate = up2date.IsNewer(System.ReleaseTag, release.Tag)

	updateStateLock.Lock()
	System.UpdateAvailable = isUpdate
	System.UpdateVersion = release.Tag
	updateStateLock.Unlock()

	return
}

// BinaryUpdate checks GitHub's Releases API for a newer build of this fork
// and, if Settings.XteveAutoUpdate is on, installs it. Always safe to call:
// it no-ops if update checking is disabled, or if this binary wasn't built
// through the official release process (System.ReleaseTag empty - a plain
// `go build .` from source has no reliable version to compare against).
func BinaryUpdate() (err error) {

	up2date.CleanupOldBinary()

	if System.GitHub.Update == false {
		return
	}

	release, isUpdate, err := checkForRelease()
	if err != nil {
		showDebug("Update:"+err.Error(), 1)
		return nil
	}

	if isUpdate == false {
		return nil
	}

	showHighlight(fmt.Sprintf("Update available:%s (you're on %s)", release.Tag, System.ReleaseTag))

	if Settings.XteveAutoUpdate == true && runningInDocker() == false {
		err = installRelease(release)
		if err != nil {
			ShowError(err, 6002)
		}
	}

	return nil
}

// runningInDocker is set via the XTEVE_REBORN_DOCKER env var baked into the
// official image. Containers are updated by pulling a new image, not by
// rewriting the binary inside a running container (which a non-root
// container user can't do anyway, and which a restart would undo).
func runningInDocker() bool {
	return os.Getenv("XTEVE_REBORN_DOCKER") == "1"
}

func installRelease(release up2date.Release) (err error) {

	if runningInDocker() {
		return fmt.Errorf("running in Docker: pull ghcr.io/theantipopau/xteve-reborn:%s and recreate the container to update", release.Tag)
	}

	showInfo(fmt.Sprintf("Update:Installing %s...", release.Tag))
	return up2date.DoUpdate(release.ZipURL, release.ChecksumsURL, release.Filename)
}

// CheckForUpdate re-checks GitHub for a newer release, for the manual
// "Check for Updates" WS command - always a fresh check rather than
// trusting a possibly-stale result from an earlier automatic one.
func CheckForUpdate() (available bool, latestVersion string, err error) {

	release, isUpdate, err := checkForRelease()
	if err != nil {
		return false, "", err
	}

	return isUpdate, release.Tag, nil
}

// InstallAvailableUpdate re-checks for and, if one is still available,
// installs a newer release - for the manual "Install Now" WS command.
func InstallAvailableUpdate() (err error) {

	release, isUpdate, err := checkForRelease()
	if err != nil {
		return err
	}

	if isUpdate == false {
		return errors.New("no update available")
	}

	return installRelease(release)
}

func conditionalUpdateChanges() (err error) {

checkVersion:
	settingsMap, err := loadJSONFileToMap(System.File.Settings)
	if err != nil || len(settingsMap) == 0 {
		return
	}

	if settingsVersion, ok := settingsMap["version"].(string); ok {

		if settingsVersion > System.DBVersion {
			showInfo("Settings DB Version:" + settingsVersion)
			showInfo("System DB Version:" + System.DBVersion)
			err = errors.New(getErrMsg(1031))
			return
		}

		// Letzte Kompatible Version (1.4.4)
		if settingsVersion < System.Compatibility {
			err = errors.New(getErrMsg(1013))
			return
		}

		switch settingsVersion {

		case "1.4.4":
			// UUID Wert in xepg.json setzen
			err = setValueForUUID()
			if err != nil {
				return
			}

			// Neuer Filter (WebUI). Alte Filtereinstellungen werden konvertiert
			if oldFilter, ok := settingsMap["filter"].([]interface{}); ok {
				var newFilterMap = convertToNewFilter(oldFilter)
				settingsMap["filter"] = newFilterMap

				settingsMap["version"] = "2.0.0"

				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}

				goto checkVersion

			} else {
				err = errors.New(getErrMsg(1030))
				return
			}

		case "2.0.0":

			if oldBuffer, ok := settingsMap["buffer"].(bool); ok {

				var newBuffer string
				switch oldBuffer {
				case true:
					newBuffer = "xteve"
				case false:
					newBuffer = "-"
				}

				settingsMap["buffer"] = newBuffer

				settingsMap["version"] = "2.1.0"

				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}

				goto checkVersion

			} else {
				err = errors.New(getErrMsg(1030))
				return
			}

		case "2.1.0":
			// Falls es in einem späteren Update Änderungen an der Datenbank gibt, geht es hier weiter

			break
		}

	} else {
		// settings.json ist zu alt (älter als Version 1.4.4)
		err = errors.New(getErrMsg(1013))
	}

	return
}

func convertToNewFilter(oldFilter []interface{}) (newFilterMap map[int]interface{}) {

	newFilterMap = make(map[int]interface{})

	switch reflect.TypeOf(oldFilter).Kind() {

	case reflect.Slice:
		s := reflect.ValueOf(oldFilter)

		for i := 0; i < s.Len(); i++ {

			var newFilter FilterStruct
			newFilter.Active = true
			newFilter.Name = fmt.Sprintf("Custom filter %d", i+1)
			newFilter.Filter = s.Index(i).Interface().(string)
			newFilter.Type = "custom-filter"
			newFilter.CaseSensitive = false

			newFilterMap[i] = newFilter

		}

	}

	return
}

func setValueForUUID() (err error) {

	xepg, err := loadJSONFileToMap(System.File.XEPG)

	for _, c := range xepg {

		var xepgChannel = c.(map[string]interface{})

		if uuidKey, ok := xepgChannel["_uuid.key"].(string); ok {

			if value, ok := xepgChannel[uuidKey].(string); ok {

				if len(value) > 0 {
					xepgChannel["_uuid.value"] = value
				}

			}

		}

	}

	err = saveMapToJSONFile(System.File.XEPG, xepg)

	return
}
