## xteve-reborn 3.0.6

A security, reliability and speed release on top of 3.0.5. Nothing new to configure.

### Security: other websites can no longer control xteve-reborn

Web authentication is off by default. Before this release, any website open in a browser on your network
could send commands to xteve-reborn's web interface, including setting the ffmpeg path, which runs a
program. Browser requests from another site are now refused. The web UI, Plex, Emby, Jellyfin, curl and
scripts work as before. Behind a reverse proxy that changes the `Host` header, the proxy must pass
`X-Forwarded-Host`.

We still recommend turning on web authentication if xteve-reborn is reachable by other people.

### Settings can no longer be corrupted by a crash

Settings and the channel database are now written to a temporary file and then swapped in. A crash or
power cut while saving used to be able to leave them truncated.

### Streaming fixes

- A memory leak while watching: each video segment sent kept a small record until the viewer stopped.
- A tuner that could spin one CPU core at 100% after a stream failed while a client was waiting.
- The "all tuners busy" video is now sent with the right video type.
- Two clients tuning at the same moment could miscount tuners.

### A crash risk under parallel requests is gone

Every web request rewrote the server's address in shared memory while other requests (Plex's device
discovery, the channel lineup, the playlist and guide) read it. A browser loads the web UI's files in
parallel, so this happened on every page load. It could make a response mix two addresses or, rarely,
crash. These reads and writes are now locked.

### Faster with large playlists

- Importing a large playlist for the first time is much faster: 20,000 new channels took over five minutes
  and now take a fraction of a second.
- Building the guide is faster on large XMLTV files.

### Upgrading

3.0.6 is a drop-in upgrade from 3.0.x: same config folder, same endpoints, no migration. The in-app
updater (non-Docker installs) will offer it automatically. Docker users pull the new image:

```
docker pull ghcr.io/theantipopau/xteve-reborn:v3.0.6
```
