package src

import (
	"os"
	"path/filepath"
	"testing"
)

// A playlist that can't be parsed is skipped, not a startup failure. The bug:
// the parse error was left in the function's named err, so when the broken
// playlist was the last one read, buildDatabaseDVR returned it and the server
// exited at startup.
func TestBuildDatabaseDVRSkipsBrokenPlaylist(t *testing.T) {

	var oldFolder, oldFiles, oldStreams = System.Folder.Data, Settings.Files.M3U, Data.Streams
	t.Cleanup(func() {
		System.Folder.Data, Settings.Files.M3U, Data.Streams = oldFolder, oldFiles, oldStreams
	})

	System.Folder.Data = t.TempDir() + string(os.PathSeparator)

	good := "#EXTM3U\n#EXTINF:-1 tvg-id=\"ch1\" group-title=\"News\",One\nhttp://example.invalid/one.ts\n"
	if err := os.WriteFile(filepath.Join(System.Folder.Data, "M3U1.m3u"), []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	// An HTML error page saved under a playlist's name is not a playlist.
	if err := os.WriteFile(filepath.Join(System.Folder.Data, "M3U2.m3u"), []byte("<html>Too many connections</html>"), 0644); err != nil {
		t.Fatal(err)
	}

	// Only the broken playlist: it used to be the last err and fail startup.
	Settings.Files.M3U = map[string]interface{}{
		"M3U2": map[string]interface{}{"name": "Broken"},
	}
	if err := buildDatabaseDVR(); err != nil {
		t.Fatalf("buildDatabaseDVR() with only a broken playlist = %v, want nil", err)
	}

	// A good playlist next to the broken one: the good channels still load.
	Settings.Files.M3U = map[string]interface{}{
		"M3U1": map[string]interface{}{"name": "Good"},
		"M3U2": map[string]interface{}{"name": "Broken"},
	}
	if err := buildDatabaseDVR(); err != nil {
		t.Fatalf("buildDatabaseDVR() with a good and a broken playlist = %v, want nil", err)
	}
	if got := len(Data.Streams.All); got != 1 {
		t.Fatalf("len(Data.Streams.All) = %d, want 1 channel from the good playlist", got)
	}
}
