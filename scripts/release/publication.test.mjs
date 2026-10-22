import test from 'node:test'
import assert from 'node:assert/strict'
import { validatePublication } from './publication.mjs'

const digest = `sha256:${'a'.repeat(64)}`
const stable = `sha256:${'b'.repeat(64)}`
const metadata = { tag: 'v0.2.0-alpha.2', version: '0.2.0-alpha.2', prerelease: true, notes: '- Fixed publishing.' }
const image = 'ghcr.io/kenny1203520/subflow'
const release = { tag_name: metadata.tag, draft: false, prerelease: true, body: `${metadata.notes}\n${image}:${metadata.version}\n${digest}` }
const check = (manifest = { digest }, published = release, actualDigest = digest, before = stable, after = stable) => validatePublication(metadata, manifest, published, image, actualDigest, before, after)
test('accepts a matching image, changelog and prerelease without changing latest', () => check())
test('accepts a first alpha without any stable latest', () => check({ digest }, release, digest, null, null))
for (const [name, change] of [
  ['wrong tag', { tag_name: 'v0.2.0-alpha.1' }],
  ['draft only', { draft: true }],
  ['incorrect prerelease flag', { prerelease: false }],
  ['missing changelog', { body: `${image}:${metadata.version}\n${digest}` }],
  ['missing image', { body: `${metadata.notes}\n${digest}` }],
  ['missing digest', { body: `${metadata.notes}\n${image}:${metadata.version}` }],
]) test(`rejects ${name}`, () => assert.throws(() => check({ digest }, { ...release, ...change })))
test('rejects a registry tag that resolves to another digest', () => assert.throws(() => check({ digest: stable }), /published digest/))
test('rejects a missing build digest', () => assert.throws(() => check({ digest }, release, ''), /registry digest/))
test('rejects changing or creating latest for an alpha', () => {
  assert.throws(() => check({ digest }, release, digest, stable, digest), /latest/)
  assert.throws(() => check({ digest }, release, digest, null, digest), /latest/)
})
