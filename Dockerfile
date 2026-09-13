# syntax=docker/dockerfile:1

# Multi-stage build: compiles a static binary in a full Go image, then
# copies just that binary into a minimal Alpine runtime image alongside
# ffmpeg (an optional buffer engine most self-hosted setups end up wanting)
# and the CA certs needed for HTTPS (M3U/XMLTV downloads over https, the
# GitHub Releases API for update checks).
#
# TARGETOS/TARGETARCH are populated automatically by BuildKit for
# multi-platform builds (docker buildx build --platform linux/amd64,linux/arm64) -
# no need to pass them by hand.

FROM golang:1.26-alpine AS build

WORKDIR /src

# Cached separately from the source copy below so `go mod download` only
# reruns when go.mod/go.sum actually change, not on every source edit.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.ReleaseTag=${VERSION}" -o /out/xteve-reborn .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates ffmpeg tzdata

COPY --from=build /out/xteve-reborn /usr/local/bin/xteve-reborn

# All config, playlists-by-path, and generated data live under /config -
# mount a volume there so it survives container recreation.
VOLUME ["/config"]
EXPOSE 34400

ENTRYPOINT ["/usr/local/bin/xteve-reborn"]
CMD ["-config", "/config"]
