## xteve-reborn 3.0.5

A fixes release on top of 3.0.4. Nothing new to configure.

### XMLTV guide mapping works again

Picking an XMLTV guide in a channel's mapping popup showed no channels and an error. The guide
dropdown was saving the wrong value in this fork, so the channel list never loaded. It now saves
the guide's file name, which is what the rest of the app already uses. If a channel was mapped while
this was broken, re-select its guide once.

### The server no longer exits at startup over a broken playlist

If a playlist's file could not be read and was the last one loaded, the server logged the error and
shut down instead of starting. Now the broken playlist is skipped and the rest load as normal.

### Failover rejects non-stream responses

A provider's HTML redirect or "too many connections" page, served with status 200, could be picked
as a channel's stream. The reachability check now requires the body to start with MPEG-TS packets.
HLS playlists and empty responses are handled as before.

### Fixes from a code review

- A failed XMLTV file used to stay in the guide list and duplicate other entries. It is now removed.
- The dashboard shows provider names as text, not HTML, so a hostile provider name cannot inject
  markup into the page.
- "Backups on: 0 of N" now says what to do: fill backups from the dashboard.

### Dialogs

The restore, unsaved-settings and delete confirmations now use the app's own dialog instead of the
browser's blocking popup. They look the same and behave the same: Cancel keeps you where you were.

### Upgrading

3.0.5 is a drop-in upgrade from 3.0.x: same config folder, same endpoints, no migration. The in-app
updater (non-Docker installs) will offer it automatically. Docker users pull the new image:

```
docker pull ghcr.io/theantipopau/xteve-reborn:v3.0.5
```
