package src

import "sync/atomic"

// These flags used to be plain ints/bools on System, checked and then set
// as two separate steps from different goroutines - so two rebuilds could
// both see "not running" and start at once, and a rebuild finishing could
// clear a flag another one had set. They're now only touched through the
// helpers below, which read and flip them atomically.

// scanState is non-zero while the channel database or XEPG is being
// rebuilt. The web UI shows the maintenance page and Plex/Emby's
// lineup_status reports a scan in progress while it's set.
var scanState int32

// rebuildQueued is set while a deferred XEPG rebuild is waiting for the
// current one to finish (see saveXEpgMapping), so repeated saves during a
// rebuild queue one follow-up instead of one each.
var rebuildQueued int32

// imageCachingState is set while channel/program images are being cached.
var imageCachingState int32

func scanInProgress() bool {
	return atomic.LoadInt32(&scanState) != 0
}

// tryStartScan marks a scan as started and reports true, or reports false
// without changing anything if one is already running.
func tryStartScan() bool {
	return atomic.CompareAndSwapInt32(&scanState, 0, 1)
}

func endScan() {
	atomic.StoreInt32(&scanState, 0)
}

func imageCachingInProgress() bool {
	return atomic.LoadInt32(&imageCachingState) != 0
}

func tryStartImageCaching() bool {
	return atomic.CompareAndSwapInt32(&imageCachingState, 0, 1)
}

func endImageCaching() {
	atomic.StoreInt32(&imageCachingState, 0)
}
