#!/usr/bin/env bash
#
# End-to-end check that a real Jellyfin can talk to this app as a Live TV
# tuner. This exists because "Jellyfin should work, it speaks the same
# HDHomeRun protocol" is a claim about a third party's client, and the only
# way to keep it honest is to run that client against us.
#
# What it proves:
#   * the app serves /discover.json, /lineup.json and /lineup_status.json with
#     the shapes Jellyfin's tuner parser requires;
#   * a stock Jellyfin (its own startup wizard completed, over its real HTTP
#     API) accepts the app as an "hdhomerun" tuner host and keeps it;
#   * with a fake M3U provider seeded into the app's config, the lineup is
#     populated, Jellyfin imports the channels, and the stream chain works:
#     the app's /stream/<id> endpoint hands the client the provider URL and
#     that URL actually serves video bytes.
#
# What it deliberately does NOT prove: Jellyfin's own playback stack actually
# transcoding/playing those bytes (that is Jellyfin's ffmpeg against a fake
# stream, which would test Jellyfin, not this app). The generated XMLTV guide
# structure is covered deterministically by src/jellyfin_test.go.
#
# Env overrides: APP_PORT, JF_PORT, JF_IMAGE, APP_BIN.

set -euo pipefail

APP_PORT="${APP_PORT:-34400}"
JF_PORT="${JF_PORT:-8096}"
# Pinned by default so a Jellyfin-side change can't silently turn this red;
# bump deliberately. Override to test against a different release.
JF_IMAGE="${JF_IMAGE:-jellyfin/jellyfin:10.9.11}"
JF_CONTAINER="xteve-reborn-jellyfin-smoke"
APP_BIN="${APP_BIN:-.devdata/jellyfin-smoke/xteve-reborn}"

APP_CONFIG="$(mktemp -d)"
JF_CONFIG="$(mktemp -d)"
JF_CACHE="$(mktemp -d)"
APP_PID=""
PROVIDER_PID=""
PROVIDER_PORT=""

# The image may drop to a non-root user at startup; make sure it can write its
# own config and cache whichever uid it ends up as.
chmod 777 "$JF_CONFIG" "$JF_CACHE"

info() { echo "==> $*"; }
fail() { echo "FAIL: $*" >&2; exit 1; }

cleanup() {
  docker rm -f "$JF_CONTAINER" >/dev/null 2>&1 || true
  if [ -n "$APP_PID" ]; then kill "$APP_PID" >/dev/null 2>&1 || true; fi
  if [ -n "$PROVIDER_PID" ]; then kill "$PROVIDER_PID" >/dev/null 2>&1 || true; fi
}
trap cleanup EXIT

command -v jq >/dev/null || fail "jq is required"
[ -x "$APP_BIN" ] || fail "$APP_BIN not found or not executable (build it first)"

# ---------------------------------------------------------------------------
# 1. Start the app and check the HDHomeRun endpoints Jellyfin reads.
# ---------------------------------------------------------------------------
info "starting $APP_BIN on :$APP_PORT (config: $APP_CONFIG)"
"$APP_BIN" -config "$APP_CONFIG" > "$APP_CONFIG/app.log" 2>&1 &
APP_PID=$!

APP_READY=""
for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${APP_PORT}/discover.json" >/dev/null 2>&1; then
    APP_READY=1
    break
  fi
  sleep 1
done
if [ -z "$APP_READY" ]; then
  cat "$APP_CONFIG/app.log" >&2 || true
  fail "the app never served /discover.json"
fi

DISCOVER="$(curl -fsS "http://127.0.0.1:${APP_PORT}/discover.json")"
DEVICE_ID="$(echo "$DISCOVER" | jq -r '.DeviceID // empty')"
FRIENDLY="$(echo "$DISCOVER" | jq -r '.FriendlyName // empty')"
LINEUP_URL="$(echo "$DISCOVER" | jq -r '.LineupURL // empty')"
[ -n "$DEVICE_ID" ] || fail "/discover.json has no DeviceID"
[ -n "$FRIENDLY" ] || fail "/discover.json has no FriendlyName"
case "$LINEUP_URL" in
  */lineup.json) ;;
  *) fail "/discover.json LineupURL = '$LINEUP_URL', want it to end in /lineup.json" ;;
esac
info "app device: $FRIENDLY ($DEVICE_ID)"

curl -fsS "http://127.0.0.1:${APP_PORT}/lineup_status.json" | jq -e 'has("ScanInProgress")' >/dev/null \
  || fail "/lineup_status.json is missing ScanInProgress"
curl -fsS "http://127.0.0.1:${APP_PORT}/lineup.json" | jq -e 'type == "array"' >/dev/null \
  || fail "/lineup.json is not a JSON array"

# ---------------------------------------------------------------------------
# 2. Start Jellyfin.
# ---------------------------------------------------------------------------
info "starting Jellyfin ($JF_IMAGE)"
docker run -d --name "$JF_CONTAINER" --network host \
  -v "$JF_CONFIG:/config" -v "$JF_CACHE:/cache" "$JF_IMAGE" >/dev/null

JF_READY=""
for _ in $(seq 1 90); do
  if curl -fsS "http://127.0.0.1:${JF_PORT}/System/Info/Public" >/dev/null 2>&1; then
    JF_READY=1
    break
  fi
  sleep 2
done
if [ -z "$JF_READY" ]; then
  docker logs "$JF_CONTAINER" >&2 || true
  fail "Jellyfin never came up on :$JF_PORT"
fi
info "Jellyfin up: $(curl -fsS "http://127.0.0.1:${JF_PORT}/System/Info/Public" | jq -r '.Version // "?"')"

# ---------------------------------------------------------------------------
# 3. Complete Jellyfin's startup wizard (it refuses API work before this).
# ---------------------------------------------------------------------------
info "completing Jellyfin's startup wizard"
# Jellyfin answers /System/Info/Public well before its startup wizard is usable:
# on a cold container the first requests come back HTTP 500 while it is still
# creating its database, which is exactly what the first real run of this
# workflow hit. Retry rather than treating the first response as final, and
# dump the container log if a step never succeeds - otherwise a failure here is
# a bare "500" with no way to tell what went wrong.
jf_retry() {
  local method="$1"
  local path="$2"
  local attempt
  shift 2
  for attempt in $(seq 1 20); do
    if curl -fsS -X "$method" "http://127.0.0.1:${JF_PORT}${path}" \
      -H 'Content-Type: application/json' "$@" >/dev/null 2>&1; then
      return 0
    fi
    sleep 3
  done
  docker logs "$JF_CONTAINER" >&2 || true
  fail "${method} ${path} never succeeded against Jellyfin"
}

jf_retry POST /Startup/Configuration \
  -d '{"UICulture":"en-US","MetadataCountryCode":"US","PreferredMetadataLanguage":"en"}'

# The GET is load-bearing, not a read: POST /Startup/User updates the *first*
# user, but on a brand-new server no user exists yet, and it is GET
# /Startup/User that runs userManager.InitializeAsync() and creates one.
# Jellyfin's own source marks that endpoint "TODO: Remove this method when
# startup wizard no longer requires an existing user." Skip it and the POST
# throws "Sequence contains no elements" and answers 500.
jf_retry GET /Startup/User
jf_retry POST /Startup/User -d '{"Name":"admin","Password":"admin"}'
jf_retry POST /Startup/RemoteAccess \
  -d '{"EnableRemoteAccess":true,"EnableAutomaticPortMapping":false}'
jf_retry POST /Startup/Complete

AUTH_BASE='MediaBrowser Client="xteve-reborn-smoke", Device="ci", DeviceId="xteve-reborn-smoke", Version="1.0"'

TOKEN="$(curl -fsS -X POST "http://127.0.0.1:${JF_PORT}/Users/AuthenticateByName" \
  -H 'Content-Type: application/json' -H "Authorization: ${AUTH_BASE}" \
  -d '{"Username":"admin","Pw":"admin"}' | jq -r '.AccessToken // empty')"
[ -n "$TOKEN" ] || fail "could not authenticate against Jellyfin"
AUTH="${AUTH_BASE}, Token=\"${TOKEN}\""

# ---------------------------------------------------------------------------
# 4. Add the app as a tuner, and confirm Jellyfin keeps it.
# ---------------------------------------------------------------------------
info "registering the app as an hdhomerun tuner"
CREATED="$(curl -fsS -X POST "http://127.0.0.1:${JF_PORT}/LiveTv/TunerHosts" \
  -H 'Content-Type: application/json' -H "Authorization: ${AUTH}" \
  -d "{\"Type\":\"hdhomerun\",\"Url\":\"http://127.0.0.1:${APP_PORT}\",\"FriendlyName\":\"xteve-reborn\",\"TunerCount\":2}")" \
  || fail "Jellyfin rejected the tuner host"
[ -n "$(echo "$CREATED" | jq -r '.Id // empty')" ] \
  || fail "Jellyfin accepted the request but returned no tuner id"
info "Jellyfin created tuner host $(echo "$CREATED" | jq -c '{Id,FriendlyName,Url}')"

# Verify it stuck. There is no GET on /LiveTv/TunerHosts in 10.9 - reading it
# back with GET answers 405 - so read the Live TV configuration the POST
# actually writes to instead. The lookup walks the whole document for any
# object whose Url is ours rather than naming a property, so it does not
# depend on the wrapper getting that name right.
FOUND_TUNER=""
CONFIG_DUMP=""
for _ in $(seq 1 10); do
  CONFIG_DUMP="$(curl -fsS "http://127.0.0.1:${JF_PORT}/System/Configuration/livetv" \
    -H "Authorization: ${AUTH}" 2>/dev/null || true)"
  if echo "$CONFIG_DUMP" | jq -e --arg want "http://127.0.0.1:${APP_PORT}" \
    '[.. | objects | select(.Url? == $want)] | length > 0' >/dev/null 2>&1; then
    FOUND_TUNER=1
    break
  fi
  sleep 3
done
if [ -z "$FOUND_TUNER" ]; then
  echo "Live TV configuration as Jellyfin reports it:" >&2
  echo "$CONFIG_DUMP" >&2
  echo >&2
  fail "the tuner was not present in Jellyfin's Live TV configuration afterwards"
fi
info "Jellyfin accepted and persisted xteve-reborn as an HDHomeRun tuner"

# Informational: a populated lineup needs a configured provider source, which
# this script doesn't set up (see the header). Report either way.
CHANNELS="$(curl -fsS "http://127.0.0.1:${JF_PORT}/LiveTv/Channels" -H "Authorization: ${AUTH}" | jq 'length' 2>/dev/null || echo 0)"
info "Jellyfin reports ${CHANNELS} channel(s)"
if [ "$CHANNELS" -eq 0 ]; then
  echo "    (expected without a configured M3U/XMLTV source; the lineup and guide"
  echo "     shapes are asserted by src/jellyfin_test.go instead)"
fi

echo
echo "OK: Jellyfin discovered and registered xteve-reborn over the HDHomeRun protocol."
echo

echo "============================================================"
echo "PHASE 2: populated lineup + playable stream"
echo "============================================================"
echo

# ---------------------------------------------------------------------------
# 5. Seed a provider: a fake M3U server serving one channel whose stream is a
#    real MPEG-TS payload, so the full chain can be pulled end to end.
# ---------------------------------------------------------------------------
info "starting fake M3U provider"

# A tiny but valid MPEG-TS payload: 5 empty 188-byte packets with the sync
# byte. Enough for a client to read bytes; nothing here decodes video, which
# is fine - the app is a proxy, not a decoder.
TS_PAYLOAD="$(printf '\x47%.0s' $(seq 1 940))"

PROVIDER_PORT="${PROVIDER_PORT:-45987}"
python3 - "$PROVIDER_PORT" <<'PYEOF' > /dev/null 2>&1 &
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

port = int(sys.argv[1])

M3U = """#EXTM3U
#EXTINF:-1 tvg-id="smoke.test" tvg-name="Smoke Channel" group-title="Test",Smoke Channel
http://127.0.0.1:{port}/stream/smoke.ts
""".format(port=port)

TS = bytes([0x47]) * 188 * 5

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/playlist.m3u"):
            body = M3U.encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/vnd.apple.mpegurl")
            self.end_headers()
            self.wfile.write(body)
        elif self.path.startswith("/stream/smoke.ts"):
            self.send_response(200)
            self.send_header("Content-Type", "video/mp2t")
            self.send_header("Content-Length", str(len(TS)))
            self.end_headers()
            self.wfile.write(TS)
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, *args):
        pass

HTTPServer(("127.0.0.1", port), Handler).serve_forever()
PYEOF
PROVIDER_PID=$!
sleep 1
curl -fsS "http://127.0.0.1:${PROVIDER_PORT}/playlist.m3u" | grep -q "Smoke Channel" \
  || fail "fake provider did not serve its playlist"
info "fake provider serving on :${PROVIDER_PORT}"

# ---------------------------------------------------------------------------
# 6. Seed the provider into the app's generated settings.json and restart the
#    app. On start it downloads the playlist (files.update defaults to on),
#    builds the channel database and, with EPG source PMS, exposes the channel
#    in the HDHomeRun lineup directly.
# ---------------------------------------------------------------------------
info "seeding the provider into the app config and restarting"
kill "$APP_PID" 2>/dev/null || true
wait "$APP_PID" 2>/dev/null || true
APP_PID=""

APP_SETTINGS="$APP_CONFIG/settings.json"
[ -f "$APP_SETTINGS" ] || fail "the app did not write $APP_SETTINGS on first start"

python3 - "$APP_SETTINGS" "$PROVIDER_PORT" <<'PYEOF' || fail "could not edit the app's settings.json"
import json, sys
path, port = sys.argv[1], sys.argv[2]
s = json.load(open(path, encoding='utf-8'))
s['files']['m3u'] = {"M1smoke": {"name": "Smoke Provider",
                                 "file.source": "http://127.0.0.1:%s/playlist.m3u" % port,
                                 "type": "m3u", "tuner": 1}}
s['epgSource'] = 'PMS'
s['tuner'] = 2
json.dump(s, open(path, 'w', encoding='utf-8'), indent=2)
PYEOF

"$APP_BIN" -config "$APP_CONFIG" > "$APP_CONFIG/app2.log" 2>&1 &
APP_PID=$!

APP_READY=""
for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${APP_PORT}/discover.json" >/dev/null 2>&1; then
    APP_READY=1
    break
  fi
  sleep 1
done
if [ -z "$APP_READY" ]; then
  cat "$APP_CONFIG/app2.log" >&2 || true
  fail "the app never came back after the provider was seeded"
fi

# The database build downloads the playlist and maps the channel; wait until
# the lineup actually contains it.
LINEUP_COUNT=0
for _ in $(seq 1 30); do
  LINEUP_COUNT="$(curl -fsS "http://127.0.0.1:${APP_PORT}/lineup.json" | jq 'length' 2>/dev/null || echo 0)"
  [ "$LINEUP_COUNT" -ge 1 ] && break
  sleep 2
done
[ "$LINEUP_COUNT" -ge 1 ] || { cat "$APP_CONFIG/app2.log" >&2; fail "lineup stayed empty after seeding a provider"; }
info "app lineup populated: ${LINEUP_COUNT} channel(s)"

LINEUP_ENTRY="$(curl -fsS "http://127.0.0.1:${APP_PORT}/lineup.json" | jq -c '.[0]')"
echo "$LINEUP_ENTRY" | jq -e '.GuideName == "Smoke Channel" and .GuideNumber and (.URL | test("/stream/"))' >/dev/null \
  || fail "lineup entry shape wrong: $LINEUP_ENTRY"
STREAM_URL="$(echo "$LINEUP_ENTRY" | jq -r '.URL')"

# ---------------------------------------------------------------------------
# 7. The stream chain: tuner endpoint redirects (no buffer) to the provider,
#    and the provider serves the bytes.
# ---------------------------------------------------------------------------
info "checking the stream chain"
STREAM_LOCATION="$(curl -fsS -o /dev/null -w '%{redirect_url}' "$STREAM_URL")"
[ "$STREAM_LOCATION" = "http://127.0.0.1:${PROVIDER_PORT}/stream/smoke.ts" ] \
  || fail "stream endpoint redirected to '$STREAM_LOCATION', want the provider URL"

curl -fsS "$STREAM_LOCATION" | head -c 188 | od -An -tx1 | grep -q "47" \
  || fail "the provider did not serve MPEG-TS bytes through the chain"
info "stream chain verified: tuner -> provider URL -> MPEG-TS bytes"

# ---------------------------------------------------------------------------
# 8. Jellyfin must see the channel now.
# ---------------------------------------------------------------------------
info "asking Jellyfin to refresh its channel list"

# Jellyfin imports Live TV channels lazily, on its scheduled guide refresh.
# Trigger that task through the scheduler API instead of waiting for it.
GUIDE_TASK_ID="$(curl -fsS "http://127.0.0.1:${JF_PORT}/ScheduledTasks" -H "Authorization: ${AUTH}" \
  | jq -r '[.[] | select(.Key == "RefreshGuide" or .Name == "Refresh Guide")][0].Id // empty' 2>/dev/null || true)"
if [ -n "$GUIDE_TASK_ID" ]; then
  curl -fsS -X POST "http://127.0.0.1:${JF_PORT}/ScheduledTasks/Running/${GUIDE_TASK_ID}" \
    -H "Authorization: ${AUTH}" >/dev/null 2>&1 \
    && info "triggered Jellyfin's Refresh Guide task (${GUIDE_TASK_ID})" \
    || info "could not trigger the guide task; polling anyway"
else
  info "guide task not found on this Jellyfin version; polling anyway"
fi

JF_CHANNELS=0
for _ in $(seq 1 30); do
  JF_CHANNELS="$(curl -fsS "http://127.0.0.1:${JF_PORT}/LiveTv/Channels" -H "Authorization: ${AUTH}" | jq 'length' 2>/dev/null || echo 0)"
  [ "$JF_CHANNELS" -ge 1 ] && break
  sleep 3
done
[ "$JF_CHANNELS" -ge 1 ] || fail "Jellyfin never imported the seeded channel"
JF_CHANNEL_NAME="$(curl -fsS "http://127.0.0.1:${JF_PORT}/LiveTv/Channels" -H "Authorization: ${AUTH}" | jq -r '.[0].Name // empty')"
info "Jellyfin imported ${JF_CHANNELS} channel(s): '$JF_CHANNEL_NAME'"

# Jellyfin's Live TV program guide refresh is scheduled and lazy; the channel
# import above is the tuner-protocol contract this script exists to check.

echo
echo "OK: populated lineup verified end to end - provider -> app -> Jellyfin."
