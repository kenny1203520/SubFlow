import assert from 'node:assert/strict'

export function validatePublication(metadata, manifest, release, image, digest, stableBefore, stableAfter) {
  assert.match(digest || '', /^sha256:[a-f0-9]{64}$/, 'build must return a registry digest')
  assert.equal(manifest.digest, digest, 'version tag must resolve to the published digest')
  assert.equal(release.tag_name, metadata.tag)
  assert.equal(release.draft, false)
  assert.equal(release.prerelease, metadata.prerelease)
  assert.ok(release.body?.includes(metadata.notes), 'release notes must contain the exact changelog section')
  assert.ok(release.body.includes(`${image}:${metadata.version}`), 'release notes must name the exact image')
  assert.ok(release.body.includes(digest), 'release notes must contain the registry digest')
  if (metadata.prerelease) assert.equal(stableAfter, stableBefore, 'alpha must not change latest')
}
