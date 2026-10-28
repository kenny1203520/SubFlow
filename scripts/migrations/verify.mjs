import { createHash } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { readFile, readdir } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const gitExecutable = process.platform === 'win32' ? 'C:\\Program Files\\Git\\cmd\\git.exe' : '/usr/bin/git'
const upgradeDir = path.join(root, 'backend', 'internal', 'upgrade')
const baselineFiles = [
  'backend/internal/adapters/pocketbase/category_kinds.go',
  'backend/internal/adapters/pocketbase/schema.go',
  'backend/internal/upgrade/0001_legacy_alpha_baseline.go',
]
const numberedMigration = /^\d{4}_[a-z0-9_]+\.go$/
const checksumPlaceholder = 'sha256:BASELINE_CHECKSUM'

function digest(entries) {
  const hash = createHash('sha256')
  for (const [name, source] of entries) hash.update(`${name}\n${source.replaceAll('\r\n', '\n')}\n`)
  return `sha256:${hash.digest('hex')}`
}

const entries = await Promise.all(baselineFiles.map(async name => {
  let source = await readFile(path.join(root, name), 'utf8')
  if (name.endsWith('0001_legacy_alpha_baseline.go')) {
    source = source.replace(/LegacyBaselineChecksum\s*=\s*"sha256:[^"]+"/, `LegacyBaselineChecksum = "${checksumPlaceholder}"`)
  }
  return [name.replaceAll('\\', '/'), source]
}))
const actualChecksum = digest(entries)
const baselineSource = await readFile(path.join(root, 'backend/internal/upgrade/0001_legacy_alpha_baseline.go'), 'utf8')
const declaredChecksum = /LegacyBaselineChecksum\s*=\s*"(sha256:[a-f0-9]+)"/.exec(baselineSource)?.[1]
if (process.argv.includes('--print-checksum')) {
  console.log(actualChecksum)
  process.exit(0)
}
if (!declaredChecksum || declaredChecksum === 'sha256:replace-after-baseline-freeze' || declaredChecksum !== actualChecksum) {
  throw new Error(`Frozen alpha.1–alpha.5 baseline checksum mismatch: expected ${declaredChecksum || 'a declared checksum'}, got ${actualChecksum}. Append a migration instead of editing the baseline.`)
}

const files = (await readdir(upgradeDir)).filter(name => numberedMigration.test(name)).sort()
const ids = []
let printedMigrationChecksum = false
for (const file of files) {
  const source = await readFile(path.join(upgradeDir, file), 'utf8')
  const id = /(?:ID|LegacyBaselineID)\s*=\s*"(\d{4}_[a-z0-9_]+)"/.exec(source)?.[1]
  const checksum = /(?:Checksum|LegacyBaselineChecksum)\s*=\s*"(sha256:[a-f0-9]+)"/.exec(source)?.[1]
  if (!id || !checksum || file.slice(0, 4) !== id.slice(0, 4)) throw new Error(`Migration ${file} must declare a matching ordered ID and static SHA-256 checksum.`)
  if (id !== '0001_legacy_alpha_baseline') {
    const canonicalSource = source.replace(/Checksum\s*=\s*"sha256:[^"]+"/, `Checksum = "${checksumPlaceholder}"`)
    const actualMigrationChecksum = digest([[`backend/internal/upgrade/${file}`, canonicalSource]])
    if (process.argv.includes(`--print-migration-checksum=${file}`)) {
      console.log(actualMigrationChecksum)
      printedMigrationChecksum = true
      break
    }
    if (checksum !== actualMigrationChecksum) throw new Error(`Migration ${id} checksum mismatch: expected ${checksum}, got ${actualMigrationChecksum}.`)
  }
  ids.push(id)
}
if (process.argv.some(value => value.startsWith('--print-migration-checksum=')) && !printedMigrationChecksum) {
  throw new Error('Requested migration checksum file was not found.')
}
if (printedMigrationChecksum) process.exit(0)
if (new Set(ids).size !== ids.length || ids.some((id, index) => Number(id.slice(0, 4)) !== index + 1)) {
  throw new Error('Migration IDs must be unique and contiguous from 0001.')
}
const registry = await readFile(path.join(upgradeDir, 'migrations.go'), 'utf8')
for (const id of ids) {
  const suffix = id.replace(/^\d{4}_/, '')
  const exportedFactory = suffix.split('_').map(value => value[0].toUpperCase() + value.slice(1)).join('')
  const factory = `${exportedFactory[0].toLowerCase()}${exportedFactory.slice(1)}(`
  if (id === '0001_legacy_alpha_baseline' && !registry.includes('legacyAlphaBaseline(')) throw new Error(`Migration ${id} is missing from the migration registry.`)
  if (id !== '0001_legacy_alpha_baseline' && !registry.includes(factory)) throw new Error(`Migration ${id} is missing from the migration registry.`)
}

const base = process.env.MIGRATION_BASE
if (base) {
  let changed
  try {
    changed = execFileSync(gitExecutable, ['diff', '--name-only', `${base}..HEAD`], { cwd: root, encoding: 'utf8' }).trim().split(/\r?\n/).filter(Boolean)
  } catch {
    throw new Error(`Could not compare migration history with base revision ${base}.`)
  }
  const protectedSchema = ['backend/internal/adapters/pocketbase/schema.go', 'backend/internal/adapters/pocketbase/category_kinds.go']
  const schemaChanged = protectedSchema.some(file => changed.includes(file))
  const newMigrations = changed.filter(file => file.startsWith('backend/internal/upgrade/') && numberedMigration.test(path.basename(file)))
  if (schemaChanged && newMigrations.length === 0) throw new Error('PocketBase schema sources changed without an appended migration.')
  for (const file of changed.filter(value => value.startsWith('backend/internal/upgrade/') && numberedMigration.test(path.basename(value)) && !value.endsWith('0001_legacy_alpha_baseline.go'))) {
    try {
      execFileSync(gitExecutable, ['cat-file', '-e', `${base}:${file}`], { cwd: root, stdio: 'ignore' })
      const previous = execFileSync(gitExecutable, ['show', `${base}:${file}`], { cwd: root, encoding: 'utf8' })
      const current = await readFile(path.join(root, file), 'utf8')
      if (previous.replaceAll('\r\n', '\n') !== current.replaceAll('\r\n', '\n')) throw new Error(`Released migration source changed: ${file}`)
    } catch (error) {
      if (error instanceof Error && error.message.startsWith('Released migration source changed:')) throw error
      // A migration absent from the base is a new append and is allowed.
    }
  }
}

console.log(`Verified ${ids.length} immutable PocketBase migration(s).`)
