# Minecraft dashboard

The Rust WASM guest owns access/section-tab state and the manifest-driven Minecraft configuration editor. `src/view.openui` composes the dashboard from Druid's restricted component library; connected host components retain endpoint, developer-service, task, statistics and backup operations.

The SPA keeps the deployment header and mounts this package once below it. A generic 12-column grid gives the access panel 3 columns and the main content 9; both stack below desktop width. The Rust guest switches Dashboard, Configuration and Backups through underline-style section tabs without React routes, retaining editor state between sections. The same navigation bar includes the Documentation and Discord support links; icons and links are document-defined content. The SPA's bundled generic dashboard stays visible until this custom package is ready. Other Scroll families continue using `../ui`.

## Build and verify

Use Rust stable and the native Bytecode Alliance component tools. No Node.js,
npm, package registry or JavaScript dependencies are required:

```sh
rustup target add wasm32-unknown-unknown
cargo install --locked wasm-tools --version 1.262.0
cargo test --manifest-path guest/Cargo.toml --locked
cargo fmt --manifest-path guest/Cargo.toml --check
./build.sh guest/Cargo.toml dist --document=src/view.openui
./build.sh public/guest/Cargo.toml public/dist --document=public/src/view.openui
# Optional live rebuilds (also Rust-only):
./build.sh guest/Cargo.toml dist --document=src/view.openui --watch
```

`build.sh` is the standalone builder from Druid UI's `packages/client/build-rust.sh`.
`wit/` is a checked-in snapshot of that SDK's canonical `packages/client/wit/`
contract; update them together when changing the SDK contract. The builder supplies
its absolute path through `DRUID_WIT_PATH` (which can explicitly select custom WIT).
It compiles with Cargo, wraps the module with `wasm-tools component new`, and
checks the Druid client export before replacing the paired artifacts.
Output is `dist/app.wasm` and `dist/app.openui`; never publish only one file.
Native tests exercise parsing, validation and lossless properties editing.
Browser verification must also execute the compiled guest with host-tool fixtures
and real file-service calls.

A standard `rust:1.99-slim-bookworm` container works with the target and
`wasm-tools` installed using the commands above; no combined Node/Rust image is needed.

`../scripts/push.sh` selects this project for Minecraft and stages its source,
build script, WIT contract, Cargo.lock, manifest and artifacts under `private/`.
It excludes build output and Cargo's target directory from source staging.
Run the staging tests and `../scripts/validate_ui_coverage.sh` before publication.

## Default public page

`public/` is a separate, minimal Rust guest and OpenUI welcome page. It makes no
host-tool calls and contains no dashboard controls, configuration or private data.
Minecraft staging declares `ui.public` with `buildcommand: build_ui_public`,
`wasm: public/dist/app.wasm` and `openui: public/dist/app.openui` and packages its
source, Cargo.lock, shared Rust builder/WIT and matching artifacts under `public/`.
An explicitly declared public UI is preserved instead of replaced. Other game
families do not receive this template.

Initialization uploads this public pair alongside the private dashboard. The
deployment remains unlisted: its owner must explicitly use **Publish deployment**
to create the Core catalog entry. Releasing a newer template does not rewrite
existing deployments or their pinned public listings.

The editor retains unsaved per-file state, masks secrets, validates fields and verifies saved bytes by reloading. Like the existing file-service gateway, saving uses `save_file`; it does not add atomic compare-and-swap protection against concurrent writers.
