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
This fork does not (yet) publish prebuilt binaries — build from source (below).

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
