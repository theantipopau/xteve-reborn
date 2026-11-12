package src

import (
	"net/http/httptest"
	"testing"
)

func TestSameOrigin(t *testing.T) {

	var cases = []struct {
		origin, host, forwarded string
		want                    bool
	}{
		{"", "192.168.1.3:34400", "", true},                                         // curl, Plex: no Origin
		{"http://192.168.1.3:34400", "192.168.1.3:34400", "", true},                 // the web UI itself
		{"https://xteve.example.com", "xteve.example.com", "", true},                // HTTPS proxy keeping Host
		{"https://xteve.example.com", "127.0.0.1:34400", "xteve.example.com", true}, // proxy rewriting Host
		{"http://evil.example", "192.168.1.3:34400", "", false},                     // another site in the browser
		{"http://192.168.1.3:34400.evil.example", "192.168.1.3:34400", "", false},
		{"null", "192.168.1.3:34400", "", false}, // sandboxed iframe / file://
	}

	for _, c := range cases {
		var r = httptest.NewRequest("GET", "/data/", nil)
		r.Host = c.host
		if len(c.origin) > 0 {
			r.Header.Set("Origin", c.origin)
		}
		if len(c.forwarded) > 0 {
			r.Header.Set("X-Forwarded-Host", c.forwarded)
		}
		if got := sameOrigin(r); got != c.want {
			t.Errorf("sameOrigin(Origin %q, Host %q, X-Forwarded-Host %q) = %v, want %v", c.origin, c.host, c.forwarded, got, c.want)
		}
	}
}
