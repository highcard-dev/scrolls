# Shared authored-release publication

## Current contract

Release and PR workflows use the same explicit catalog in `scripts/push.sh`.
They push directly to `<registry>/<identity-id>/<repository>:<tag>`; there is
no staging/import API. Core publishes reviewed revisions by copying an exact
snapshot into the owner's public username project. The private source remains.
See the monorepo's `docs/scroll-lifecycle-api-cutover.md` for the Core contract.

1. Validate and build without publication credentials.
2. Enter a fresh runner and protected `scroll-release` or `scroll-preview`
   environment. Fork PRs never publish; a missing readiness gate fails closed.
3. Build the pinned lifecycle-capable CLI and the Scroll UI.
4. Push owner-scoped revisions tagged with the source commit. Runtime images
   stay in a separate namespace. Revisions embed presentation; category-only
   artifacts are excluded from owner publication.
5. For public releases, `publish-owned` authenticates as the owner, fetches
   each repository's publication preview, verifies every pushed tag against
   that preview, and submits it unchanged as `expectedSnapshot`.
6. Core rejects changed content. The helper does not automatically retry
   conflicts or replace pending operation intent; inspect and use the current
   manual recovery flow before retrying.

Public publication approves the complete repository, not only the latest CI
tag. Use separate release and preview identities/projects. Preview pushes stay
private. Publication does not cancel an in-flight run when another run starts.

## Configuration

Provision these through the protected environments, never source control:

| Setting | Purpose |
| --- | --- |
| `SCROLL_LIFECYCLE_READY=true` | Explicit provisioning gate |
| `SCROLL_OWNER_ID` | Authenticated identity ID naming the private project |
| `SCROLL_REGISTRY_HOST` | Scheme-less Harbor host |
| `SCROLL_REGISTRY_USER`, `SCROLL_REGISTRY_PASSWORD` | Ordinary owner-scoped registry credentials |
| `SCROLL_REGISTRY_RUNTIME_NAMESPACE` | Runtime-image namespace; workflow defaults to `druid-team` |
| `SCROLL_LIFECYCLE_URL`, `SCROLL_AUTH_URL` | Core and Auth URLs sharing the trusted origin |
| `SCROLL_PUBLISH_EMAIL`, `SCROLL_PUBLISH_PASSWORD` | Ordinary publisher sign-in for public publication |
| `SCROLL_LIFECYCLE_REVIEWED=1` | Confirms content review before public publication |
| `GO_REPO_TOKEN` | Read-only access to the pinned CLI source |

Optional `DRUID_SCROLL_RUNTIME_IMAGE` and `DRUID_SCROLL_STEAMCMD_IMAGE`
override runtime image references. Owner builds use the commit timestamp unless
`SOURCE_DATE_EPOCH` is explicitly set. The workflow sets `SCROLL_PUSH_MANIFEST`
and commit-based `SCROLL_TAG_SUFFIX`; successful pushes populate the manifest.

`SCROLL_REGISTRY_NAMESPACE` is not a workflow requirement: `SCROLL_OWNER_ID`
determines artifact destinations, and runtime images have their own setting.
The shared push script still supports namespaces and category publication for
local bootstrap. It retains its serial default, bounded parallel option,
category barrier, and failure handling.

## Rollout and verification

- Deploy matching Core/runtime contracts through their normal CI.
- Configure protected environments, required reviewers and separate identities
  before enabling the readiness gate. Workflow YAML does not provision those
  policies.
- Publish a Druid UI package containing `ensureFileService`,
  `addAndRunCommand` and `stopCommand` before releasing a consumer that uses
  these imports. Local `1000.*` versions and localhost registry URLs are not
  release dependencies.
- Run `go test ./scripts/publish-owned ./scripts/stage-scroll-ui ./scripts/validate-release-workflow`
  and `bash ./scripts/tests/push-parallel.test.sh`. These test the publisher
  contract and catalog; they do not constitute a production publication.
- Verify the first approved preview/release with the provisioned credentials.
  Do not weaken approval or readiness gates to make a run green.

The former staging/import publisher and its tests were replaced. Historical
acceptance results remain in Git history and the monorepo lifecycle audit;
they do not certify this direct-owner publisher or a production rollout.
