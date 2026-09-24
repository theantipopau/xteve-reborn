package m3u

import (
	"strings"
	"testing"
)

// Per-stream directives - #KODIPROP (DRM / clearkey keys), #EXTVLCOPT (player
// options such as a user agent a provider insists on), #EXTHTTP (custom
// request headers) - are not part of a channel's identity, but they must
// survive parsing, in order, so the M3U xTeVe Reborn serves can carry them on
// to M3U players.
func TestMakeInterfaceFromM3UKeepsStreamDirectives(t *testing.T) {

	var content = strings.Join([]string{
		`#EXTM3U url-tvg="http://example.com/file.xml"`,
		`#EXTINF:0 tvg-id="drm.one" group-title="Sports",DRM One`,
		`#KODIPROP:inputstream.adaptive.license_type=clearkey`,
		`#KODIPROP:inputstream.adaptive.license_key=abc123:def456`,
		`#EXTVLCOPT:http-user-agent=ExamplePlayer/1.0`,
		`#EXTHTTP:{"Cookie":"session=xyz"}`,
		`http://example.com/stream/drm1`,
		`#EXTINF:0 tvg-id="plain.one" group-title="News",Plain One`,
		`http://example.com/stream/plain1`,
	}, "\n")

	streams, err := MakeInterfaceFromM3U([]byte(content))
	if err != nil {
		t.Fatal(err)
	}

	if len(streams) != 2 {
		t.Fatalf("parsed %d streams, want 2", len(streams))
	}

	var first = streams[0].(map[string]string)

	if first["name"] != "DRM One" || first["url"] != "http://example.com/stream/drm1" {
		t.Errorf("first stream = %v", first)
	}

	if first["group-title"] != "Sports" || first["tvg-id"] != "drm.one" {
		t.Errorf("first stream lost its EXTINF parameters: %v", first)
	}

	var want = strings.Join([]string{
		"#KODIPROP:inputstream.adaptive.license_type=clearkey",
		"#KODIPROP:inputstream.adaptive.license_key=abc123:def456",
		"#EXTVLCOPT:http-user-agent=ExamplePlayer/1.0",
		`#EXTHTTP:{"Cookie":"session=xyz"}`,
	}, "\n")

	if first["_directives"] != want {
		t.Errorf("_directives =\n%q\nwant\n%q", first["_directives"], want)
	}

	// A stream without directives must not grow an empty one, and directives
	// must never leak into the filter values.
	var second = streams[1].(map[string]string)

	if len(second["_directives"]) != 0 {
		t.Errorf("second stream has directives: %q", second["_directives"])
	}

	if strings.Contains(second["_values"], "KODIPROP") {
		t.Errorf("directives leaked into _values: %q", second["_values"])
	}

}
