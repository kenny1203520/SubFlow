<script setup lang="ts">
const props = withDefaults(defineProps<{ icon?: string }>(), { icon: 'tag' })
const glyphs: Record<string, string> = {
  food_dining: '\u{1F37D}', transport: '\u{1F68C}', housing: '\u{1F3E0}', utilities: '\u{1F4A1}',
  shopping: '\u{1F6CD}', entertainment: '\u{1F3AC}', health: '\u{2764}', education: '\u{1F4DA}',
  travel: '\u{2708}', insurance: '\u{1F6E1}', software_digital: '\u{1F4BB}', memberships: '\u{1F465}',
  taxes_fees: '%', gifts_donations: '\u{1F381}', other: '\u{22EF}', tag: '\u{1F3F7}',
}
</script>
<template><span class="category-icon" aria-hidden="true">{{ glyphs[props.icon] || glyphs.tag }}</span></template>
