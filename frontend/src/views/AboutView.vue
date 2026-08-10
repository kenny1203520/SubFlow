<script setup lang="ts">
import './about.css'
import { computed } from 'vue'
import { useI18n } from '../i18n'
import { appDeveloper, appVersion, changelogUrl, containerImageRef, containerImageUrl, repositoryUrl, supportUrl } from '../appInfo'

const { tr } = useI18n()
const year = new Date().getFullYear()
const links = computed(() => [
  { key: 'aboutChangelog', label: tr('aboutChangelog'), detail: tr('aboutChangelogDetail'), href: changelogUrl, icon: 'M7 4h7l4 4v12H7zM14 4v4h4M10 13h6M10 17h4' },
  { key: 'aboutSource', label: tr('aboutSource'), detail: repositoryUrl.replace('https://', ''), href: repositoryUrl, icon: 'M9 19c-4 1.5-4-2.5-6-3m12 5v-3.5c0-1 .1-1.4-.5-2 2.8-.3 5.5-1.4 5.5-6a4.6 4.6 0 0 0-1.3-3.2 4.3 4.3 0 0 0-.1-3.2s-1.1-.3-3.5 1.3a12 12 0 0 0-6.2 0C6.5 2.8 5.4 3.1 5.4 3.1a4.3 4.3 0 0 0-.1 3.2A4.6 4.6 0 0 0 4 9.5c0 4.6 2.7 5.7 5.5 6-.6.6-.6 1.2-.5 2V21' },
  { key: 'aboutImage', label: tr('aboutImage'), detail: containerImageRef, href: containerImageUrl, icon: 'M3 13h18v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2zM6 13V9h4v4M12 13V5h4v8' },
  { key: 'aboutSupport', label: tr('aboutSupport'), detail: tr('aboutSupportDetail'), href: supportUrl, icon: 'M4 5h16v14H4zM4 6l8 6 8-6' },
])
</script>

<template>
  <section class="page about-page">
    <div class="about-hero">
      <div class="about-mark">SF</div>
      <h1>{{ tr('appName') }}</h1>
      <p>{{ tr('aboutTagline') }}</p>
    </div>
    <div class="about-facts">
      <article class="card about-fact">
        <span class="about-fact-icon"><svg viewBox="0 0 24 24"><path d="m20.6 13.4-7.2 7.2a2 2 0 0 1-2.8 0l-7.2-7.2a2 2 0 0 1-.6-1.4V4a1 1 0 0 1 1-1h8a2 2 0 0 1 1.4.6l7.4 7.4a2 2 0 0 1 0 2.8Z" /><circle cx="7.5" cy="7.5" r="1.2" /></svg></span>
        <span><small>{{ tr('aboutVersion') }}</small><strong>v{{ appVersion }}</strong></span>
      </article>
      <article class="card about-fact">
        <span class="about-fact-icon"><svg viewBox="0 0 24 24"><path d="m9 18-6-6 6-6M15 6l6 6-6 6" /></svg></span>
        <span><small>{{ tr('aboutDeveloper') }}</small><strong>{{ appDeveloper }}</strong></span>
      </article>
    </div>
    <section class="card about-links">
      <h2>{{ tr('aboutResources') }}</h2>
      <a v-for="link in links" :key="link.key" class="about-link" :href="link.href" target="_blank" rel="noopener">
        <span class="about-link-icon"><svg viewBox="0 0 24 24"><path :d="link.icon" /></svg></span>
        <span class="grow"><strong>{{ link.label }}</strong><small>{{ link.detail }}</small></span>
        <span class="about-link-arrow" aria-hidden="true">›</span>
      </a>
    </section>
    <p class="about-footer">{{ tr('aboutCopyright', { year }) }}</p>
  </section>
</template>
