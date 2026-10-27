# Scroll configuration admin UI

Every released game-server Scroll receives the same sandboxed configuration editor during publication. Staging generates a family manifest, copies `dist/app.wasm` and its companion `dist/app.openui` into `private/dist/`, and adds the private UI entry to the staged `scroll.yaml`.

Minecraft variants share one typed `server.properties` schema. Other families expose their native configuration files and always retain a lossless Raw view, so newly introduced or uncommon options remain editable even before a typed control is added.

The editor behavior and its tests live in `src/config-editor`. Druid UI 4 builds `src/app.ts` into a WASM guest; `src/view.openui` declares the document. The SPA owns the trusted React configuration-editor component and scoped file-service tools. Parsing, validation, secret masking, lossless edits and save verification remain in the guest.

Use Node.js 24 or newer. The current SDK pin is `4.0.0-rc.1`; until that RC is publicly published, local development obtains it from Verdaccio through `druid-ui-dev.sh setup`.

Rebuild the UI after changing its source:

```bash
cd ui
npm ci
npm test
npm run type-check
npm run build
```

Then run `go test ./scripts/stage-scroll-ui` and `./scripts/validate_ui_coverage.sh` before publishing.
