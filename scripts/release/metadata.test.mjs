import test from 'node:test'
import assert from 'node:assert/strict'
import { releaseMetadata } from './metadata.mjs'

const version = '0.2.0-alpha.2'
const pkg = { version }
const lock = { version, packages: { '': { version } } }
const changelog = `## [Unreleased]\n\n## [${version}] - 2026-10-22\n\n### Fixed\n- Restore publishing.\n\n## [0.2.0-alpha.1] - 2026-10-21\nOld notes.`

test('extracts only the candidate changelog and marks alpha as prerelease', () => {
  const result = releaseMetadata(`v${version}`, pkg, lock, changelog.replaceAll('\n', '\r\n'))
  assert.equal(result.prerelease, true)
  assert.equal(result.notes, '### Fixed\n- Restore publishing.')
})
test('stable releases are not prereleases', () => {
  assert.equal(releaseMetadata('v1.2.3', { version: '1.2.3' }, { version: '1.2.3', packages: { '': { version: '1.2.3' } } }, '## [1.2.3] - 2026-10-22\n- Published.').prerelease, false)
})
for (const tag of ['develop', '0.2.0-alpha.2', 'v01.2.3', 'v0.2.0-alpha.02', 'v0.2.0;echo bad', 'v0.2.0+build']) {
  test(`rejects invalid tag ${tag}`, () => assert.throws(() => releaseMetadata(tag, pkg, lock, changelog), /SemVer/))
}
test('rejects each inconsistent version source', () => {
  for (const [p, l] of [[{ version: 'wrong' }, lock], [pkg, { ...lock, version: 'wrong' }], [pkg, { ...lock, packages: { '': { version: 'wrong' } } }]]) {
    assert.throws(() => releaseMetadata(`v${version}`, p, l, changelog), /must match/)
  }
})
test('rejects an embedded backend version that does not match the frontend and tag', () => {
  assert.throws(() => releaseMetadata(`v${version}`, pkg, lock, changelog, '0.2.0-alpha.5'), /embedded backend version must match/)
})
for (const notes of ['', '## [0.2.0-alpha.20] - 2026-10-22\nOther version.', `## [${version}]\nNo date.`, `## [${version}] - 2026-10-22\n### Empty`, `${changelog}\n## [${version}] - 2026-10-22\nDuplicate.`]) {
  test(`rejects missing, empty or duplicate changelog (${notes.length})`, () => assert.throws(() => releaseMetadata(`v${version}`, pkg, lock, notes), /CHANGELOG|changelog/))
}
