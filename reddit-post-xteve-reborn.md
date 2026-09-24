# Reddit post: xteve-reborn

> Pre-posting checklist:
> - Repo is **public** (verified): https://github.com/theantipopau/xteve-reborn
> - Prebuilt binaries + a multi-arch Docker image are on the
>   [Releases page](https://github.com/theantipopau/xteve-reborn/releases). Current release: **3.0.3**.
> - Include the repo link, the Releases link, and `ghcr.io/theantipopau/xteve-reborn:latest`.
> - Skim each subreddit's self-promotion / flair rules first, and don't post identical copies
>   everywhere the same day.
> - A screenshot of the new UI (dashboard, or the wizard's final step) gets far more engagement than
>   text alone, and neither reveals any of your channel/provider info.

---

## Suggested title

**The original xTeVe hasn't had a real release since 2021, so I forked it, fixed the crashes and
security bugs, modernized it, and rebuilt the UI — xteve-reborn 3.0.2**

Alternate:
**I forked abandoned xTeVe (the M3U proxy for Plex/Emby/Jellyfin Live TV), fixed real bugs, and
shipped it as xteve-reborn — binaries and Docker included now**

---

## Post body

If you use **xTeVe** for Plex DVR or Emby Live TV, you've probably noticed the same thing I did: the
[upstream project](https://github.com/xteve-project/xteve) hasn't had a real release since `2.2.0` in
2021. It still basically works, but it's been quietly rotting — ancient dependencies, real bugs never
fixed, and a UI that looks exactly like it did five years ago.

So instead of just complaining, I forked it: **[xteve-reborn](https://github.com/theantipopau/xteve-reborn)**.
Same core idea — it merges your M3U playlists and XMLTV guide data and presents itself to
Plex/Emby/Jellyfin as an HDHomeRun-style tuner, so your IPTV sources show up as Live TV with a real
guide. It's now at **3.0.3**, and the first releases with prebuilt binaries and a multi-arch Docker
image.

**What I fixed:**

- **A whole class of crash bugs.** The map behind every channel's mapping/EPG data was read and
  rewritten from multiple goroutines with no lock — and Go panics fatally on concurrent map access,
  so editing a mapping right as a scheduled refresh ran could take the whole app down. Audited the
  rest of the shared state and found four more instances (including the streaming-URL cache on the
  hot stream path). All locked, all with `go test -race` regression tests, and `-race` runs in CI now.
- **A stored XSS.** Channel names and group titles — whatever your M3U/XMLTV provider sends — were
  injected into several tables via `innerHTML`, so a malicious provider could run script in your
  admin session. Now `textContent` throughout.
- **Broken password hashing.** It was a single fast HMAC that ignored the per-user salt, so identical
  passwords hashed identically on every install. Now salted PBKDF2-HMAC-SHA256 (210k iterations,
  constant-time compare). *Breaking change:* delete `authentication.json` and recreate your user when
  upgrading.
- **Streams that could hang forever.** The live-stream HTTP client had *zero* timeout, so a provider
  that accepted a connection but never answered could hang the stream indefinitely. Same for
  playlist/EPG downloads, logo fetches, and update checks — all now have sane timeouts, with the
  response body left unbounded so long streams aren't cut off.
- **Backup / failover channels that share the load** (borrowed from the Threadfin fork, then made
  smarter): up to 3 backup URLs per channel, and the tuner picks a healthy source with no active
  viewers first, then the least-loaded one — so the same channel from three providers doesn't pile
  onto one account until it hits its connection limit. One dashboard click auto-fills the backups
  from your other providers by matching channel names, so multi-provider setups get failover
  across the whole lineup without editing channels one by one. (The load-aware selection idea
  came from u/c0y0t3d3n's own IPTV tuner — credited in the README.)
- **A real self-updater.** The old one shipped pointed at upstream's binaries with auto-update on (it
  would have overwritten the fork). It now pulls this repo's releases, and every release ships a
  SHA-256 checksums file it verifies before installing.
- **Official Docker image** — multi-arch on GHCR, runs as a normal user via `PUID`/`PGID`, and
  reports real container health (`HEALTHCHECK` against the tuner endpoint, new in 3.0.2).
- **Auto-refreshing guide (new in 3.0.1).** Sources used to re-download only at scheduled times
  (default midnight), so a guide change upstream didn't reach Plex until the next day. Now each
  source is checked every 60 minutes (configurable) and the guide rebuilds immediately when content
  actually changed, using conditional requests so unchanged files aren't re-sent.
- **A full UI overhaul** — new dark navy/cyan design system, real typography, new icons and logo, a
  working mobile layout, empty states, and toasts instead of blocking `alert()` popups.
- **Light theme and an accessibility pass (new in 3.0.2)** — the UI now follows your OS light/dark
  setting (it was dark-only before), keyboard focus is visible again, the detail/mapping popups are
  real dialogs for screen readers, and the login and first-run forms work with password managers.
- **Jellyfin** — same HDHomeRun protocol, and now covered by contract tests plus a container-based
  end-to-end check that drives a real Jellyfin through its own setup wizard and registers the tuner.

**Smaller stuff, same spirit:** deps updated and Go bumped 1.16 → 1.24+ (some deps were 5+ years
stale); the app's own reported guide URL 404'd (hardcoded to the wrong filename); the Log page showed
literal `&nbsp;` and silently dropped every other line past 500 entries; streams 404'd if the client
added a query string; the advertised server IP was often wrong on boxes with Docker/VPN/multiple
NICs; the configured User-Agent was set on the response instead of the request (a no-op); a stale
embedded web-asset bundle meant production builds served an old UI; roughly 4,000 lines of dead
JavaScript that was still being bundled into every build; plus a few never-merged upstream PRs ported
over and some resource leaks fixed — a file-descriptor leak in the guide's gzip writer, in 3.0.2.

**How to get it:**
- Binaries: [Releases](https://github.com/theantipopau/xteve-reborn/releases) — Windows (amd64),
  Linux and macOS (amd64 + arm64), with SHA-256 checksums.
- Docker: `docker pull ghcr.io/theantipopau/xteve-reborn:latest` (compose example in the repo;
  `--network host` gives the best shot at SSDP auto-discovery, but manual tuner entry works fine
  without it).
- From source: one `go build`.

It's still a personal project I'm actively working on, not a polished 1.0 — but you can just grab a
binary or pull the image and try it. Not affiliated with the original xTeVe project; full credit to
the original authors. MIT licensed, same as upstream.

**Repo:** https://github.com/theantipopau/xteve-reborn

Feedback, issues, and PRs welcome.

---

## Notes per subreddit (tweak before posting — don't post identical copies everywhere same-day)

- **r/PleX** — lead with the Plex DVR angle. The streaming-timeout fix and backup/failover channels
  are what Plex users care about most (fewer stuck/dead tuner sessions); the fixed guide URL and
  auto-refresh matter to anyone who's had a lineup go stale. Mention you can drop a Windows binary
  over an existing install. Flair likely required (e.g. "Tool").
- **r/selfhosted** — your best fit; the post above is written for this audience, so it can go
  basically as-is. Lean on "abandoned project, modernized + hardened + containerized" and include the
  `docker pull` line in the body rather than buried.
- **r/homelab** — works as-is. Add a line about where it fits in a typical stack (Plex/Emby/Jellyfin
  → xteve-reborn → your IPTV/OTA source).
- **r/emby** — swap the opening to lead with Emby Live TV; note Emby auto-discovery works and it also
  plays nicely as a Jellyfin tuner.
- **r/jellyfin** — lead with Jellyfin. You can now say it's verified end-to-end: a real Jellyfin
  container is driven through its own setup wizard and registers xteve-reborn as a tuner, in CI. What
  is *not* covered is a populated channel lineup or a stream, so still invite issue reports for
  anything Jellyfin-specific.
- **r/docker** — lead with the image: multi-arch, non-root PUID/PGID, GHCR, compose example,
  update-by-pull. Trim the bug list to a couple of lines.
- **r/opensource** / **r/github** — frame it as reviving an abandoned FOSS project; that audience
  cares more about the bugs found by reading code, `-race` fixes, and toolchain modernization than the
  streaming specifics.
