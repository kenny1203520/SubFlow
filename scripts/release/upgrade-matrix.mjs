import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFile } from 'node:fs/promises'

const fixtures = JSON.parse(await readFile(new URL('./upgrade-fixtures.json', import.meta.url), 'utf8'))
const candidate = process.argv[2] || 'subflow:candidate'
assert.equal(fixtures.length, 5, 'The released alpha.1–alpha.5 upgrade matrix must stay complete.')
for (const [index, fixture] of fixtures.entries()) {
  assert.equal(fixture.version, `alpha${index + 1}`, 'Historical fixture ordering is immutable.')
  assert.match(fixture.image, /^ghcr\.io\/kenny1203520\/subflow@sha256:[a-f0-9]{64}$/)
  console.log(`\nRehearsing ${fixture.version} → ${candidate}`)
  execFileSync(process.execPath, ['scripts/release/upgrade-smoke.mjs', fixture.image, candidate, fixture.version], { stdio: 'inherit' })
}

