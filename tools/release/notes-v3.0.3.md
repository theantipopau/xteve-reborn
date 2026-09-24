## xteve-reborn 3.0.3

### Backup channels that share the load

Backup channels (up to 3 backup URLs per channel) previously picked the **first source that
answered** a quick reachability check. That fixes dead providers, but with the same channel
available from several IPTV providers it always favored the fastest responder — which could be the
account already serving streams, and the one about to be cut off for exceeding its connection
limit.

Selection is now load-aware:

- A healthy **and idle** primary wins immediately — one probe, exactly the cost the old behavior
  paid. Channels without backups configured pay nothing, as before.
- If the primary is down **or already serving streams**, the first reachable source with no active
  viewers wins.
- If every reachable source is busy, the one with the fewest active viewers wins, spreading
  viewers across providers instead of stacking them on one account.
- If nothing responds, the primary is returned unchanged and the existing error handling takes
  over, exactly as before.

The idea — pick the source with the most free connections — comes from
[c0y0t3d3n's iptv project](https://github.com/c0y0t3d3n/iptv); thanks for sharing it. The counts
here are per-instance (how many viewers *this* xTeVe Reborn is serving from each URL);
provider-truth, account-wide counts need Xtream Codes support, scoped in
[issue #1](https://github.com/theantipopau/xteve-reborn/issues/1).

### One click: auto-fill backups across providers

Backup channels required editing every channel by hand. The dashboard's XEPG Channels card now has
**"Fill backups from other providers"**: every active channel whose name (case-insensitive) is
carried by two or more of your M3U providers gets its empty backup slots filled with the other
providers' URLs — at most one backup per provider, never the channel's own URL, never duplicating
a backup you configured manually. **"Rebuild backups"** (with a confirmation) rebuilds all three
slots of multi-provider channels from scratch, replacing manually-entered ones.

When nothing can be filled, the toast says why: no two providers share a channel name, shared
names exist but each is single-provider, or the slots are already full. A guide rebuild runs
automatically after a fill.

### Fixed: UDPxy ignored backup URLs

With an [UDPxy](https://github.com/pderichai/udpxy) relay configured for multicast IPTV, only the
channel's primary URL was rewritten from `udp://@...` to HTTP through the relay. A backup chosen
on failover could bypass the relay entirely — and `udp://` URLs can't be probed for reachability,
so multicast backups were effectively unusable. All candidate URLs are rewritten before selection
now, so whichever source wins is both probeable and directly playable by the client.

### Verified harder

- The Docker smoke test now fails unless the container's own `HEALTHCHECK` reports `healthy` —
  previously it only checked the endpoints from outside, and Docker's verdict was never read.
- The Jellyfin end-to-end workflow grew a second phase: a fake M3U provider (a small Python HTTP
  server, no new dependencies) is seeded into the app's configuration, the app restarts and builds
  its database, and the workflow asserts a populated lineup, follows the `/stream/` redirect to
  the provider's bytes, and confirms Jellyfin imports the channel. Previously it only proved tuner
  registration against an empty lineup.

### Project

- A small project website: [theantipopau.github.io/xteve-reborn](https://theantipopau.github.io/xteve-reborn/)
  (served from this repo's `docs/` folder).
- Added repository topics so the project is findable for iptv / m3u / xmltv / hdhomerun searches.

### Upgrading

3.0.3 is a drop-in upgrade from 3.0.x: same config folder, same endpoints, no settings changes.
The in-app updater (non-Docker installs) will offer it automatically; Docker users pull the new
image:

```
docker pull ghcr.io/theantipopau/xteve-reborn:v3.0.3
```

**Full changelog:** https://github.com/theantipopau/xteve-reborn/blob/main/runningchangelog.md
