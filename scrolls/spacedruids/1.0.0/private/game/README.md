# SpaceDruids

Self-contained Scroll adapted from Druid UI's Rust browser-API example. Both the private dashboard and public page use the same game and restricted Druid component vocabulary. Gameplay is local to each browser; the tiny deployment process keeps the deployment available and does not implement a multiplayer server.

Build with Cargo (`wasm32-unknown-unknown`) and wasm-tools 1.262.0:

```sh
sh build.sh
```

The build emits paired WASM/OpenUI artifacts into `private/dist` and `public/dist`. The `build_ui` Druid command rebuilds both from the deployment's `private/game/` source. Artifact upload is separate from explicitly listing the public page in Core.

`wit/` contains the canonical Druid UI client, state, host, Canvas, keyboard, animation and audio contracts used by this example. Browser API support must be enabled in the consuming Druid UI host.

Keyboard: arrows/A/D to move, Space to fire, P to pause, R to restart, M to toggle sound. The buttons also support discrete movement/fire for touch users. Sound starts disabled. High scores last for the current page session.
