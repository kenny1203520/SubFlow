<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from '../i18n'
import BaseDropdown from './BaseDropdown.vue'
import { timezoneOffset, timezoneLabel } from '../timezone'

type TimeZone = { name: string; displayName: string; offset: string; offsetMinutes: number }
const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { locale, t } = useI18n(); const query = ref(''); const open = ref(false); const searchInput = ref<HTMLInputElement | null>(null)
function getDisplayName(name: string) { return new Intl.DateTimeFormat(locale.value, { timeZone: name, timeZoneName: 'long' }).formatToParts(new Date()).find(value => value.type === 'timeZoneName')?.value || name.replaceAll('_', ' ') }
const allZones = computed<TimeZone[]>(() => ['UTC', ...Intl.supportedValuesOf('timeZone').filter(name => name !== 'UTC')].map(name => { const offset = timezoneOffset(name); return { name, displayName: getDisplayName(name), offset: offset.label, offsetMinutes: offset.minutes } }).sort((a, b) => a.offsetMinutes - b.offsetMinutes || a.displayName.localeCompare(b.displayName, locale.value)))
const selectedZone = computed(() => allZones.value.find(zone => zone.name === props.modelValue))
const matchingZones = computed(() => { const term = query.value.trim().toLocaleLowerCase(locale.value); return allZones.value.filter(zone => !term || `${zone.name} ${zone.displayName} ${zone.offset}`.toLocaleLowerCase(locale.value).includes(term)) })
const groups = computed(() => matchingZones.value.reduce<Record<string, TimeZone[]>>((all, zone) => { (all[zone.offset] ||= []).push(zone); return all }, {}))
const label = computed(() => props.modelValue ? timezoneLabel(props.modelValue) : t.value.timezone)
function select(value: string, close: () => void) { emit('update:modelValue', value); query.value = ''; close() }
function focusSearch() { nextTick(() => searchInput.value?.focus()) }
</script>
<template>
    <BaseDropdown v-model="open" class="timezone-select" :panel-label="t.timezone" @opened="focusSearch"><template
            #trigger="{ open: isOpen, toggle }"><button class="timezone-trigger" type="button" :aria-expanded="isOpen"
                @click="toggle"><span>{{ label }}</span><svg viewBox="0 0 24 24" aria-hidden="true">
                    <path d="m7 10 5 5 5-5" />
                </svg></button></template><template #default="{ close }">
            <div class="timezone-dropdown">
                <div class="timezone-search"><svg viewBox="0 0 24 24" aria-hidden="true">
                        <circle cx="11" cy="11" r="6" />
                        <path d="m16 16 4 4" />
                    </svg><input ref="searchInput" v-model="query" :placeholder="t.searchTimezone"></div>
                <div v-if="!query && selectedZone" class="timezone-current"><small>{{ t.currentTimezone }}</small><button
                        type="button" class="timezone-option selected" role="option" aria-selected="true"
                        @click="select(selectedZone.name, close)"><span><strong>{{ selectedZone.displayName }}</strong><em>{{ selectedZone.name }}
                                · {{ selectedZone.offset }}</em></span><svg viewBox="0 0 24 24" aria-hidden="true">
                            <path d="m5 12 4.2 4.2L19 6.8" />
                        </svg></button></div>
                <p v-if="!matchingZones.length" class="timezone-empty">{{ t.noTimezone }}</p>
                <div v-for="(items, offset) in groups" :key="offset" class="timezone-group">
                    <small>{{ offset }}</small><button v-for="zone in items" :key="zone.name" class="timezone-option"
                        :class="{ selected: zone.name === modelValue }" type="button" role="option"
                        :aria-selected="zone.name === modelValue"
                        @click="select(zone.name, close)"><span><strong>{{ zone.displayName }}</strong><em>{{ zone.name }}</em></span><svg
                            v-if="zone.name === modelValue" viewBox="0 0 24 24" aria-hidden="true">
                            <path d="m5 12 4.2 4.2L19 6.8" />
                        </svg></button></div>
            </div>
        </template>
    </BaseDropdown>
</template>
