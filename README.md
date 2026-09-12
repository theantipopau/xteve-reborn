<p align="center">
  <img src="html/img/logo-reborn.png" alt="xTeVe Reborn" width="560">
</p>

# xteve-reborn
## M3U Proxy for Plex DVR, Emby Live TV, and Jellyfin Live TV.

A standalone, personally-maintained fork of [xteve-project/xteve](https://github.com/xteve-project/xteve),
started because upstream hadn't seen a real release in about five years. This fork modernizes the
toolchain, fixes bugs, hardens the Plex/Emby streaming path, and gives the web UI a serious visual
overhaul, on top of the original project. Not affiliated with the upstream xTeVe project.

Original documentation for setup and configuration (still largely applicable) is
[here](https://github.com/xteve-project/xTeVe-Documentation/blob/master/en/configuration.md).

The built-in self-updater is disabled in this fork (see `xteve.go`) — updates come from
this repo's own commits/releases, not upstream's binaries.

---

## What's different from upstream xTeVe (v3.0.0)

Upstream's last tagged release was `2.2.0`, from 2021. Everything below is new in this fork.

#### Toolchain & dependencies
* Go bumped from 1.16 → 1.24+ (tested on 1.27); module renamed `xteve-reborn`
* All dependencies updated to current versions: `gorilla/websocket` v1.4.2 → v1.5.3,
  `koron/go-ssdp` v0.0.2 → v0.9.1, `golang.org/x/{crypto,net,sys,text}` bumped from
  2020-era pseudo-versions to current releases
* Removed the abandoned `kardianos/osext` dependency — it only existed to work around a gap
  in the standard library that hasn't existed since Go 1.8 (`os.Executable()`)
* Added a GitHub Actions CI workflow: `gofmt`, `go vet`, `go build`, `go test` on every push/PR
* `.gitattributes` added to stop line-ending churn on every commit
* Whole tree run through `gofmt`

#### Correctness fixes
* **Channel mapping bug**: the "no XMLTV file/mapping assigned" check in `xepg.go` tested
  `XmltvFile` twice instead of also checking `XMapping` — channels with one set but not the
  other were silently skipped
* **Silent JSON bug**: a stray space in the `buffer.size.kb` struct tag broke `omitempty` for
  that field without erroring
* Fixed several `fmt.Errorf(fmt.Sprintf(...))` redundancies that were tripping `go vet`'s
  format-string checks, and removed dead/unreachable code after unconditional returns
* **Windows build-asset bug**: the dev-mode/production web-asset bundler
  (`html-build.go`) built its generated Go source with raw string concatenation; on Windows,
  file paths contain backslashes, which produced invalid Go escape sequences and could silently
  corrupt or fail to compile the embedded web UI. Now uses `strconv.Quote` and forward-slash-
  normalized keys, so it's correct on every OS
* **Dev-mode fallback bug**: `-dev` mode was supposed to read UI files straight from disk so
  changes show up without recompiling, but it actually gated on the *compiled-in* asset map
  first and 404'd on any file added after the last build. Dev mode now reads from disk directly

#### Plex / Emby connection & streaming reliability
* The live-stream HTTP client was a bare `&http.Client{}` with **no timeout of any kind** — a
  provider that accepted the connection but never responded would hang the stream goroutine
  forever. Replaced with a shared client that bounds dial/TLS/response-header time (so a dead
  provider fails fast) while leaving the response body unbounded (so long-running live TV
  streams aren't cut off)
* Same problem existed for M3U/XMLTV downloads, channel-logo image fetches, and update-version
  checks — all used `http.DefaultClient`/a fresh no-timeout client with no bound at all. Each now
  has an appropriately-sized timeout for what it's fetching (a few seconds for small
  API calls, minutes for large playlist/EPG downloads)
* Disabled the built-in GitHub self-updater's upstream pointer — it shipped pointed at
  `xteve-project/xTeVe-Downloads` with auto-update *on*, which would have overwritten this
  fork with upstream's binaries the first time it checked

#### Web UI
* Full visual refresh of `html/css/base.css` and `html/css/screen.css`: new dark navy/cyan
  design system (CSS custom properties for color, spacing, radius), modern system-font stack
  instead of Arial, redesigned buttons/inputs/checkboxes/tables, refined sidebar navigation with
  active-state highlighting, softer shadows and rounded cards throughout
* New logo and favicon (`html/img/logo-reborn*.png`, `favicon-*.png`) wired into every page,
  including the login/first-run/setup-wizard screens, which previously showed no logo at all
  (`#header.imgCenter` existed in the markup but had never been styled)
* No markup/JS behavior changed by the visual refresh itself — every element ID and class the
  UI's TypeScript relies on was left intact, only the styling

#### UI/UX reliability
* **Found and fixed a silent-freeze race condition**: the whole web UI serializes every
  server request through one global "connection in flight" lock (`SERVER_CONNECTION`), and a
  background log poll (`updateLog`, firing every 10 seconds) shared that exact same lock with
  every user-initiated action. Any click — opening a mapping/user/file popup, Save, Bulk Edit —
  that happened to land while a poll was mid-flight was silently dropped: no error, no
  indication anything happened, just an unresponsive button until the next poll cycle left a
  gap. The log poll now has its own independent lock, so it can never block a real interaction.
  Found this by reading `network_ts.ts`/`menu_ts.ts` after reproducing exactly this symptom
  while testing the mapping editor.
* Added a client-side request timeout (15s): previously, a request that never got a response —
  dropped connection, server restarting mid-request — left the UI-wide lock stuck forever with
  no explanation, requiring a full page reload to recover. It now releases the lock and tells
  the user what happened.
* Set up a working TypeScript build (`ts/*.ts` → `html/js/*_ts.js`) to verify these fixes compile
  cleanly and match the project's existing compiled-output conventions, since the repo didn't
  have a documented, reproducible way to do this before.

#### Security
* **Fixed stored XSS**: channel names, group titles, and similar fields — all sourced from
  whatever M3U/XMLTV provider is configured — were injected into the Mapping/Playlist/XMLTV/
  Filter/Users tables via `innerHTML` instead of `textContent`. A malicious or compromised
  provider could embed markup/script in a channel name and have it execute in the admin's
  browser session. Fixed in the shared table-cell renderer plus three related spots (the
  mapping-detail popup's subtitle, the post-edit row refresh, and the log viewer).
* **Hardened password storage**: the password hashing scheme computed a fast, single HMAC-SHA256
  over a constant message and silently ignored the per-user salt it was given, meaning identical
  passwords hashed identically across every installation. Replaced with salted PBKDF2-HMAC-SHA256
  at 210,000 iterations, with constant-time comparison. **Breaking change**: delete
  `authentication.json` and recreate your user if upgrading from before this fix.
* Fixed a panic: a malformed HTTP Basic-Auth header could crash the request goroutine on an
  out-of-bounds slice access.
* Hardened the session cookie (explicit `Path=/`, `SameSite=Lax`).

#### Fixes sourced from upstream's stale open PRs
xteve-project/xteve has several small, correct, never-merged PRs sitting open for years. Ported
the ones that still apply:
* EPG programs with no poster of their own fell back to a blank image instead of the channel's
  logo, due to a variable-naming bug (`xteve-project/xteve#302`).
* The server's advertised IP was whichever non-loopback address enumeration happened to hit last
  — often wrong on a box with Docker bridges, VPN tunnels, or multiple NICs. Now detected via the
  OS's own outbound-routing decision, falling back to the old behavior if there's no route out
  (`xteve-project/xteve#266`).
* The configured User-Agent was never actually sent on M3U/XMLTV downloads — it was being set on
  the HTTP *response* instead of the *request*, a no-op (`xteve-project/xteve#398`, minus that
  PR's unrelated bundled Dockerfile changes).
* A few missing `resp.Body.Close()` calls and a defer-inside-a-loop that held every image-cache
  download's file handle open until the whole batch finished instead of per-item.

#### Concurrency and a broken guide URL
* **Fixed a crash-on-concurrent-access bug in the core channel database.** `Data.XEPG.Channels`
  (the map backing every channel's mapping/EPG data) was read and rewritten from several
  independent goroutines — WS command handlers, the maintenance loop's scheduled provider
  refresh, any HTTP handler building a lineup or M3U/XMLTV response — with zero coordination
  between them. Go maps panic with a fatal, unrecoverable error on concurrent read/write; this
  isn't a theoretical risk, it's a real crash waiting for the wrong two things to happen at once
  (e.g. editing a channel mapping right as a scheduled provider refresh rebuilds the database).
  Added a dedicated lock around every function that touches this map (`xepgLock` in
  `config.go`), verified it actually fixes the problem — not just that it compiles — by getting
  a C toolchain working so `go test -race` runs at all (it needs cgo), confirming the exact
  unlocked access pattern reliably triggers a data race, then confirming the locked version
  doesn't. `go test -race ./...` is now part of CI so this can't quietly regress.
* **Fixed a broken guide URL.** While testing the fix above end-to-end, found that the app's
  own reported XMLTV guide URL — the one shown in the dashboard and meant to be pasted into
  Plex/Emby — pointed at `/xmltv/xteve.xml`, hardcoded, while the actual generated file is named
  after the app (`xteve-reborn.xml` in this fork). Anyone copying the URL this app itself
  displayed would get a 404. Root cause: this fork's rename from `xTeVe` never propagated to
  these hardcoded literals. Fixed everywhere the filename was hardcoded instead of derived from
  the app name, so it's also correct for anyone who renames the binary again later.
* **Audited the rest of the shared state for the same class of bug**, since the XEPG fix was
  never really about XEPG specifically — it's about anything shared between the maintenance
  loop / WS handlers and the per-request response builder with no lock. Found and fixed four
  more: `Data.Cache.StreamingURLS` (written on every lineup/M3U rebuild, read on every single
  stream start — the hottest path in the app, had no lock at all; given its own dedicated
  `streamingURLsLock` rather than reusing `xepgLock` so a channel-surf never waits on a full EPG
  rebuild), `System.Notification` (same alias-escapes-into-a-response bug as the original XEPG
  fix, on a different map), `Data.XMLTV.Mapping` (the exact same bug, three lines away from
  where it had already been fixed for `Data.XEPG.Channels` — its writer had no lock either), and
  `Data.Streams.*`/`Data.StreamPreviewUI.*`/`Data.Filter`/`Data.Playlist.M3U.Groups.*` (all
  rebuilt wholesale with no lock, read by every WS response). Added regression tests for each,
  verified by temporarily removing a lock and confirming `go test -race` actually catches it
  before restoring the fix — same discipline as the original.

#### Ported from Threadfin
[Threadfin](https://github.com/Threadfin/Threadfin) is a more actively-developed community fork
of xTeVe (1.7k+ stars, regular releases, adds Jellyfin support). Rather than switching to it
wholesale, brought over the pieces that fit this fork's own architecture:
* **Backup/failover channels** — up to 3 backup stream URLs per channel. A lightweight
  reachability check picks the first one that actually responds (primary first) before a client
  connects, so a dead primary provider doesn't mean a dead channel. Channels with no backups
  configured pay no cost — the check is skipped entirely.
* **Jellyfin listed as a supported target** — it speaks the same HDHomeRun tuner protocol Plex
  and Emby do, so this should already work; added to the docs accordingly.

#### Clearer EPG source guidance
Addressed a real complaint: Plex only ever showed guide data for US channels, nothing for
international channels (Sky Sports, Australian sports, etc.) or 24/7 loop channels. Root cause is
config, not a bug — the "PMS" EPG Source option delegates entirely to Plex/Emby's own built-in
guide database, which is overwhelmingly US/Canada-focused. The wizard and Settings screen now
explain this plainly and point non-US users at XEPG (bring your own XMLTV guide) instead, plus
mention the existing "xTeVe Dummy" placeholder-schedule fallback for channels with no real EPG
data at all.

#### Web UI structure & polish
* **Replaced the entire sidebar icon set.** The old one was a random mismatched grab-bag — flat
  raster PNGs in inconsistent styles, and the "Filter" nav item's icon was literally a heart
  (favorites, not filtering — clearly a leftover mistake, not a design choice). Redrew all 8 as a
  cohesive, consistent line-icon SVG family, including a proper funnel for Filter.
* **Redesigned the dashboard status bar** from a dense, monospace, table-of-abbreviations into a
  real stat grid: primary at-a-glance numbers (Streams, EPG Source, Errors, Warnings, etc.) up
  front with clear labels, technical details (OS/Arch, DVR IP, UUID) visually de-emphasized, and
  the long M3U/XEPG URLs broken out into their own row instead of competing for space in the same
  grid.
* **Actually implemented mobile responsiveness** that had been started and abandoned: the main
  dashboard's viewport meta tag was commented out (while every other page had it enabled — clearly
  an oversight), and a `.phone` CSS class was referenced throughout the markup but had no matching
  rule anywhere, so it did nothing. Re-enabled the viewport tag, removed the dead class, and built
  real mobile-first behavior: the sidebar collapses to an icon-only rail below 620px width instead
  of eating 60%+ of a phone screen, and the technical stats/URL row hide until there's room for
  them.
* **Added empty states.** Tables (Playlist, Filter, XMLTV, Users, Mapping) rendered as a bare
  header with nothing below when empty, with no indication of why or what to do next. Each now
  shows a short, specific message pointing at the right next action.
* **Removed a dead, misleading setting.** "Automatic update of xTeVe" in Settings still looked
  live and toggleable, but this fork's self-updater is unconditionally disabled at a lower level
  (see the "Plex / Emby connection & streaming reliability" section above) — the checkbox hasn't
  done anything since that change. Removed it rather than leave a control that lies about what it
  controls.
* **Replaced native `alert()` popups with toast notifications.** Blocking browser alerts looked
  jarring next to the rest of the redesigned UI and stop all interaction until dismissed. Added a
  small toast system (color-coded by severity, auto-dismissing, click to close now) and swapped
  every real alert over to it. While auditing every call site, found three that had nothing useful
  to say: a password-confirmation check that alerted the literal placeholder text `"sdafsd"`
  instead of the real inline error message it already sets, and two blank `alert()` calls (one
  fired every time a file picker was dismissed without choosing a file) that just popped an empty
  dialog for no reason. All three were dead debug leftovers - removed rather than converted.

#### Fixed the embedded web-asset bundle going stale, and made sure it can't again
Production builds serve the entire `html/` folder (markup, CSS, JS, images) from a single
generated Go file, `src/webUI.go`, embedded as base64 so the binary is self-contained. A
production-mode smoke test turned up new icon assets 404ing that worked fine in `-dev` mode —
the bundle had gone stale and never picked up several rounds of Web UI work. Root cause and fix,
in order:
* **The generator's map iteration order was random.** Every regeneration reordered its ~90+
  entries even when nothing under `html/` had actually changed, so a real diff and a
  same-content reshuffle looked identical — which is exactly how genuine changes kept getting
  discarded as "just noise" during testing. Fixed by sorting entries before writing them out, so
  regenerating with no source changes now produces a byte-identical file.
* **The sort itself wasn't platform-stable.** It sorted on the raw, OS-native path separator
  (`\` on Windows, `/` on Linux), which compare differently — so a bundle built on Windows and one
  built on Linux from the *same* `html/` tree could still legitimately disagree. Fixed by sorting
  on the already slash-normalized key instead.
* **A `.gitattributes` line-ending rule was silently corruption-prone.** `*.ts text eol=lf`,
  added for the TypeScript sources under `ts/`, also matched `html/video/stream-limit.ts` — an
  MPEG-TS binary video sample that just happens to share the extension. Git's clean filter would
  rewrite `\r\n` byte sequences inside that binary file on its next `git add`. Caught before any
  actual corruption landed (verified with `git hash-object`, filtered vs. `--no-filters`); fixed
  by scoping the TypeScript rule to `ts/*.ts` and explicitly marking the video file binary.
* **16 tracked files had stale CRLF sitting in the working tree.** Left over from before
  `.gitattributes` enforced `eol=lf` — git never retroactively fixes already-checked-out files,
  and `git status`/`diff` hide the discrepancy entirely since they normalize on the fly for
  comparison. The bundle generator reads files directly off disk, bypassing git's filters, so it
  was embedding those stale CRLF bytes. A fresh checkout never has this problem, which is exactly
  why the bug was invisible locally and only surfaced against a truly clean checkout.
* **Added a CI check** (`tools/verify-embedded-assets`) that regenerates the bundle on every push
  and fails the build if it doesn't match what's committed, so this class of bug gets caught
  immediately instead of shipping silently in a release binary.

#### Wizard now ends by telling you exactly what to paste into Plex
Feedback after `3.0.0-pre`: the setup wizard finished silently and dropped straight into the
dashboard, leaving the user to find the right address themselves among several stat-bar fields —
confusing on a box with more than one network interface — and figure out Plex/Emby setup
unassisted. It now ends on a dedicated step: the exact address in a copy-button box, plus one-line
instructions for Plex (Settings → Live TV & DVR → Set Up Plex DVR, with a manual-entry fallback if
auto-discovery doesn't find it) and Emby/Jellyfin. While building this, found and fixed a real bug:
the copy button called `navigator.clipboard.writeText()` unconditionally and always claimed
success — that API requires a secure context (HTTPS or localhost), but this app is normally reached
over plain `http://<lan-ip>:<port>`, so the write silently failed there and the toast lied. Added a
fallback (`copyTextToClipboard()`) that reports failure honestly instead.

---

## Requirements
### Plex
* Plex Media Server (1.11.1.4730 or newer)
* Plex Client with DVR support
* Plex Pass

### Emby
* Emby Server (3.5.3.0 or newer)
* Emby Client with Live-TV support
* Emby Premiere

### Jellyfin
* Jellyfin Server (10.7.1 or newer)
* Jellyfin Client with Live TV support
* Add xteve-reborn as a Live TV tuner source of type "HDHomeRun" — the HDHomeRun tuner protocol
  this project emulates is a de facto standard, so Jellyfin should detect it the same way Plex
  and Emby do. Not yet verified end-to-end against a real Jellyfin instance in this fork; please
  open an issue if you hit anything Jellyfin-specific.

--- 

## Features

#### Files
* Merge external M3U files
* Merge external XMLTV files
* Automatic M3U and XMLTV update
* M3U and XMLTV export

#### Channel management
* Filtering streams
* Channel mapping
* Channel order
* Channel logos
* Channel categories
* Up to 3 backup stream URLs per channel — if the primary fails to respond, the next reachable
  one is used automatically (see the Mapping editor's channel popup)

#### Streaming
* Buffer with HLS / M3U8 support
* Re-streaming
* Number of tuners adjustable
* Compatible with Plex / Emby / Jellyfin EPG

---

## Downloads
Prebuilt binaries (Windows/Linux/macOS, amd64+arm64) are published on the
[Releases page](https://github.com/theantipopau/xteve-reborn/releases). The current release is a
prerelease (`3.0.0-pre`) — functionally complete and CI-tested, but new enough to not have real-world
mileage yet. You can also build from source (below).

#### Docker images from the original project (Linux 64 Bit)
Thanks to @alturismo and @LeeD for creating the Docker Images.

**Created by alturismo:**  
[xTeVe](https://hub.docker.com/r/alturismo/xteve)  
[xTeVe / Guide2go](https://hub.docker.com/r/alturismo/xteve_guide2go)  
[xTeVe / Guide2go / owi2plex](https://hub.docker.com/r/alturismo/xteve_g2g_owi)

Including:  
- Guide2go: XMLTV grabber for Schedules Direct  
- owi2plex: XMLTV file grabber for Enigma receivers

**Created by LeeD:**  
[xTeVe / Guide2go / Zap2XML](https://hub.docker.com/r/dnsforge/xteve)  

Including:  
- Guide2go: XMLTV grabber for Schedules Direct  
- Zap2XML: Perl based zap2it XMLTV grabber  
- Bash: A Unix / Linux shell  
- Crond: Daemon to execute scheduled commands  
- Perl: Programming language   

---

## Build from source code [Go / Golang]

#### Requirements
* [Go](https://golang.org) 1.24 or newer

#### Build
```
go build .
```

---
