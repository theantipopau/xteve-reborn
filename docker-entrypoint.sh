#!/bin/sh
# Drops from root to PUID:PGID (default 1000:1000) before starting, after
# making sure that user owns /config - so files written to a bind-mounted
# config folder belong to a normal user on the host, not root. Set PUID=0 to
# keep running as root.
set -e

if [ "$(id -u)" = "0" ]; then
  PUID="${PUID:-1000}"
  PGID="${PGID:-1000}"

  mkdir -p /config
  chown -R "$PUID:$PGID" /config

  exec su-exec "$PUID:$PGID" /usr/local/bin/xteve-reborn "$@"
fi

exec /usr/local/bin/xteve-reborn "$@"
