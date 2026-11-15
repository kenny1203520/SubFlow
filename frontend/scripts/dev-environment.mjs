import { parseEnv } from 'node:util'

const viteSettings = new Set(['SUBFLOW_DATA_DIR', 'SUBFLOW_DEV_BACKEND_HOST', 'SUBFLOW_HTTP_PORT'])

export function selectViteEnvironment(parentEnvironment, envFileContents) {
  const environment = { ...parentEnvironment }
  if (envFileContents !== undefined) {
    const fileEnvironment = parseEnv(envFileContents)
    for (const name of viteSettings) {
      if (environment[name] === undefined && fileEnvironment[name] !== undefined) {
        environment[name] = fileEnvironment[name]
      }
    }
  }

  for (const name of Object.keys(environment)) {
    if (name.startsWith('SUBFLOW_') && !viteSettings.has(name)) {
      delete environment[name]
    }
  }

  return environment
}
