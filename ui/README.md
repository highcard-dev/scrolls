# Scroll configuration admin UI

Every released game-server Scroll receives the same sandboxed configuration-editor component during publication. The staging command generates a family manifest, copies `dist/app.wasm` to `private/dist/app.wasm`, and adds the declarative private UI entry to the staged `scroll.yaml`.

Minecraft variants share one typed `server.properties` schema. Other families expose their native configuration files and always retain a lossless Raw view, so newly introduced or uncommon options remain editable even before a typed control is added.

The committed WASM package is built from `src/app.tsx` with Druid UI 2.x. SHA-256:

`15cd01c1ead7ca2cd15e46dfca1840c35c6a46f09637c6eaa6ee0cb86a8d4a8b`

Rebuild after `@druid-ui/config-editor` 2.1 is published:

```bash
cd ui
npm install
npm run build
```

Then run `go test ./scripts` and `./scripts/validate_ui_coverage.sh` before publishing.
