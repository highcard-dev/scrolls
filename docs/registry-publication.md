# Registry updates

CI builds the Scroll UI, validates the catalog, logs into the registry and runs
`scripts/push.sh`. That shared script is the explicit catalog used by both CI
and local seeding. Release runs update the existing repositories and tags,
with category metadata pushed before Scroll artifacts.

There are no Druid account credentials, Core/Auth API calls, publication
snapshots, catalog registration calls or visibility changes. These repositories
are already public and registered; CI only updates their OCI artifacts.

## Configuration

Only two GitHub Actions secrets are required:

- `SCROLL_REGISTRY_USER`
- `SCROLL_REGISTRY_PASSWORD`

Optional repository variables override the existing defaults:

| Variable | Default |
| --- | --- |
| `SCROLL_REGISTRY_HOST` | `artifacts.druid.gg` |
| `SCROLL_REGISTRY_NAMESPACE` | `druid-team` |
| `SCROLL_PREVIEW_NAMESPACE` | `druid-team-experimental` |
| `SCROLL_REGISTRY_RUNTIME_NAMESPACE` | Release namespace |
| `DRUID_CLI_VERSION` | `v0.1.257` |

`DRUID_CLI_VERSION` selects the released CLI and default runtime image tag.
`DRUID_SCROLL_RUNTIME_IMAGE` and `DRUID_SCROLL_STEAMCMD_IMAGE` can override the
runtime image references independently. CI downloads the CLI release binary;
it does not check out or build CLI source.

Same-repository PRs push preview tags suffixed with `-pr<number>` into the
preview namespace without changing release tags or category metadata. Fork
PRs only build and validate. Runtime image references stay in the release
namespace. Both reusable-workflow callers inherit the repository secrets.

## Verification and dependencies

Run `bash scripts/tests/push-parallel.test.sh` and
`bash scripts/validate_all_scrolls.sh` locally. Use `SCROLL_PUSH_DRY_RUN=1 bash
scripts/push.sh` to inspect the catalog without registry writes. The push script
retains its serial default, bounded parallel option and category barrier.

The existing Go tree generator, Scroll validator and UI staging helper remain:
they generate or validate actual artifact content. The CI-only Go API publisher
and workflow parser are removed.

The Scroll UI still needs published Druid UI packages exporting
`ensureFileService`, `addAndRunCommand` and `stopCommand`, and a matching lockfile.
Local unpublished packages do not prove that a clean CI build will pass.
Registry updates themselves do not depend on Core publication APIs or a Druid
account.
