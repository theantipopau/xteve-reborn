<p align="center">
  <img src="html/img/logo-reborn.png" alt="xTeVe Reborn" width="560">
</p>

# xteve-reborn
## M3U Proxy for Plex DVR and Emby Live TV.

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
* No functional/JS behavior changed — every element ID and class the UI's TypeScript relies on
  was left intact, only the visual styling

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

#### Streaming
* Buffer with HLS / M3U8 support
* Re-streaming
* Number of tuners adjustable
* Compatible with Plex / Emby EPG

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
