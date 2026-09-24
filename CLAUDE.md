# Working on xteve-reborn

An actively-maintained fork of [xteve-project/xteve](https://github.com/xteve-project/xteve).
Go backend, single-page vanilla-TS front end. MIT licensed.

## Read this first

**[`runningchangelog.md`](runningchangelog.md) is the running log of changes** — newest
first, one section per change set, with what changed, why, and how it was verified.
Read it before starting work so you know the current state and, importantly, which
things are explicitly *not yet verified*. Add an entry there when you land a change;
this repo has no PR template or commit-lore hook, so that file is the record.

Also worth knowing: `README.md` is the user-facing feature/compat document (its
"3.0.1" section is the current release notes), `changelog-beta.md` is upstream's
frozen 2.x beta history (do not append to it), and `reddit-post-xteve-reborn.md`
is a launch post, not documentation.

## Commands

```sh
go build ./...            # build
go vet ./...              # vet
gofmt -l .                # formatting check (must be empty)
go test ./...             # tests
go test -race ./...       # race tests (needs cgo / a C toolchain)
```

CI (`.github/workflows/ci.yml`) runs all of the above on every push and PR.

## The two generated-artifact rules

These are the easiest things to get wrong. Both are enforced in CI.

1. **Anything under `html/` is embedded into `src/webUI.go`** (base64, one entry per
   file) by walking the folder. After changing *anything* under `html/`, regenerate:

   ```sh
   go run ./tools/verify-embedded-assets
   ```

   CI regenerates and fails if `src/webUI.go` differs. Regeneration is
   deterministic and idempotent — running it twice with no source change must
   produce a byte-identical file.

2. **`html/js/*_ts.js` is compiled output, not source.** The sources are
   `ts/*.ts`; edit those. The project's convention is a global `tsc`:

   ```sh
   cd ts && ./compileJS.sh        # tsc *.ts --outDir ../html/js/
   ```

   There is no `package.json`/`tsconfig.json` and no pinned TypeScript version, so
   if `tsc` isn't available, prefer changes to `html/css/*.css` and `html/*.html`
   (both hand-written) over hand-editing compiled JS — editing generated JS
   silently desyncs it from `ts/`.

## Conventions

- **Concurrency**: shared state is guarded by explicit locks (`xepgLock`,
  `streamingURLsLock`, `providerLock`, `updateStateLock`). Data is written by the
  maintenance loop / WS handlers and read by every per-request response builder.
  Add a lock, and add a `-race` regression test, for any new shared map or slice.
- **Untrusted input**: channel names, group titles, and anything else sourced from
  an M3U/XMLTV provider is hostile. Use `textContent`/`escapeHTML()`, never raw
  `innerHTML`. Timeouts belong on every outbound HTTP client.
- **Assets/UI**: colours are CSS custom properties in `html/css/base.css` (`:root`),
  with a light theme in the following `@media (prefers-color-scheme: light)`
  block — add new colours as tokens there rather than hardcoding them.
- **No native dialogs**: use `showToast()` (or a popup) instead of `alert()`.
  `confirm()` still appears in a few places and should be converted.
- **Release assets are a contract**: `src/internal/up2date/client` looks up an
  exact asset name (`xteve-reborn_<version>_<os>_<arch>.zip`), reads an exact
  top-level zip entry (`xteve-reborn`, or `.exe`), and requires the
  `xteve-reborn_<version>_checksums.txt` file. `.github/workflows/release.yml`
  produces these; changing any of it silently breaks self-update for that release.

## Verification expectations

Prefer proving a change works over asserting it does. Run the tests, and for
anything touching the web UI or HTTP surface, exercise it: build the binary, run
it (`./xteve-reborn -config <dir> -port <port>`), and curl the endpoints. If
something genuinely can't be verified (no Docker, third-party API), say so
plainly in the changelog entry rather than leaving it implied.
