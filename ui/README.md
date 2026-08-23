# Scroll configuration admin UI

Every released game-server Scroll receives the same sandboxed configuration-editor component during publication. The staging command generates a family manifest, copies `dist/app.wasm` to `private/dist/app.wasm`, and adds the declarative private UI entry to the staged `scroll.yaml`.

Minecraft variants share one typed `server.properties` schema. Other families expose their native configuration files and always retain a lossless Raw view, so newly introduced or uncommon options remain editable even before a typed control is added.

The editor implementation and its tests live in `src/config-editor`. The committed WASM package is built from `src/app.tsx` with Druid UI 2.x and is rebuilt by the PR and release workflows before staging.

Rebuild the UI after changing its source:

```bash
cd ui
npm ci
npm test
npm run type-check
npm run build
```

Then run `go test ./scripts` and `./scripts/validate_ui_coverage.sh` before publishing.
