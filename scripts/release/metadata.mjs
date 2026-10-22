import { readFile, mkdir, writeFile, appendFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

const identifier = '(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)'
const versionPattern = new RegExp(`^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(?:-(${identifier}(?:\\.${identifier})*))?$`)

export function releaseMetadata(tag, pkg, lock, changelog) {
  if (!versionPattern.test(tag)) throw new Error('Release ref must be a v-prefixed SemVer tag without build metadata.')
  const version = tag.slice(1)
  if ([pkg.version, lock.version, lock.packages?.['']?.version].some(value => value !== version)) {
    throw new Error('Tag, package.json and both package-lock.json versions must match.')
  }
  const lines = changelog.split(/\r?\n/)
  const headings = lines.flatMap((line, index) => line.startsWith(`## [${version}]`) ? [index] : [])
  if (headings.length !== 1 || !new RegExp(`^## \\[${version.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\] - \\d{4}-\\d{2}-\\d{2}$`).test(lines[headings[0]])) {
    throw new Error('CHANGELOG must contain exactly one dated section for this version.')
  }
  const start = headings[0] + 1
  const next = lines.findIndex((line, index) => index >= start && line.startsWith('## ['))
  const notes = lines.slice(start, next < 0 ? undefined : next).join('\n').trim()
  if (!notes || !notes.split('\n').some(line => line.trim() && !line.startsWith('#'))) {
    throw new Error('The release changelog section must contain meaningful notes.')
  }
  return { tag, version, prerelease: version.includes('-'), notes }
}

export async function prepareRelease(root, tag, refType) {
  if (refType !== 'tag') throw new Error('Publishing requires an existing version tag. Select that tag when running this workflow manually.')
  const [pkg, lock, changelog] = await Promise.all([
    readFile(resolve(root, 'frontend/package.json'), 'utf8'),
    readFile(resolve(root, 'frontend/package-lock.json'), 'utf8'),
    readFile(resolve(root, 'CHANGELOG.md'), 'utf8'),
  ])
  const metadata = releaseMetadata(tag, JSON.parse(pkg), JSON.parse(lock), changelog)
  await mkdir(resolve(root, '.release'), { recursive: true })
  await writeFile(resolve(root, '.release/notes.md'), `${metadata.notes}\n`)
  await writeFile(resolve(root, '.release/metadata.json'), `${JSON.stringify(metadata, null, 2)}\n`)
  if (process.env.GITHUB_OUTPUT) await appendFile(process.env.GITHUB_OUTPUT, `version=${metadata.version}\nprerelease=${metadata.prerelease}\n`)
  return metadata
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const result = await prepareRelease(process.cwd(), process.env.RELEASE_TAG, process.env.RELEASE_REF_TYPE)
  console.log(`Validated ${result.tag}; prerelease=${result.prerelease}; changelog is present.`)
}
