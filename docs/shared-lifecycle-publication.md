# Shared authored-release publication

Implementation is opt-in and locally tested. The production/PR workflows have
**not** been switched: authorization for that ongoing CI change and dedicated
Team credential provisioning remain outstanding. Do not treat this as deployed.

## Contract

`scripts/push.sh` remains the single explicit catalog and retains its category
barrier, serial default and bounded concurrency. `SCROLL_PUBLISH_MODE=lifecycle`
adds the shared Core policy after each rendered artifact push:

1. Verify the authenticated publisher's private project against the configured
   owner and registry. Reject admin/public-project registry credentials.
2. Render/stage the existing Scroll and private UI. Include inherited family
   `.meta` in the revision, not just a mutable category artifact.
3. Push into that private project with a cryptographically unique staging tag.
4. Resolve that tag once, then import its immutable digest through Core.
5. Core applies canonical identity before hashing, verifies final bytes, handles
   immutable-tag conflicts and refreshes the catalog under its repository lock.
6. When explicitly reviewed, publish through the same repository-wide operation
   used by customers. No public-registry credentials or Team bypass are involved.

The source staging artifact is retained. Backup snapshots are rejected by the
authored-release endpoint; they must use backup promotion with provenance.

## Configuration contract (not provisioned)

- `SCROLL_LIFECYCLE_URL`: trusted Core base URL, including `/core` at the gateway.
- `SCROLL_LIFECYCLE_OWNER`: identity ID of the dedicated publisher account.
- `SCROLL_REGISTRY_HOST`: scheme-less configured Harbor host.
- `SCROLL_REGISTRY_NAMESPACE`: must equal that identity ID (private staging).
- `SCROLL_REGISTRY_USER` / `SCROLL_REGISTRY_PASSWORD`: robot credential belonging
  to that private project, issued by the normal account registry-credential flow.
- `SCROLL_AUTH_URL`, `SCROLL_TEAM_EMAIL`, `SCROLL_TEAM_PASSWORD`: normal dedicated
  account sign-in. Auth must have the same trusted origin as Core. Session cookies
  stay in memory, owner JWTs refresh before lifecycle calls, and sign-out is
  attempted at exit. Auth credentials are not passed to the renderer subprocess.
- Alternatively `SCROLL_LIFECYCLE_TOKEN`: a fresh externally supplied owner JWT;
  expiration fails closed. Never commit or print credentials.
- `SCROLL_REGISTRY_RUNTIME_NAMESPACE=druid-team`: keeps runtime images separate
  from the private staging project.
- `SCROLL_TAG_SUFFIX=-<commit>`: immutable revision names for repeatable CI.
  Reusing a tag with changed content is a conflict, not an overwrite.
- Lifecycle mode sets `SOURCE_DATE_EPOCH` from the checked-out commit unless
  explicitly supplied. The matching CLI must support it and deterministic root
  layer ordering: otherwise identical rebuilds correctly conflict as different
  manifest bytes.
- `SCROLL_LIFECYCLE_PUBLISH=1` plus `SCROLL_LIFECYCLE_REVIEWED=1`: explicitly
  reviewed public publication. Without publish, imports remain private unless
  the destination was already public, in which case unreviewed import fails.
- `SCROLL_LIFECYCLE_REPOSITORY_SUFFIX=-pr<number>`: **separate private preview
  repositories**. A PR tag in an already-public release repository is not private.

Keep secrets restricted to trusted publication jobs and require a release review
gate. No production account, robot, GitHub secret, or environment has been created
by this implementation. Both Core import support and the matching runtime/server
lifecycle PRs must be deployed before enabling the workflows.

## Verification

- Go publisher tests cover shared private/public operations, immutable staging,
  expected-owner rejection before push, failed-push isolation, separate private
  preview identity, explicit review, fresh JWTs and redirect credential isolation.
- Metadata staging tests check inherited presentation and the existing private UI.
- `SCROLL_PUSH_UI=0 bash scripts/tests/push-parallel.test.sh`: passed.
- Full staging-enabled concurrency run: passed, 18 categories / 127 artifacts.
  One earlier run during concurrent edits stopped early (47 artifacts); its
  original staging error was not retained. The rerun passed without a pool fix;
  do not misreport that earlier result as a diagnosed production defect.
- Core shared policy/adapter/schema tests and real local Harbor/Postgres tests
  cover authored imports, immutable conflicts, public review guards, byte
  preservation and private staging. See the monorepo verification document.
- Authenticated helper → local gateway/auth → live Core → Harbor acceptance
  passed in 7.10s, including private import, an identical digest on rebuild,
  shared publication and anonymous visibility. The fixture's robot, projects,
  catalog row, auth account/session and development credit were cleaned up.
  Four retained fixtures from earlier diagnosis attempts were also removed with
  exact identity/name/email guards. CI credential provisioning and rollout are
  still outstanding; no production account or registry was touched.
- Earlier live failures identified fixture username/schema mistakes and a real
  ORAS creation-timestamp reproducibility gap. The timestamp fix preserves
  immutable-tag conflicts rather than weakening them.
- Subsequent retry failures exposed nondeterministic root-layer ordering in the
  CLI. A focused real OCI push regression reproduced it in 0.08s; sorting the
  discovered paths fixed the byte difference without weakening conflict checks.
  The final live test explicitly signs out after signup and signs back in with
  the dedicated fixture account. Three consecutive runs passed (8.53s, 6.25s,
  6.16s), including exact retry, publication, anonymous visibility and cleanup.

On interruption, inspect final registry state before retrying. A disconnected
HTTP request does not prove that an import or publication was rolled back.
