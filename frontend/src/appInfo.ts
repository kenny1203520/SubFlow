// Project metadata shown on the About page. Keep the version out of here:
// frontend/package.json is the single source of truth and vite.config.ts
// injects it as VITE_APP_VERSION at build time.
export const appVersion: string = import.meta.env.VITE_APP_VERSION || '0.0.0'
export const appDeveloper = 'Kenny'

export const repositoryUrl = 'https://github.com/kenny1203520/SubFlow'
export const releasesUrl = `${repositoryUrl}/releases`
export const changelogUrl = `${repositoryUrl}/blob/develop/CHANGELOG.md`
export const supportUrl = `${repositoryUrl}/issues`

// Published by .github/workflows/release.yml on every version tag.
export const containerImageRef = 'ghcr.io/kenny1203520/subflow'
export const containerImageUrl = `${repositoryUrl}/pkgs/container/subflow`
