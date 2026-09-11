package src

import (
	"errors"
	"net"
	"net/http"
	"time"
)

// defaultHTTPClient is for small one-shot fetches (update checks, channel
// logo downloads) where the whole request should give up quickly if the
// remote server never finishes responding.
var defaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// fileDownloadHTTPClient is for downloading provider M3U/XMLTV files, which
// can be large and slow. It gives up a lot faster than dropped connections
// used to hang (indefinitely, via http.DefaultClient), but still allows
// enough time for a big file over a slow connection.
var fileDownloadHTTPClient = &http.Client{
	Timeout: 5 * time.Minute,
}

// streamHTTPClient is for live stream connections. It bounds how long we wait
// for the upstream provider to start responding (dial, TLS handshake,
// response headers), but intentionally has no overall request timeout since a
// live TV stream's body is read for as long as a client is watching.
var streamHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return errors.New("Redirect")
	},
}

// preflightHTTPClient is for quickly checking whether a stream URL is
// reachable before committing to it (used to decide between a channel's
// primary and backup URLs). Deliberately much shorter than streamHTTPClient
// so failing over to a backup doesn't make the viewer wait through a full
// connection timeout on a dead primary.
var preflightHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	},
}
