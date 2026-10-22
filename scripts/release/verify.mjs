import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFile, writeFile, appendFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { validatePublication } from './publication.mjs'

function run(command, args) {
  return execFileSync(process.platform === 'win32' ? 'rtk' : command, process.platform === 'win32' ? ['proxy', command, ...args] : args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim()
}
function manifest(ref, optional = false) {
  try { return JSON.parse(run('docker', ['buildx', 'imagetools', 'inspect', ref, '--format', '{{json .Manifest}}'])) }
  catch (error) {
    if (optional && /not found|manifest unknown/.test(String(error.stderr))) return null
    throw new Error(`Cannot inspect registry manifest for ${ref}.`)
  }
}
const metadata = JSON.parse(await readFile('.release/metadata.json', 'utf8'))
const image = process.env.IMAGE_NAME || 'ghcr.io/kenny1203520/subflow'
if (process.argv[2] === 'snapshot-latest') {
  await writeFile('.release/stable.json', JSON.stringify({ digest: manifest(`${image}:latest`, true)?.digest || null }))
} else {
  const digest = process.env.RELEASE_DIGEST
  assert.match(digest || '', /^sha256:[a-f0-9]{64}$/, 'build must return a registry digest')
  const remote = manifest(`${image}:${metadata.version}`)
  const repo = process.env.GITHUB_REPOSITORY || 'kenny1203520/SubFlow'
  assert.match(repo, /^[\w.-]+\/[\w.-]+$/)
  const release = JSON.parse(run('gh', ['api', `repos/${repo}/releases/tags/${metadata.tag}`]))
  let stableBefore, stableAfter
  if (metadata.prerelease) {
    const stable = JSON.parse(await readFile('.release/stable.json', 'utf8'))
    stableBefore = stable.digest
    stableAfter = manifest(`${image}:latest`, true)?.digest || null
  }
  validatePublication(metadata, remote, release, image, digest, stableBefore, stableAfter)
  const result = { tag: metadata.tag, source: process.env.GITHUB_SHA, image: `${image}:${metadata.version}`, digest, prerelease: release.prerelease, url: release.html_url, verified: true }
  await writeFile(resolve('.release/result.json'), `${JSON.stringify(result, null, 2)}\n`)
  if (process.env.GITHUB_STEP_SUMMARY) await appendFile(process.env.GITHUB_STEP_SUMMARY, `## Release verified\n\n- ${release.html_url}\n- Image: \`${result.image}\`\n- Digest: \`${digest}\`\n- Source: \`${result.source}\`\n`)
  console.log(JSON.stringify(result))
}
