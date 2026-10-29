import { spawnSync } from 'node:child_process'
import { mkdirSync, mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const codeql = process.env.CODEQL_CLI || 'codeql'
const searchPath = process.env.CODEQL_SEARCH_PATH?.trim()
const resourceArgs = searchPath ? ['--ram=4096', '--threads=4'] : []
const codeScanningConfig = path.join(repositoryRoot, '.github', 'codeql', 'codeql-config.yml')
const reportDirectory = path.join(repositoryRoot, '.codeql')
mkdirSync(reportDirectory, { recursive: true })

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
    const report = path.join(reportDirectory, `${analysis.language}.sarif`)
    const createArgs = [
      'database', 'create', database,
      `--language=${analysis.language}`,
      `--source-root=${analysis.sourceRoot}`,
      `--codescanning-config=${codeScanningConfig}`,
    ]
    if (analysis.buildCommand) createArgs.push(`--command=${analysis.buildCommand}`)

    console.log(`\nCodeQL: creating ${analysis.language} database`)
    run(createArgs, { cwd: analysis.workingDirectory })
    console.log(`CodeQL: analyzing ${analysis.language}`)
    run([
      'database', 'analyze',
      ...(searchPath ? [`--search-path=${searchPath}`] : []),
      ...resourceArgs,
      database,
      analysis.pack,
      '--format=sarif-latest',
      `--output=${report}`,
    ])
    const sarif = JSON.parse(readFileSync(report, 'utf8'))
    const unresolved = sarif.runs?.flatMap(runResult => runResult.results || []).filter(result =>
      !result.suppressions?.some(suppression => suppression.status === 'accepted'),
    ) || []
    if (unresolved.length > 0) {
      const findings = unresolved.map(result => `${result.ruleId || 'unknown'}: ${result.message?.text || 'finding'}`)
      throw new Error(`CodeQL reported ${unresolved.length} unresolved ${analysis.language} finding(s):\n${findings.join('\n')}`)
    }
    console.log(`CodeQL: ${analysis.language} SARIF contains no unresolved findings (${report})`)
  }

  console.log(`\nCodeQL local analysis completed for Go and JavaScript/TypeScript. SARIF reports are retained in ${reportDirectory}.`)
} catch (error) {
  console.error(`\n${error instanceof Error ? error.message : String(error)}`)
  process.exitCode = 1
} finally {
  rmSync(temporaryRoot, { recursive: true, force: true })
}
