package src

import (
	"sync"
	"testing"
)

// TestStreamingURLsConcurrentAccess exercises Data.Cache.StreamingURLS the
// way real traffic does: createStreamingURL is called on every /lineup.json
// request and M3U rebuild (writer), while getStreamInfo is called on every
// single stream start in the Stream HTTP handler (reader) - two of the most
// frequent operations in the whole app, running from unrelated goroutines
// with no coordination before streamingURLsLock existed. A Plex client
// changing channels while another client's DVR poll rebuilds the lineup
// would race on this exact map.
func TestStreamingURLsConcurrentAccess(t *testing.T) {

	streamingURLsLock.Lock()
	Data.Cache.StreamingURLS = make(map[string]StreamInfo)
	streamingURLsLock.Unlock()

	var wg sync.WaitGroup

	// Writers: mirror getLineup/buildM3U generating a streaming URL per channel.
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_, err := createStreamingURL("DVR", "playlist", "1", "Channel", "http://example.com/stream")
				if err != nil {
					t.Error(err)
				}
			}
		}(w)
	}

	// Readers: mirror the Stream handler looking up the URL a client asked for.
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_, _ = getStreamInfo("some-url-id")
			}
		}()
	}

	wg.Wait()
}

// TestNotificationConcurrentAccess exercises System.Notification: written by
// addNotification (any showHighlight call, e.g. a live backup restore) and
// read wholesale by setDefaultResponseData on every WS response - which
// happens continuously as long as any browser tab is open.
func TestNotificationConcurrentAccess(t *testing.T) {

	notificationLock.Lock()
	System.Notification = make(map[string]Notification)
	notificationLock.Unlock()

	var wg sync.WaitGroup

	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				var notification Notification
				notification.Type = "info"
				notification.Message = "test"
				_ = addNotification(notification)
			}
		}(w)
	}

	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				notificationLock.Lock()
				snapshot := make(map[string]Notification, len(System.Notification))
				for k, v := range System.Notification {
					snapshot[k] = v
				}
				notificationLock.Unlock()
				_ = snapshot
			}
		}()
	}

	wg.Wait()
}

// TestXMLTVMappingAndStreamsConcurrentAccess exercises the rest of the
// state xepgLock now covers beyond Data.XEPG.Channels: Data.XMLTV.Mapping
// (written by createXEPGMapping) and Data.Streams.Active/Data.Playlist.M3U
// (written by buildDatabaseDVR), both read wholesale by
// setDefaultResponseData on every WS response while the maintenance loop's
// scheduled refresh - or an admin adding a playlist - rebuilds them.
func TestXMLTVMappingAndStreamsConcurrentAccess(t *testing.T) {

	xepgLock.Lock()
	Data.XMLTV.Mapping = make(map[string]interface{})
	Data.Streams.Active = make([]interface{}, 0)
	Data.Streams.All = make([]interface{}, 0)
	xepgLock.Unlock()

	var wg sync.WaitGroup

	// Writers: mirror createXEPGMapping/buildDatabaseDVR replacing the
	// whole map/slices from scratch on each rebuild.
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				newMapping := make(map[string]interface{}, 4)
				for j := 0; j < 4; j++ {
					newMapping[string(rune('a'+j))] = n*1000 + i
				}
				newStreams := make([]interface{}, 0, 4)
				for j := 0; j < 4; j++ {
					newStreams = append(newStreams, j)
				}

				xepgLock.Lock()
				Data.XMLTV.Mapping = newMapping
				Data.Streams.Active = newStreams
				Data.Streams.All = newStreams
				xepgLock.Unlock()
			}
		}(w)
	}

	// Readers: mirror setDefaultResponseData's snapshot before a response
	// escapes to be JSON-marshaled.
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				xepgLock.Lock()
				mappingSnapshot := make(map[string]interface{}, len(Data.XMLTV.Mapping))
				for k, v := range Data.XMLTV.Mapping {
					mappingSnapshot[k] = v
				}
				streamsSnapshot := Data.Streams.Active
				xepgLock.Unlock()
				_ = mappingSnapshot
				_ = streamsSnapshot
			}
		}()
	}

	wg.Wait()
}
