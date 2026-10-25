import { spawnSync } from 'node:child_process'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const codeql = process.env.CODEQL_CLI || 'codeql'

function run(args, options = {}) {
  const result = spawnSync(codeql, args, {
    cwd: repositoryRoot,
    encoding: 'utf8',
    stdio: 'inherit',
    ...options,
  })

  if (result.error) {
    if (result.error.code === 'ENOENT') {
      throw new Error('CodeQL CLI was not found. Install the CLI and put it on PATH, or set CODEQL_CLI to its executable path.')
    }
    throw result.error
  }
  if (result.status !== 0) throw new Error(`CodeQL command failed with exit code ${result.status}: ${args.join(' ')}`)
}

const temporaryRoot = mkdtempSync(path.join(tmpdir(), 'subflow-codeql-'))

try {
  run(['version'])

  const analyses = [
    {
      language: 'javascript-typescript',
      pack: 'codeql/javascript-queries',
      sourceRoot: repositoryRoot,
      workingDirectory: repositoryRoot,
    },
    {
      language: 'go',
      pack: 'codeql/go-queries',
      sourceRoot: path.join(repositoryRoot, 'backend'),
      workingDirectory: path.join(repositoryRoot, 'backend'),
      buildCommand: 'go build ./...',
    },
  ]

  for (const analysis of analyses) {
    const database = path.join(temporaryRoot, analysis.language)
    const report = path.join(temporaryRoot, `${analysis.language}.sarif`)
    const createArgs = [
      'database', 'create', database,
      `--language=${analysis.language}`,
      `--source-root=${analysis.sourceRoot}`,
    ]
    if (analysis.buildCommand) createArgs.push(`--command=${analysis.buildCommand}`)

    console.log(`\nCodeQL: creating ${analysis.language} database`)
    run(createArgs, { cwd: analysis.workingDirectory })
    console.log(`CodeQL: analyzing ${analysis.language}`)
    run(['database', 'analyze', database, analysis.pack, '--format=sarif-latest', `--output=${report}`])
  }

  console.log('\nCodeQL local analysis completed for Go and JavaScript/TypeScript.')
} catch (error) {
  console.error(`\n${error instanceof Error ? error.message : String(error)}`)
  process.exitCode = 1
} finally {
  rmSync(temporaryRoot, { recursive: true, force: true })
}
