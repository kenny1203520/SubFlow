# SubFlow Release Process

Version tags drive publication. A release-branch push runs CI. An annotated v<version> tag runs .github/workflows/release.yml, publishes the single runtime image to GHCR and creates the matching GitHub Release.

## Credentials and metadata

Configure Actions secrets DOCKERHUB_USERNAME and DOCKERHUB_TOKEN for entitled access to Docker Hardened Images. GHCR publication uses GITHUB_TOKEN with packages/write and contents/write. Package visibility is configured separately in GitHub Packages.

frontend/package.json is the display-version source. Its version, both lockfile version fields, tag (without v) and exactly one dated nonempty CHANGELOG section must agree. Metadata validation rejects branches, malformed SemVer and missing/mismatched entries before building.

Production needs a persistent SUBFLOW_SETTINGS_ENCRYPTION_KEY. Preserve it with the PocketBase backup; never put credentials or bearer tokens into release artifacts.

## Preparation and Git Flow

Start from clean updated develop and create release/<version>. Update package/lockfile versions and changelog, then open a release PR back to develop. Commit independent fixes separately using Conventional Commits.

Run backend commands in backend/ and frontend commands in frontend/:

```powershell
rtk go vet ./...
rtk go test ./...
rtk npm ci
rtk npm audit --audit-level=moderate
rtk npm test -- --coverage
rtk npm run build
rtk npm run test:e2e
```

Go tests precede frontend build in CI to retain the tracked embedded development placeholder. Trusted CI also builds/qualifies the production container and Alpha 1 upgrade/restore. Fork PRs cannot access DHI credentials; container qualification must pass on a trusted branch before publishing.

Stable releases merge into master before tagging and back into develop. Prereleases tag the verified release-branch commit directly and merge back into develop after publication; master stays on stable.

```powershell
rtk git status --short --branch
rtk git rev-parse HEAD
rtk git tag -a v0.2.0-alpha.4 -m "SubFlow 0.2.0-alpha.4"
rtk git push origin v0.2.0-alpha.4
```

## Workflow gates and outputs

The tag workflow validates metadata, then runs reusable Verify: Go vet/tests/coverage and govulncheck, npm install/audit, 25 release contract tests, frontend tests/build and all Playwright desktop/touch projects. Screenshots, reports and retained failure traces/video are uploaded even when later steps fail.

Before publication, a production container must pass setup, registration, login, permission, direct reload, version and logout-audit checks. An isolated Alpha 1 data fixture is backed up offline, upgraded, restarted and restored to another volume under the old image. Only test-owned containers/volumes are removed.

Publishing pushes the full version tag. Stable releases also update major.minor and latest; prereleases update neither stable alias. Release notes contain the exact changelog section and versioned/digest Docker pull commands.

Post-publication checks compare the actual registry digest and GitHub release to expected outputs and ensure prereleases have not changed latest. Release outcome fails unless metadata, Verify and publication all succeed. A green Verify job alone does not prove publication.

## Recovery and evidence

For interrupted tags, inspect the failed/cancelled job and rerun without moving the tag:

```powershell
rtk gh run view <run-id> --repo kenny1203520/SubFlow
rtk gh run rerun <run-id> --failed --repo kenny1203520/SubFlow
```

Tags containing the dispatch-enabled workflow can also be dispatched explicitly:

```powershell
rtk gh workflow run release.yml --ref v0.2.0-alpha.4 --repo kenny1203520/SubFlow
```

Branch dispatch is intentionally rejected. Older tags lacking workflow_dispatch use rerun. Never silently replace a published version tag.

Retain workflow/source SHA, image digest and Release URL. release-evidence artifacts (30 days) contain metadata, notes, stable snapshot, container smoke, upgrade/restore and publication results. Trusted container and Playwright evidence lasts 14 days. After publication, merge the release PR into develop with a merge commit.

See [Alpha 4 qualification](releases/0.2.0-alpha.4.md), [Alpha 3 qualification](releases/0.2.0-alpha.3.md), [Playwright coverage](../frontend/e2e/COVERAGE.md) and [deployment](deployment.md).
