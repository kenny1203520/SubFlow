import { computed, ref, watch } from 'vue'
export type ThemePreference = 'light' | 'dark' | 'system'
const saved = typeof localStorage === 'undefined' ? null : localStorage.getItem('subflow-theme')
const preference = ref<ThemePreference>(saved === 'light' || saved === 'dark' ? saved : 'system')
const media = typeof window === 'undefined' ? null : window.matchMedia('(prefers-color-scheme: dark)')
const resolved = computed<'light'|'dark'>(() => preference.value === 'system' ? (media?.matches ? 'dark' : 'light') : preference.value)
function apply() { document.documentElement.dataset.theme = resolved.value; document.documentElement.style.colorScheme = resolved.value }
watch([preference, resolved], apply, { immediate: true })
media?.addEventListener('change', () => { if (preference.value === 'system') apply() })
export function useTheme() { function setTheme(next: ThemePreference) { preference.value = next; localStorage.setItem('subflow-theme', next) }; function toggle() { setTheme(resolved.value === 'dark' ? 'light' : 'dark') }; return { preference, resolved, setTheme, toggle } }
