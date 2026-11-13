import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFile } from 'node:fs/promises'

const fixtures = JSON.parse(await readFile(new URL('./upgrade-fixtures.json', import.meta.url), 'utf8'))
const candidate = process.argv[2] || 'subflow:candidate'
assert.deepEqual(fixtures.map(fixture => fixture.version), ['alpha1', 'alpha2', 'alpha3', 'alpha4', 'alpha5', 'beta', 'beta1'])
for (const fixture of fixtures) {
  assert.match(fixture.image, /^ghcr\.io\/kenny1203520\/subflow@sha256:[a-f0-9]{64}$/)
  console.log(`\nRehearsing ${fixture.version} → ${candidate}`)
  execFileSync(process.execPath, ['scripts/release/upgrade-smoke.mjs', fixture.image, candidate, fixture.version], { stdio: 'inherit' })
}

