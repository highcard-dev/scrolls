# Agent notes — scrolls

Context for humans and coding agents working in this repository.

## Layout

- Game server definitions live under `scrolls/` (not the repo root).
- Each product line has a `.meta/` folder with locale markdown used for registry category metadata. Basenames must match `^[a-z]{2}-[A-Z]{2}\.md$` (for example `en-US.md`, `de-DE.md`).
- Versioned scrolls sit in per-version directories with `scroll.yaml` (for example `scrolls/minecraft/minecraft-spigot/1.21.7/`).

## Release workflow

- `.github/workflows/release.yml` installs `druid` from [highcard-dev/druid-cli](https://github.com/highcard-dev/druid-cli), validates scrolls, logs into the registry, pushes **categories**, then pushes individual scrolls.
- When adding a new scroll family, you usually need both a **Push Categories** line (for that family’s `.meta`) and **Pushing new scrolls** lines for each version directory.

## `druid registry push category` — three positional arguments

The command signature is:

```text
druid registry push category <repo> <category> [<scrollDir>]
```

| Position | Meaning |
|----------|---------|
| 1 | OCI repository (for example `artifacts.druid.gg/druid-team/scroll-minecraft-spigot`) |
| 2 | **Category label** — only used to form the OCI tag `druid-category--<category>`. It is **not** the path to `.meta`. |
| 3 | Directory containing the locale `*.md` files. Relative to the process working directory (and combined with druid’s global `--cwd` if set). Omitting this makes `scrollDir` default to `.`, so druid looks for `de-DE.md` / `en-US.md` in the wrong place. |

**Common mistake:** passing only two arguments after `category`, where the second looks like a path (for example `./scrolls/minecraft/foo/.meta`). Then that string is treated as the category label, `scrollDir` stays `.`, and the CLI errors with no files matching the locale pattern.

**Convention here:** use the same short category string as `druid registry push ... --category <name>` for that product in the same workflow (for example `minecraft`, `rust`, `palworld`, `hytale`).

## Validation

- `./scripts/validate_all_scrolls.sh` runs `druid scroll validate --strict` on every directory that contains a `scroll.yaml`.

## Source of truth for CLI behavior

- Implementations live in **druid-cli** (for example `cmd/registry_push_category.go`, `internal/core/services/registry/oci.go`). If behavior is unclear, read that repo or search it (including via Sourcebot MCP if configured).
