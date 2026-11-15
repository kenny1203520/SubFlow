import { describe, expect, it } from 'vitest'
import { selectViteEnvironment } from './dev-environment.mjs'

describe('selectViteEnvironment', () => {
  it('loads only Vite connection settings and keeps process overrides', () => {
    const environment = selectViteEnvironment({
      PATH: 'tools',
      SUBFLOW_HTTP_PORT: '9090',
      SUBFLOW_SETTINGS_ENCRYPTION_KEY: 'process-secret',
    }, [
      'SUBFLOW_SETTINGS_ENCRYPTION_KEY=file-secret',
      'SUBFLOW_ENV=development',
      'SUBFLOW_DATA_DIR=backend/pb_data',
      'SUBFLOW_DEV_BACKEND_HOST=127.0.0.1',
      'SUBFLOW_HTTP_PORT=8080',
    ].join('\n'))

    expect(environment).toEqual({
      PATH: 'tools',
      SUBFLOW_DATA_DIR: 'backend/pb_data',
      SUBFLOW_DEV_BACKEND_HOST: '127.0.0.1',
      SUBFLOW_HTTP_PORT: '9090',
    })
  })

  it('keeps allowed shell settings and removes backend secrets without a file', () => {
    const environment = selectViteEnvironment({
      SUBFLOW_HTTP_PORT: '9000',
      SUBFLOW_SETTINGS_ENCRYPTION_KEY: 'process-secret',
      SUBFLOW_ENV: 'development',
    })

    expect(environment).toEqual({ SUBFLOW_HTTP_PORT: '9000' })
  })
})
