# Multiplayer demo

The Bun backend in the Scroll's `data/` folder runs as `start` in Druid, not in OpenHands or the browser. SQLite is built into Bun; no dependency installation or separate database container is needed. The root-mounted procedure runs `data/server.ts`; `data/highscores.sqlite` and its WAL files persist on the deployment volume. The `highscores.sqlite*` chunk has `skip_update: true`, so Scroll updates preserve the database as well.

The Rust client uses Druid's typed WebSocket WIT bridge and the shared Canvas/Keyboard component catalog. The backend runs a fixed 30 Hz simulation, broadcasts packed binary snapshots, and the browser draws on animation frames with interpolated ship positions. Controls are a single binary bitmask byte. Display names and the small top-10 leaderboard use infrequent JSON text frames.

## Build and test

From the client directory:

```sh
cargo test --locked
cargo fmt --check
sh build.sh
```

Requires Rust with `wasm32-unknown-unknown` and wasm-tools 1.262.0, not npm. Paired outputs go to `private/dist` and `public/dist`. The optional runtime `DRUID_PORT_HTTP_URL` configures both documents during rebuild.

From the Scroll's `data/` directory:

```sh
bun test game.test.ts server.test.ts
bun server.ts
```

The backend exposes `/ws`, `/health` and read-only `/highscores`. To test outside Druid, enter `ws://localhost:8080/ws` in the UI. In Druid, the server rewrites both documents with its assigned HTTP endpoint and uploads them using its existing `ui:publish` permission. This uploads artifacts only; it never creates or updates a Core public listing. Publication errors are logged and the endpoint can still be entered manually.

## Demo rules and limits

- Up to eight simultaneous anonymous pilots, one ship per connection. There is no account/session/resume framework.
- Three lives each, temporary spawn/hit shields, no friendly fire, cooperative waves. R respawns after three seconds and resets the current run score.
- Scores come only from server-verified hits and are persisted immediately. SQLite keeps the best score per display name; choosing another person's name is possible and explicitly not secure competitive identity.
- Leaving removes that ship and its bullets. High scores survive disconnects and server restarts; live arenas do not.
- Public backend access is intentional for this demo. Do not expose unrelated deployment operations or treat it as hardened multiplayer infrastructure.
