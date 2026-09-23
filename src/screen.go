package src

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// logLock guards WebScreenLog. Log lines are written from every goroutine
// in the app (each stream start logs from its own HTTP handler) and read by
// every WS response, so all access goes through appendWebLog/resetWebLog/
// snapshotWebLog. The previous code declared a fresh mutex inside each log
// function, which locks nothing since no two calls ever share it.
var logLock sync.Mutex

// alignLogMessage pads "Key:value" so values line up in a column, the way
// the console and Log page have always displayed them. Messages without a
// colon are returned unchanged.
func alignLogMessage(str string) string {

	var msg = strings.SplitN(str, ":", 2)
	if len(msg) != 2 {
		return str
	}

	var padding = 23 - len(msg[0])
	if padding < 0 {
		padding = 0
	}

	return msg[0] + ":" + strings.Repeat(" ", padding) + msg[1]
}

// appendWebLog adds one line to the in-memory log shown on the Log page and
// trims it back to Settings.LogEntriesRAM.
func appendWebLog(logMsg string) {

	logLock.Lock()
	defer logLock.Unlock()

	WebScreenLog.Log = append(WebScreenLog.Log, time.Now().Format("2006-01-02 15:04:05")+" "+logMsg)
	trimWebLog()
}

// trimWebLog keeps only the newest Settings.LogEntriesRAM lines and
// recounts errors/warnings from what's left. Caller must hold logLock.
func trimWebLog() {

	var limit = Settings.LogEntriesRAM

	if limit > 0 && len(WebScreenLog.Log) > limit {
		// Copied into a fresh slice so the dropped lines' backing array can
		// actually be freed rather than pinned by a subslice.
		WebScreenLog.Log = append([]string(nil), WebScreenLog.Log[len(WebScreenLog.Log)-limit:]...)
	}

	WebScreenLog.Warnings = 0
	WebScreenLog.Errors = 0

	for _, line := range WebScreenLog.Log {

		if strings.Contains(line, "WARNING") {
			WebScreenLog.Warnings++
		}

		if strings.Contains(line, "ERROR") {
			WebScreenLog.Errors++
		}

	}

}

// resetWebLog clears the in-memory log (Log page "Reset" button, restore).
func resetWebLog() {

	logLock.Lock()
	defer logLock.Unlock()

	WebScreenLog.Log = make([]string, 0)
	WebScreenLog.Errors = 0
	WebScreenLog.Warnings = 0
}

// snapshotWebLog returns a copy of the log for a WS response, which is
// JSON-marshaled after the lock is released.
func snapshotWebLog() (snapshot WebScreenLogStruct) {

	logLock.Lock()
	defer logLock.Unlock()

	snapshot.Errors = WebScreenLog.Errors
	snapshot.Warnings = WebScreenLog.Warnings
	snapshot.Log = append([]string(nil), WebScreenLog.Log...)

	return
}

func showInfo(str string) {

	if System.Flag.Info == true {
		return
	}

	if strings.Contains(str, ":") == false {
		return
	}

	var logMsg = fmt.Sprintf("[%s] %s", System.Name, alignLogMessage(str))

	printLogOnScreen(logMsg, "info")
	appendWebLog(logMsg)
}

func showDebug(str string, level int) {

	if System.Flag.Debug < level {
		return
	}

	if strings.Contains(str, ":") == false {
		return
	}

	var logMsg = fmt.Sprintf("[DEBUG] %s", alignLogMessage(str))

	printLogOnScreen(logMsg, "debug")
	appendWebLog(logMsg)
}

func showHighlight(str string) {

	var logMsg = fmt.Sprintf("[%s] %s", System.Name, alignLogMessage(str))

	printLogOnScreen(logMsg, "highlight")
	appendWebLog(logMsg)

	var message = str
	if msg := strings.SplitN(str, ":", 2); len(msg) == 2 {
		message = msg[1]
	}

	var notification Notification
	notification.Type = "info"
	notification.Message = message

	addNotification(notification)
}

func showWarning(errCode int) {

	var logMsg = fmt.Sprintf("[%s] [WARNING] %s", System.Name, getErrMsg(errCode))

	printLogOnScreen(logMsg, "warning")
	appendWebLog(logMsg)
}

// ShowError : Zeigt die Fehlermeldungen in der Konsole
func ShowError(err error, errCode int) {

	var logMsg = fmt.Sprintf("[%s] [ERROR] %s (%s) - EC: %d", System.Name, err, getErrMsg(errCode), errCode)

	printLogOnScreen(logMsg, "error")
	appendWebLog(logMsg)
}

func printLogOnScreen(logMsg string, logType string) {

	var color string

	switch logType {

	case "info":
		color = "\033[0m"

	case "debug":
		color = "\033[35m"

	case "highlight":
		color = "\033[32m"

	case "warning":
		color = "\033[33m"

	case "error":
		color = "\033[31m"

	}

	switch runtime.GOOS {

	case "windows":
		log.Println(logMsg)

	default:
		fmt.Print(color)
		log.Println(logMsg)
		fmt.Print("\033[0m")

	}

}

// Fehlercodes
func getErrMsg(errCode int) (errMsg string) {

	switch errCode {

	case 0:
		return

	// Errors
	case 1001:
		errMsg = fmt.Sprintf("Web server could not be started.")
	case 1002:
		errMsg = fmt.Sprintf("No local IP address found.")
	case 1003:
		errMsg = fmt.Sprintf("Invalid xml")
	case 1004:
		errMsg = fmt.Sprintf("File not found")
	case 1005:
		errMsg = fmt.Sprintf("Invalid M3U file, an extended M3U file is required.")
	case 1006:
		errMsg = fmt.Sprintf("No playlist!")
	case 1007:
		errMsg = fmt.Sprintf("XEPG requires an XMLTV file.")
	case 1010:
		errMsg = fmt.Sprintf("Invalid file compression")
	case 1011:
		errMsg = fmt.Sprintf("Data is corrupt or unavailable, %s now uses an older version of this file", System.Name)
	case 1012:
		errMsg = fmt.Sprintf("Invalid formatting of the time")
	case 1013:
		errMsg = fmt.Sprintf("Invalid settings file (settings.json), file must be at least version %s", System.Compatibility)
	case 1014:
		errMsg = fmt.Sprintf("Invalid filter rule")

	case 1020:
		errMsg = fmt.Sprintf("Data could not be saved, invalid keyword")

	// Datenbank Update
	case 1030:
		errMsg = fmt.Sprintf("Invalid settings file (%s)", System.File.Settings)
	case 1031:
		errMsg = fmt.Sprintf("Database error. The database version of your settings is not compatible with this version.")

	// M3U Parser
	case 1050:
		errMsg = fmt.Sprintf("Invalid duration specification in the M3U8 playlist.")

	case 1060:
		errMsg = fmt.Sprintf("Invalid characters found in the tvg parameters, streams with invalid parameters were skipped.")

	// Dateisystem
	case 1070:
		errMsg = fmt.Sprintf("Folder could not be created.")
	case 1071:
		errMsg = fmt.Sprintf("File could not be created")
	case 1072:
		errMsg = fmt.Sprintf("File not found")

	// Backup
	case 1090:
		errMsg = fmt.Sprintf("Automatic backup failed")

	// Websockets
	case 1100:
		errMsg = fmt.Sprintf("WebUI build error")
	case 1101:
		errMsg = fmt.Sprintf("WebUI request error")
	case 1102:
		errMsg = fmt.Sprintf("WebUI response error")

	// PMS Guide Numbers
	case 1200:
		errMsg = fmt.Sprintf("Could not create file")

	// Stream URL Fehler
	case 1201:
		errMsg = fmt.Sprintf("Plex stream error")
	case 1202:
		errMsg = fmt.Sprintf("Steaming URL could not be found in any playlist")
	case 1203:
		errMsg = fmt.Sprintf("Steaming URL could not be found in any playlist")
	case 1204:
		errMsg = fmt.Sprintf("Streaming was stopped by third party transcoder (FFmpeg / VLC)")

	// Warnings
	case 2000:
		errMsg = fmt.Sprintf("Plex can not handle more than %d streams. If you do not use Plex, you can ignore this warning.", System.PlexChannelLimit)
	case 2001:
		errMsg = fmt.Sprintf("%s has loaded more than %d streams. Use the filter to reduce the number of streams.", System.Name, System.UnfilteredChannelLimit)
	case 2002:
		errMsg = fmt.Sprintf("PMS can not play m3u8 streams")
	case 2003:
		errMsg = fmt.Sprintf("PMS can not play streams over RTSP.")
	case 2004:
		errMsg = fmt.Sprintf("Buffer is disabled for this stream.")
	case 2005:
		errMsg = fmt.Sprintf("There are no channels mapped, use the mapping menu to assign EPG data to the channels.")
	case 2010:
		errMsg = fmt.Sprintf("No valid streaming URL")
	case 2020:
		errMsg = fmt.Sprintf("FFmpeg binary was not found. Check the FFmpeg binary path in Settings.")
	case 2021:
		errMsg = "VLC binary was not found. Check the VLC binary path in Settings."

	case 2099:
		errMsg = "Updates have been disabled by the developer"

	// Tuner
	case 2105:
		errMsg = fmt.Sprintf("The number of tuners has changed. Remove %s from Plex / Emby / Jellyfin Live TV and add it again for the change to take effect.", System.Name)
	case 2106:
		errMsg = "This function is only available with XEPG as EPG source"

	case 2110:
		errMsg = "Running as root is not recommended."

	case 2300:
		errMsg = fmt.Sprintf("No channel logo found in the XMLTV or M3U file.")
	case 2301:
		errMsg = fmt.Sprintf("XMLTV file no longer available, channel has been deactivated.")
	case 2302:
		errMsg = fmt.Sprintf("Channel ID in the XMLTV file has changed. Channel has been deactivated.")

	// Benutzerauthentifizierung
	case 3000:
		errMsg = fmt.Sprintf("Database for user authentication could not be initialized.")
	case 3001:
		errMsg = fmt.Sprintf("The user has no authorization to load the channels.")

	// Buffer
	case 4000:
		errMsg = fmt.Sprintf("Connection to streaming source was interrupted.")
	case 4001:
		errMsg = fmt.Sprintf("Too many errors connecting to the provider. Streaming is canceled.")
	case 4002:
		errMsg = fmt.Sprintf("New URL for the redirect to the streaming server is missing")
	case 4003:
		errMsg = fmt.Sprintf("Server sends an incompatible content-type")
	case 4004:
		errMsg = fmt.Sprintf("This error message comes from the provider")
	case 4005:
		errMsg = fmt.Sprintf("Temporary buffer files could not be deleted")
	case 4006:
		errMsg = fmt.Sprintf("Server connection timeout")
	case 4007:
		errMsg = fmt.Sprintf("Primary channel unreachable, switched to a backup channel")

	// Buffer (M3U8)
	case 4050:
		errMsg = fmt.Sprintf("Invalid M3U8 file")
	case 4051:
		errMsg = fmt.Sprintf("#EXTM3U header is missing")

	// Caching
	case 4100:
		errMsg = fmt.Sprintf("Unknown content type for downloaded image")
	case 4101:
		errMsg = fmt.Sprintf("Invalid URL, original URL is used for this image")

	// API
	case 5000:
		errMsg = fmt.Sprintf("Invalid API command")

	// Update Server
	case 6001:
		errMsg = fmt.Sprintf("Ivalid key")
	case 6002:
		errMsg = fmt.Sprintf("Update failed")
	case 6003:
		errMsg = fmt.Sprintf("Update server not available")
	case 6004:
		errMsg = "Update available"

	default:
		errMsg = fmt.Sprintf("Unknown error / warning (%d)", errCode)
	}

	return errMsg
}

func addNotification(notification Notification) (err error) {

	notificationLock.Lock()
	defer notificationLock.Unlock()

	var i int
	var t = time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
	notification.Time = strconv.FormatInt(t, 10)
	notification.New = true

	if len(notification.Headline) == 0 {
		notification.Headline = strings.ToUpper(notification.Type)
	}

	if len(System.Notification) == 0 {
		System.Notification = make(map[string]Notification)
	}

	System.Notification[notification.Time] = notification

	for key := range System.Notification {

		if i < len(System.Notification)-10 {
			delete(System.Notification, key)
		}

		i++

	}

	return
}
