## xteve-reborn 3.0.2

### The interface now follows your light/dark setting

The web UI was dark-only. It now ships a light theme that follows your OS or browser preference
automatically. The palette was already custom-property based, so this is the same design in a second
colour scheme rather than a redesign. There is no in-app toggle — change it in your OS.

### Fixes
- Dropdowns had no arrow. A global `appearance: none` stripped the native indicator with nothing
  put back, so a `<select>` looked exactly like a read-only text field.
- `Cancel` buttons rendered as a *second primary button* with red text on it — a stylesheet
  shorthand was leaving the accent gradient in place.
- Keyboard focus is visible again. Buttons had their focus outline removed with nothing replacing
  it, so keyboard users had no idea where they were.
- The XMLTV gzip writer leaked a file descriptor: the output file was never closed, so every guide
  rebuild leaked one and on Windows left the `.gz` locked against being replaced. Rare when the
  guide only rebuilt daily; not rare since 3.0.1 rebuilds it whenever a source changes.

### Accessibility
- Pages declare their language, the detail and mapping popups are real dialogs for screen readers,
  toasts are announced through a live region, and the CSS-background logo is labelled.
- The login and first-run forms carry autofill hints, so password managers can fill them.
- Animations respect `prefers-reduced-motion`.

### Docker
- The image now reports real health. A `HEALTHCHECK` probes the tuner endpoint, so
  `docker compose ps`, Portainer and Unraid show actual container health and
  `depends_on: {condition: service_healthy}` works. If you change the app's port, set
  `XTEVE_REBORN_PORT` to match.

  ```
  docker pull ghcr.io/theantipopau/xteve-reborn:v3.0.2
  ```

### Smaller, and better tested
- Removed roughly 4,000 lines of dead legacy JavaScript that was still being embedded into every
  build — the UI moved to TypeScript, but the old hand-written files were left behind and the
  bundler walks the whole `html/` folder.
- A Jellyfin contract suite pins the HDHomeRun discovery fields, lineup entry shape and generated
  XMLTV guide structure Jellyfin depends on, and runs in CI on every push. A container-based
  end-to-end check against a real Jellyfin is also in the repo, but has not been run yet.
- Releases are built by CI from the tag, replacing a manual, Windows-only build script.

### Upgrading
Drop the new binary over the old one, or use **Install Now** from the dashboard banner. No settings
or data changes are needed.
