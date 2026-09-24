## xteve-reborn 3.0.4

### Provider metadata now survives into the M3U you serve

Per-stream directives — `#KODIPROP`, `#EXTVLCOPT`, `#EXTHTTP`, `#EXTGRP` — were parsed away and
never came back. The M3U that xTeVe Reborn serves now carries them verbatim, between a channel's
`#EXTINF` and its URL, exactly where the provider put them.

That is what gets a provider-mandated `http-user-agent` (`#EXTVLCOPT`) or extra request headers
(`#EXTHTTP`) through to Kodi, TiviMate and VLC, and what carries `#KODIPROP` licence metadata for
DRM streams. To be explicit about the limits: Plex, Emby and Jellyfin parse none of these lines, and
buffered re-streaming re-produces the video (which drops DRM regardless) — this is a win for M3U
clients and for no-buffer mode.

### Failover now knows the provider's own account limit

3.0.3's selection only knew how many viewers *this* instance was serving from each URL, which is
blind to other apps sharing the same account. For Xtream Codes providers the account status
(`max_connections`, `active_cons`) is now read in the background and folded into selection:

- a source whose provider account is already **full** is no longer treated as idle,
- between equally loaded sources, the one with more free connections wins,
- providers that don't expose an account limit keep the previous behaviour exactly.

The stream path never waits for a provider: it reads a cached answer and, when that is stale,
starts a single background refresh (3s timeout, 2 minute TTL, 30s minimum interval). Chosen sources
log their remaining provider connections when known.

Reachability probes are also cached for 15 seconds now. A channel with backups used to probe up to
four URLs *per play request*; it now probes each source at most once per 15s, so a popular channel
stops re-probing every upstream for every viewer.

### The dashboard shows whether failover can work at all

- **`Backups on: 812 of 1200`** — active channels that have at least one backup configured. A
  channel without one is a channel that cannot fail over, and there was no way to see that before.
- **`Sources: 3 ok`** / **`1 failing (Provider X)`** — the outcome of the last check of every
  playlist and XMLTV source, instead of reading the log.

### Channel limits are settings now

`plex.channel.limit` and `unfiltered.channel.limit` were hard-coded at 480. They are now fields in
Settings (still 480 by default, capped at 100000). Above the unfiltered limit nothing activates
automatically until a filter exists — which is why a large Jellyfin or Emby lineup could look
empty — so that is now something to raise rather than a reason to write a dummy filter. Both are
advisory (they drive warnings and that activation gate), never a hard cap.

### Fixed: auto-fill backups did nothing after a guide rebuild

`Data.XEPG.Channels` holds structs after a rebuild and maps after loading `xepg.json`, and
auto-fill only understood maps — so on a normally rebuilt database it skipped every channel and
reported "No two providers share a channel name". It now handles both, which is what made **Fill
backups from other providers** look like it silently did nothing.

Auto-fill also matches channel names tolerantly now (`HD`, `FHD`, `(Backup)`, `-`/`_` separators),
while deliberately keeping numbered and time-shifted channels apart (`BBC One +1` is not `BBC
One`), and gained a **Preview backups** action that reports what it would change without saving
anything.

### Also

- The web UI templates are now covered by a test that renders every page and script against the
  language file, so a settings label can no longer go missing and truncate a served script. It
  immediately found a typo'd placeholder reference that had been silently leaving the temp-path
  field blank.
- Two leftovers in the buffered streaming path that set a `Content-Length` *after* the response
  headers had already been sent were removed, with the ordering contract documented — writing a
  body after declaring a zero length is what breaks restreaming to strict HTTP clients, and the
  next refactor that moves those lines will now trip over the comment instead of Plex.
- Small credit to the [xTeVe-dietpi](https://github.com/chtugha/xTeVe-dietpi) fork and the Reddit
  user who mentioned it: their private build's per-stream metadata idea is what this release's
  directive passthrough started from (see below).

### Upgrading

3.0.4 is a drop-in upgrade from 3.0.x: same config folder, same endpoints. New settings get their
defaults on first start, and the XEPG database gains a field on its next rebuild — no migration
step. The in-app updater (non-Docker installs) will offer it automatically; Docker users pull the
new image:

```
docker pull ghcr.io/theantipopau/xteve-reborn:v3.0.4
```

### Thanks

- **[c0y0t3d3n/iptv](https://github.com/c0y0t3d3n/iptv)** — load-aware source selection and
  name-matched lineup merging, which this release extends with provider account limits.
- **[chtugha/xTeVe-dietpi](https://github.com/chtugha/xTeVe-dietpi)** and the Reddit user who
  pointed at it — prompting the per-stream metadata passthrough.
- **[Threadfin](https://github.com/Threadfin/Threadfin)** — backup/failover channels and Jellyfin
  support.
- **[xTeVe](https://github.com/xteve-project/xTeVe)** by saroxan — the original project.

**Full changelog:** https://github.com/theantipopau/xteve-reborn/blob/main/runningchangelog.md
