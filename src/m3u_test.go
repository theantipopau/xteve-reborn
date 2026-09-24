package src

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xteve-reborn/src/internal/imgcache"
)

// buildM3U must put a channel's provider directives back between its #EXTINF
// and its stream URL - that is where M3U players expect them, and it is the
// only place they can be seen at all (Plex, Emby and Jellyfin ignore
// #KODIPROP and friends entirely).
func TestBuildM3UEmitStreamDirectives(t *testing.T) {

	setupTunerTest(t)

	var oldAppName, oldDomain = System.AppName, System.Domain
	var oldImages = Data.Cache.Images
	t.Cleanup(func() {
		System.AppName = oldAppName
		System.Domain = oldDomain
		Data.Cache.Images = oldImages
	})

	var dir = t.TempDir()
	System.File.M3U = filepath.Join(dir, "xteve.m3u")
	System.AppName = "xteve"
	System.Domain = "tuner.local:34400"

	// Caching off, so logo URLs pass straight through.
	var images, err = imgcache.New(dir, "", false)
	if err != nil {
		t.Fatal(err)
	}
	Data.Cache.Images = images

	var directives = strings.Join([]string{
		"#KODIPROP:inputstream.adaptive.license_type=clearkey",
		"#EXTVLCOPT:http-user-agent=ExamplePlayer/1.0",
	}, "\n")

	setChannels(map[string]interface{}{
		"x-ID.0": map[string]interface{}{
			"_file.m3u.id":  "M1",
			"x-name":        "DRM One",
			"x-channelID":   "1000",
			"x-epg":         "x-ID.0",
			"x-group-title": "Sports",
			"url":           "http://provider.example/drm1",
			"x-active":      true,
			"x-directives":  directives,
		},
		"x-ID.1": map[string]interface{}{
			"_file.m3u.id":  "M1",
			"x-name":        "Plain One",
			"x-channelID":   "1001",
			"x-epg":         "x-ID.1",
			"x-group-title": "News",
			"url":           "http://provider.example/plain1",
			"x-active":      true,
		},
	})

	if _, err := buildM3U(nil); err != nil {
		t.Fatal(err)
	}

	var body, readErr = os.ReadFile(System.File.M3U)
	if readErr != nil {
		t.Fatal(readErr)
	}

	var lines = strings.Split(string(body), "\n")

	var extinf = -1
	for i, line := range lines {
		if strings.HasPrefix(line, "#EXTINF") && strings.HasSuffix(line, ",DRM One") {
			extinf = i
			break
		}
	}

	if extinf == -1 {
		t.Fatalf("no #EXTINF for the directive channel in:\n%s", string(body))
	}

	if strings.HasPrefix(lines[extinf+1], "#KODIPROP:") == false {
		t.Errorf("first directive is not directly after the #EXTINF: %q", lines[extinf+1])
	}

	if strings.HasPrefix(lines[extinf+2], "#EXTVLCOPT:") == false {
		t.Errorf("directives are out of order: %q", lines[extinf+2])
	}

	if strings.HasPrefix(lines[extinf+3], "http") == false {
		t.Errorf("stream URL is not after the directives: %q", lines[extinf+3])
	}

	// The channel without directives stays exactly as it was: EXTINF, URL.
	for i, line := range lines {
		if strings.HasPrefix(line, "#EXTINF") && strings.HasSuffix(line, ",Plain One") {
			if strings.HasPrefix(lines[i+1], "http") == false {
				t.Errorf("channel without directives gained extra lines: %q", lines[i:i+3])
			}
			break
		}
	}

}
