import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const fixtures = JSON.parse(await readFile(new URL('./upgrade-fixtures.json', import.meta.url), 'utf8'))

test('pins a distinct immutable container source for every released alpha.1–alpha.5 fixture', () => {
  assert.deepEqual(fixtures.map(item => item.version), ['alpha1', 'alpha2', 'alpha3', 'alpha4', 'alpha5'])
  assert.equal(new Set(fixtures.map(item => item.image)).size, 5)
  for (const fixture of fixtures) {
    assert.match(fixture.image, /^ghcr\.io\/kenny1203520\/subflow@sha256:[a-f0-9]{64}$/)
  }
})
