# Running changelog

A working log of changes as they land, newest first. One section per change set,
with what changed, why, and how it was verified.

Distinct from `changelog-beta.md` (upstream's old 2.x beta history) and from the
per-release notes written for GitHub Releases. This file is the running record;
entries here can be rolled into release notes when a version is tagged.

Convention: add new entries at the top under `## Unreleased` (or the version
being worked on), and say plainly what has *not* been verified yet.

---

## 3.0.2 — published, then corrected; and the Jellyfin workflow's first real run

### Release and push
This is where the 3.0.2 work was first committed — it had sat uncommitted through the whole
prep pass, and that mattered. A release had already been published from the GUI against a
`v3.0.2` tag pointing at `bfc8f4e`, a commit whose `xteve.go` still said
`Version = "3.0.1.0301"`. The attached binaries were built from the working tree and were
correct; the tag's source archive, and the GHCR container built from that tag, were not.

- Committed the work as `c0dff80` (31 files, +1634/−4155) and force-moved `v3.0.2` onto it.
- `release.yml` and `jellyfin.yml` were **untracked**, so GitHub had neither — the release
  workflow could not have run for any tag. Both are live now.
- Re-tagged once more after the `/lineup.json` fix (`4d97768`) so the shipped binaries carry
  it. Once CI could publish, it replaced the manually-uploaded assets with builds from the
  tagged commit (`-trimpath -ldflags "-s -w ..."`, ~5.9 MB against the build script's ~10.5 MB
  unstripped).
- **Follow-up:** `tools/release/build-release.ps1` passes neither `-s -w` nor `-trimpath`, so
  it no longer produces artifacts comparable to CI's. Align it or retire it now that releases
  are built from the tag.

### Bugs the first real CI runs found
1. **`/lineup.json` served `null` with no channels.** `getLineup` used `var lineup Lineup`,
   and a nil slice marshals to `null` — the HDHomeRun protocol defines the endpoint as a JSON
   array. Easiest to hit on a fresh install with no provider source, which is exactly what the
   Jellyfin container does. Fixed by initialising to `Lineup{}`, with
   `TestJellyfinEmptyLineupIsArray`; verified by temporarily reverting it, which fails with
   "= null, want []".
2. **`release.yml` could never have published anything.** Its zip step wrote into `dist/`
   without creating it, so all five build jobs exited 15 ("zip I/O error: No such file or
   directory"). Added `mkdir -p`.
3. **The publish step was not idempotent.** A re-pushed tag would have failed on "release
   already exists"; it now edits the notes and uploads with `--clobber`.
4. **`run.sh` was committed `100644`**, so CI could not invoke it at all (exit 126).

### First real run of the Jellyfin workflow
It had never been executed. It took four failures to go green, each a different cause.

The notable one: **`GET /LiveTv/TunerHosts` does not exist in Jellyfin 10.9 and answers
405**, which `curl -f` turns into a silent failure that looks exactly like "the tuner was not
persisted". The tuner had in fact been created — the POST returned our URL and an id. Two of
the four cycles were spent guessing at this before a diagnostic dump showed the 405. The
check now reads `/System/Configuration/livetv` and walks the document for any object whose
`Url` is ours.

Also fixed en route:
- **Retry the startup wizard.** A cold Jellyfin answers `/System/Info/Public` long before it
  is ready, so the first `/Startup/Configuration` POST 500s.
- **Call `GET /Startup/User` before `POST /Startup/User`.** The POST updates the *first* user,
  which does not exist yet on a new server; the GET is what runs `userManager.InitializeAsync()`
  and creates it. Jellyfin's own source marks that endpoint "TODO: Remove this method when
  startup wizard no longer requires an existing user."
- Dump `docker logs` and the offending response body on failure. Every one of the four
diagnoses came from that, not from reasoning about the API.

**Verified green:** `Jellyfin`, `CI` and `Docker` all pass on `97e1722`.

### Release artifacts re-verified once CI took over
`sha256sum -c` passes against the published checksums for all five zips; each contains exactly
one entry at the zip root (`xteve-reborn`, `xteve-reborn.exe` on Windows); and the CI
binaries carry both `3.0.2.0302` and `ReleaseTag=v3.0.2`. The release body is
`tools/release/notes-v3.0.2.md`, kept in sync by the workflow's `edit` path on re-runs.

**Still not verified:** the `HEALTHCHECK` reporting `healthy` has never been observed
directly. The Docker workflow builds the image and starts it to smoke-test the entrypoint,
which is stronger than nothing, but not the same as reading the health status.

---

## 3.0.2 — release checklist

State of the release as prepared for a `3.0.2` tag. Ticked = actually verified in this
working tree; unticked = needs a human action outside it.

**Code, docs and checks**
- [x] `Version` bumped to `3.0.2.0302` (build number moved with it)
- [x] README: new `#### 3.0.2` section, heading bumped to v3.0.2, Downloads names 3.0.2,
      Docker section documents the health check, Jellyfin note states what is/isn't tested
- [x] Reddit draft rewritten for accuracy against 3.0.1/3.0.2
- [x] `CLAUDE.md` added (build/test + the two generated-artifact rules); `README-DEV.md`
      stub replaced with pointers to it and to this file
- [x] `gofmt -l .` clean · `go vet ./...` · `go build ./...` · `go test ./...` all pass
- [x] `src/webUI.go` regenerated and byte-identical on re-run (CI's sync check will pass)
- [x] All 4 workflow YAMLs parse; `tools/jellyfin-smoke/run.sh` passes `bash -n`
- [x] No temp/scratch files left in the tree
- [ ] `go test -race ./...` — **could not run here** (no C toolchain for cgo); CI runs it

**Verified by actually running it**
- [x] App boots and reports `3.0.2.0302` (`-info`, and `/discover.json` FirmwareVersion)
- [x] `/web/` serves the UI; every referenced script, stylesheet and image returns 200
- [x] Dark **and** light themes rendered and inspected in a real browser
- [x] Component probe in both themes: select chevron, three button styles, checkbox, toasts
- [x] Keyboard focus ring visible; `.cancel` no longer paints as a primary button
- [x] `showToast()` reuses the static `aria-live` container (no duplicate created)
- [x] Release artifact layout consumed correctly by the real updater code
- [x] Jellyfin contract tests pass

**Still open before tagging (not done here)**
- [x] Run `.github/workflows/jellyfin.yml` once for real — green on `97e1722` after four
      failures, one of which was a real bug in the app (see above). Adding a `pull_request`
      trigger is now the remaining step, once it's stayed green over a Jellyfin release or two
- [ ] `docker build .` and confirm the new `HEALTHCHECK` reports `healthy`
- [x] Commit — `c0dff80`, `907dca1`, `4d97768`, `88a9fb0`, `9ad9669`, `cadd469`, `97e1722`
- [x] Tag `v3.0.2` and push; the release workflow built and published the assets
- [ ] Optionally comment on the live subreddit threads: the posted text predates the
      light theme and accessibility work

**Known side effect outside the repo**
- Running the binary with `-info` resolved the config folder to the user's home
  (`~/.xteve-reborn`) rather than the `-config` path passed to it, and created a default
  config there. Nothing inside the repository was affected.

---

## 3.0.2 — web UI modernization

Scope: theming, accessibility, and control affordances in the web UI. Confined to
`html/css/*.css` and `html/*.html` (both hand-written) — no `ts/*.ts` changes, because
`tsc` isn't installed here and hand-editing the compiled `html/js/*_ts.js` would
desync it from the TypeScript sources.

### 6. Light theme (automatic, from the OS preference)

**What:** All colours in `html/css/base.css` were already custom properties, but the
palette was dark-only and several values were hardcoded. Tokenised the stragglers
(`--on-accent`, `--scrollbar-thumb`, `--code-text`, `--debug`, `--overlay`,
`--overlay-strong`, `--input-disabled-text`) and added a
`@media (prefers-color-scheme: light)` block that redefines the whole set, plus
`color-scheme` on `:root` so native chrome (scrollbars, form controls, autofill
background) follows the theme instead of staying dark.

**Why:** A self-hosted admin UI that's dark-only regardless of the OS setting feels
stuck in the past, and the design was already token-based, so this was mostly
plumbing rather than a redesign. The light accent is deliberately darkened from
`#22d3ee` to `#0e7490` — the dark theme's cyan fails contrast as text/button colour
on a light surface.

**Files:** `html/css/base.css`, `html/css/screen.css`, all five `html/*.html`.

**Verified:** Ran the app and loaded it in a browser at both `prefers-color-scheme:
dark` and `light`; screenshotted the setup wizard in each and confirmed the palette,
contrast and card/surface hierarchy hold. Confirmed the computed tokens switch
(`--bg` `#0a0d14` → `#f4f6fa`, `--accent` `#22d3ee` → `#0e7490`, `color-scheme`
`dark` → `light`) rather than assuming the media query matched.

**Not done:** there is no in-app light/dark toggle — that needs JS, and JS here means
recompiling TypeScript. The theme follows the OS only.

### 7. Accessibility pass

**What:**
- `lang="en"` on every page (none had it).
- `:focus-visible` rings — `base.css` set `outline: none` on buttons with **no**
  replacement, so keyboard users had no visible focus anywhere; plus a matching
  treatment for the sidebar nav items, which had none either.
- `<meta name="color-scheme" content="dark light">` and dark/light `theme-color`
  pairs, so the mobile browser chrome matches the theme.
- Dialog semantics: `#popup-custom` is now `role="dialog" aria-modal="true"`.
- A static `<div id="toast-container" aria-live="polite">` in `index.html` and
  `configuration.html`. `showToast()` appends to it, so toasts are announced;
  previously the container was created by script with no live-region semantics
  at all. The script already reused an existing container by id, so no JS change
  was needed.
- `role="status"` on the loading overlay (it has no text), `role="img"` +
  `aria-label` on the CSS-background logo elements (invisible to screen readers).
- `aria-label` on the login/create-account inputs — the `h5` above each is a visual
  heading, not an associated label — plus `autocomplete="username"` /
  `current-password` / `new-password` so password managers can fill them, and
  `autofocus` on the username field.

**Why:** The UI had zero ARIA and no visible keyboard focus, and the auth forms
were unfillable by password managers.

**Files:** all five `html/*.html`, `html/css/base.css`, `html/css/screen.css`.

**Verified:** Fetched the rendered pages from the running app and confirmed the
attributes are present in the server's output, not just the source (templates
run server-side). Confirmed the real `showToast()` reuses the static container
(no duplicate is created) and leaves `aria-live="polite"` intact. Visually
confirmed the focus ring on a focused button in both themes.

**Not verified:** screen-reader behaviour (no AT available here). The clickable
`<span class="stat-action">` elements in the dashboard stat bar are still mouse-only
— they need to become real `<button>`s, which is a markup + TS change.

### 8. Modern control affordances, and a pre-existing Cancel-button bug

**What:**
- **Selects had no dropdown arrow.** The global `* { -webkit-appearance: none }`
  removed the native indicator and nothing replaced it, so a `<select>` looked
  exactly like a readonly text field. Added a drawn chevron via a tokenised
  `--select-arrow` data-URI (a different stroke colour per theme) with matching
  padding.
- **Fixed `.cancel` buttons.** `.cancel` set `background-color: transparent`, but
  the base rule sets the accent gradient through the `background` shorthand — so
  the gradient's background-image survived and a Cancel button rendered as a
  second primary button with red text on it. Changed to the `background`
  shorthand, which resets background-image too.
- `prefers-reduced-motion` support: the spinner, toast slide-in and hover fades
  are decorative, so they're disabled when the OS asks for reduced motion.
- The logo is an opaque dark plate (the artwork isn't available light-mode-safe —
  `logo_w_600x200.png` is the same art with a light plate but *white* text, so it's
  unusable), which read as a stray dark rectangle on a light page. Rounded it into
  a deliberate brand chip (`border-radius` on the containers that the artwork fills
  exactly).

**Why:** These are the small things that make a UI feel dated, and two of them
(the missing select arrow and the broken Cancel button) were functional defects,
not taste.

**Files:** `html/css/base.css`, `html/css/screen.css`.

**Verified:** Injected a component probe into the running app's own page and read
back computed styles: `padding-right: 34px`, the chevron data-URI applied at
`right 12px / 12px 8px`, and `.cancel` resolving to `background-image: none` with
the danger border/text. Screenshotted the probe in both themes — select chevron,
primary/secondary/cancel buttons, checkbox and all three toast severities.

  This is worth calling out because the first attempt was wrong: the chevron was
  added in a separate `select` rule, and the existing `select { padding: 9px 10px }`
  shorthand (same specificity, later in the file) silently reset `padding-right`,
  putting the arrow on top of the text. Only the computed-style probe caught it.

**Not verified:** a full click-through of every screen in both themes. The wizard,
login and component surfaces were checked; the dashboard, mapping editor and
settings screens were not (they need a populated config/config wizard run).

### 9. Added `CLAUDE.md` (agent instructions)

**What:** New root `CLAUDE.md` covering the build/test commands, the two
generated-artifact rules (regenerate `src/webUI.go` after any `html/` change;
`html/js/*_ts.js` comes from `ts/`), the concurrency/untrusted-input conventions,
the release-asset naming contract, and a pointer to this file as the running log.

**Why:** There was no agent/contributor instruction file, so the non-obvious traps
(the bundle regeneration requirement, the TS→JS direction, the fact that this
changelog is the record) had to be rediscovered each time.

**Files:** `CLAUDE.md`.

### 10. Release prep for 3.0.2

**What:** Bumped `Version` in `xteve.go` from `3.0.1.0301` to `3.0.2.0302` (the last
segment is the build number parsed in `main`, so it needs to move together with the
version). Rewrote the README's release-facing parts: the "What's different from
upstream xTeVe" heading now says v3.0.2, a new `#### 3.0.2` section covers this batch,
the Downloads section names 3.0.2 and points at the CI release workflow, the Docker
section documents the health check and `XTEVE_REBORN_PORT`, and the Jellyfin
requirements note now states what is actually tested versus what hasn't been run.
Replaced the `README-DEV.md` stub with pointers to `CLAUDE.md` and this file.

**Why:** The README is the main user-facing document and the front page of a repo that
has just been posted to Reddit, so it shouldn't describe a release that no longer
matches the code.

**Files:** `xteve.go`, `README.md`, `README-DEV.md`, and the two test blocks in
`src/jellyfin_test.go` (decoupled from the version constant — they used to hardcode
`3.0.1.0301`, which would have quietly drifted).

**Verified:** `gofmt`/`vet`/`build`/`test` clean; ran the binary and confirmed
`/discover.json` reports `FirmwareVersion: 3.0.2` and the dashboard version string
follows from it. No `html/` change in this step, so no bundle regeneration was needed.

**Not verified:** the 3.0.2 tag/release itself — nothing is tagged or pushed from here.

---

## 3.0.2 — post-3.0.1 maintenance pass

Scope: four maintenance items plus one bug the new tests exposed.

### 1. Removed ~4,000 lines of dead legacy JavaScript (front-end cleanup)

**What:** Deleted ten files under `html/js/` that no page loads any more:
`authentication.js`, `base.js`, `classes_ts.js`, `configuaration.js`, `data.js`,
`files.js`, `log.js`, `mapping-editor.js`, `menu.js`, `users.js`. Regenerated
`src/webUI.go` to drop them from the embedded bundle.

**Why:** The UI was migrated to TypeScript (`ts/*.ts` → `html/js/*_ts.js`), and
every page now loads only the seven compiled `*_ts.js` files. The old
hand-written files were left behind. Because `src/html-build.go` walks the whole
`html/` folder and base64-embeds everything into `src/webUI.go`, those dead files
were being compiled into every binary — roughly 4,000 lines / ~120 KB of
unreachable JavaScript, plus a genuinely confusing "which of these two files is
live?" hazard for anyone touching the UI. `classes_ts.js` was a particularly easy
trap: it still defined `MainMenu`/`MainMenuItem`, which now live in
`menu_ts.ts`/`settings_ts.ts`.

**Files:** `html/js/*` (10 deletions), `src/webUI.go` (regenerated).

**Verified:**
- Every file confirmed unreferenced first (no `<script src>` in any page, and no
  references from `ts/`). The released UI works with only the `*_ts.js` set, so
  the loaded set is self-sufficient by construction.
- `go build ./...`, `go vet ./...`, `go test ./...` all pass; `gofmt -l .` clean.
- Bundle regeneration is idempotent — running `go run ./tools/verify-embedded-assets`
  twice produces a byte-identical `src/webUI.go`, so the CI sync check passes.
  Entry count drops 51 → 41.
- **Ran the actual app** and fetched it over HTTP: `/web/` returns 200, all seven
  referenced scripts return 200, a deleted file (`js/mapping-editor.js`) returns
  404 as it should, and CSS/logo/nav-icon assets still return 200.
- Not verified: a browser click-through of every UI screen (no browser tooling
  used here). The pages, scripts and assets all serve, and no behaviour changed,
  but the interactive flows weren't exercised.

### 2. Docker health check

**What:** Added a `HEALTHCHECK` to the `Dockerfile` probing the app's own
`/discover.json` tuner endpoint, plus a matching `healthcheck` in the
`docker-compose.yml` example. Added an `XTEVE_REBORN_PORT` env var (default
`34400`) that only the health check reads, so a changed app port can be
accommodated without editing the image.

**Why:** There was no health check, so Docker/Portainer/Unraid (and compose
`depends_on: condition: service_healthy`) could only tell "process started" from
"process dead" — not "running but wedged" (e.g. stuck behind a failed provider
refresh). The app already serves `/discover.json` on every interface, so the probe
needed no new endpoint. Uses busybox `wget` because Alpine's `curl` isn't
installed in the image.

**Files:** `Dockerfile`, `docker-compose.yml`.

**Verified:** `docker-compose.yml` parses as valid YAML; the `$$`-escaped
`${XTEVE_REBORN_PORT:-34400}` is left for the container's shell to expand rather
than interpolated by Compose. Confirmed the server binds `:port` on all
interfaces (`src/webserver.go`), so the loopback probe works under both host and
bridge networking.

**Not verified:** no `docker build`/`docker run` was possible in this environment,
so the health check has not actually reported `healthy` yet — worth a one-off
`docker build . && docker run` before the next release.

### 3. Release builds moved into CI

**What:** Added `.github/workflows/release.yml`. On a `v*` tag it builds all five
platform binaries (windows/amd64, linux/amd64, linux/arm64, darwin/amd64,
darwin/arm64), zips each with the binary at the zip root, generates the
`xteve-reborn_<version>_checksums.txt` file, and creates the GitHub release
(auto-generated notes; `-` in the version ⇒ prerelease so it can't take the
`latest` pointer).

**Why:** Cutting a release meant running `tools/release/build-release.ps1` by hand
on Windows, so official binaries could only be produced from one machine and one
OS, and the release had to be assembled manually afterwards.

**Files:** `.github/workflows/release.yml` (new).

**Verified:** YAML parses. The load-bearing part — the artifact layout the in-app
updater requires — was reproduced locally and checked with the *real* updater code:
built `xteve-reborn_3.0.1_linux_amd64.zip` with the workflow's exact flags, then
confirmed `parseChecksum` accepts the generated checksums file, that the zip hashes
to the listed value, and that `extractFile` extracts the `xteve-reborn` entry from
it (13,279,392 bytes). That check caught a real bug in the workflow as first
written: `sha256sum ./*.zip` writes `./`-prefixed names, and `parseChecksum`
compares against `filepath.Base(url)`, so **every release would have refused to
self-update**. Fixed to a bare `*.zip` glob.

**Not verified:** the workflow has not run on GitHub Actions yet (no tag pushed).

### 4. Jellyfin coverage

**What:** Two things, split by what each can actually prove:

- `src/jellyfin_test.go` — deterministic contract tests, no Docker: the fields
  Jellyfin reads from `/discover.json` (including that `LineupURL` points back at
  the host the client reached us on), the `device.xml` UPnP discovery fields
  Jellyfin matches on, `/lineup_status.json` (and that `ScanPossible` stays 0 so
  clients don't hang waiting on a scan), the `{GuideNumber, GuideName, URL}` shape
  of lineup entries, and the generated XMLTV guide — root element, channel
  `id`/`display-name`, and that programmes are attached to the *lineup* channel id
  with start/stop/title carried through.
- `tools/jellyfin-smoke/run.sh` + `.github/workflows/jellyfin.yml` — a real
  Jellyfin container, its startup wizard completed over its own HTTP API, asked to
  add the app as an `hdhomerun` tuner host and then confirmed to have kept it.

**Why:** The README says Jellyfin "should just work, not yet verified end-to-end",
because it's served by the same HDHomeRun emulation as Plex/Emby. That's exactly
the kind of claim that rots silently — a field Jellyfin needs disappears, and you
find out from a user's issue report. The guide is the sharpest edge: a programme
attached to the wrong channel id isn't an error, it just shows as "no guide data".

**Files:** `src/jellyfin_test.go` (new), `tools/jellyfin-smoke/run.sh` (new),
`.github/workflows/jellyfin.yml` (new).

**Verified:** The three Go tests pass (`go test ./src/ -run 'TestJellyfin|TestXMLTVGuide'`),
and run in the existing CI. `run.sh` passes `bash -n`; the workflow parses as YAML.

**Update — the workflow has now actually run, and passes.** See
"First real run of the Jellyfin workflow" below: it failed four times before going
green, three of those for reasons in the workflow itself and one of them a real bug
in the app. What follows is what was still unknown when it was written.

**Known limits — read before trusting the workflow:**
- It has one green run on `main` (`97e1722`). It's still deliberately *not* wired to
  `pull_request`, and should be proven green across a Jellyfin release or two before
  it gates merges.
- `JF_IMAGE` is pinned to `jellyfin/jellyfin:10.9.11`. **Verified to exist** on Docker Hub
  via its tags API (multi-arch: amd64, arm/v7, arm64; still being pulled), so the pin
  itself is sound and `docker pull` won't fail on a missing tag. It is, however, a
  September 2024 release, so moving to a current Jellyfin is a deliberate follow-up —
  and doing so would also re-check the API shapes below.
- The exact Jellyfin API paths/shapes used (`/Startup/*`,
  `/Users/AuthenticateByName`, `POST /LiveTv/TunerHosts`) are best-effort from the
  published API. `GET /LiveTv/TunerHosts` does **not** exist in 10.9 (405); the check
  reads `/System/Configuration/livetv` instead.
- The workflow covers discovery + tuner registration, **not** a populated lineup or
  a stream fetch. Doing that needs a seeded provider source (an M3U entry in
  `settings.json` plus an active mapped channel), so the lineup and guide shapes are
  covered by the Go tests instead.
  Extending the workflow to assert channels/streams once it's green is the obvious
  follow-up.

### 5. Fixed a file-descriptor leak in `compressGZIP` (found by the new tests)

**What:** `src/compression.go`'s `compressGZIP` created the output `.gz` file and
never closed it — and because it used `f, err := os.Create(...)` inside the
function, it also shadowed the named return error, so a failure to create the file
was silently swallowed. It now closes the gzip writer and the file, reports the
first error, and returns early when the path is empty (compression off).

**Why:** This leaked a file descriptor on **every** guide rebuild, and on Windows
left the `.gz` file locked so it couldn't be replaced or deleted. It was rare when
the guide was only rebuilt at the daily scheduled refresh, but 3.0.1 rebuilds the
guide whenever a source changes (checked every 60 minutes), so a long-running
instance accumulated handles steadily.

**Files:** `src/compression.go`.

**Verified:** `TestXMLTVGuideContract` calls the real `createXMLTVFile` path. Before
the fix, the test's `t.TempDir()` cleanup failed on Windows with "The process
cannot access the file because it is being used by another process"; after the fix
it passes cleanly and the guide content assertions hold. `go test ./...`,
`go vet`, `gofmt` clean.

**Not verified:** `go test -race` could not run in this environment (no C toolchain
for cgo); CI runs it. The change adds no concurrency, so no race-safety impact is
expected.

---

## Verification summary for this pass

| Check | Result |
| --- | --- |
| `gofmt -l .` | clean |
| `go vet ./...` | pass |
| `go build ./...` | pass |
| `go test ./...` | pass (incl. 3 new Jellyfin tests) |
| `go test -race ./...` | not run here (no gcc); runs in CI |
| embedded bundle idempotent + in sync | yes (51 → 41 entries) |
| app serves `/web/` + all referenced assets | yes, verified over HTTP |
| release artifact consumable by updater | yes, verified against real updater code |
| `docker build` / healthcheck actually reported healthy | not run here |
| Jellyfin container workflow executed | not run here |

## Follow-ups this left open

**Infrastructure**
- Run `.github/workflows/jellyfin.yml` once for real; fix the pinned tag and any
  API-shape drift, then add a `pull_request` trigger.
- Extend the Jellyfin workflow to seed a provider source and assert a populated
  lineup and a stream fetch.
- `docker build` once to confirm the health check reports `healthy`.

**Web UI**
- Give the mapping/search tables a virtualised or fragment-based renderer — they
  currently `appendChild` one row at a time, so a 10k-channel playlist builds the
  whole DOM on every menu open.
- Replace the per-action WebSocket with one persistent multiplexed connection. It
  would retire the shared-lock class of bug entirely and let the server push state
  instead of the client polling `updateLog` on a timer.
- Turn the clickable `<span class="stat-action">` elements into real `<button>`s so
  they're keyboard-operable.
- Convert the three remaining `confirm()` dialogs to the in-app popup.
- Add an in-app theme toggle (needs a TS change, so it needs `tsc`).
- A proper light-mode logo asset, so the dark brand plate isn't being worked around
  with CSS rounding.
- Deep-linkable menu state, so refresh/Back doesn't lose your place.
