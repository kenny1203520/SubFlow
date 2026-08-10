<script setup lang="ts">
import { useTheme } from '../theme'
import { useI18n } from '../i18n'
const { resolved, toggle } = useTheme(); const { t } = useI18n()
</script>
<template><button class="theme-switcher" type="button" :aria-label="resolved === 'dark' ? t.switchToLight : t.switchToDark"
        @click="toggle"><svg v-if="resolved === 'dark'" viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="12" cy="12" r="4" />
            <path
                d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
        </svg><svg v-else viewBox="0 0 24 24" aria-hidden="true">
            <path d="M20.7 15.5A8.6 8.6 0 0 1 8.5 3.3 8.6 8.6 0 1 0 20.7 15.5Z" />
        </svg></button></template>
