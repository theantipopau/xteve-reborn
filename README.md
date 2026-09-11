# xteve-reborn
## M3U Proxy for Plex DVR and Emby Live TV.

A standalone, personally-maintained fork of [xteve-project/xteve](https://github.com/xteve-project/xteve),
started to modernize the toolchain, fix bugs, refresh the UI, and add features on top of the
original project. Not affiliated with the upstream xTeVe project.

Original documentation for setup and configuration (still largely applicable) is
[here](https://github.com/xteve-project/xTeVe-Documentation/blob/master/en/configuration.md).

The built-in self-updater is disabled in this fork (see `xteve.go`) — updates come from
this repo's own commits/releases, not upstream's binaries.

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

