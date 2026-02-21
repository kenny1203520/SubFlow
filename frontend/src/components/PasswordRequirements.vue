<template>
    <div class="mt-2 text-xs bg-slate-50/80 p-3 rounded-xl border border-slate-200 shadow-sm animate-fade-in text-left">
        <div class="flex items-center justify-between mb-2">
            <p class="font-semibold text-slate-700">{{ t('auth.requirements.title', 'Password Requirements:') }}</p>
            <span v-if="password" class="font-bold text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full"
                :class="strengthColorClass.text + ' ' + strengthColorClass.bg">
                {{ strengthLabel }}
            </span>
        </div>

        <!-- Progress Bar -->
        <div v-if="password" class="w-full bg-slate-200 rounded-full h-1.5 mb-3 overflow-hidden">
            <div class="h-1.5 rounded-full transition-all duration-300" :class="strengthColorClass.bar"
                :style="{ width: strengthPercentage + '%' }">
            </div>
        </div>

        <ul class="space-y-1.5">
            <li class="flex items-center gap-2 transition-colors duration-200"
                :class="lengthValid ? 'text-emerald-600' : 'text-slate-500'">
                <svg v-if="lengthValid" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-emerald-500"
                    viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clip-rule="evenodd" />
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-slate-300" viewBox="0 0 20 20"
                    fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                        clip-rule="evenodd" />
                </svg>
                <span>{{ t('auth.requirements.length', 'At least 8 characters') }}</span>
            </li>
            <li class="flex items-center gap-2 transition-colors duration-200"
                :class="uppercaseValid ? 'text-emerald-600' : 'text-slate-500'">
                <svg v-if="uppercaseValid" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-emerald-500"
                    viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clip-rule="evenodd" />
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-slate-300" viewBox="0 0 20 20"
                    fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                        clip-rule="evenodd" />
                </svg>
                <span>{{ t('auth.requirements.uppercase', 'One uppercase letter') }}</span>
            </li>
            <li class="flex items-center gap-2 transition-colors duration-200"
                :class="numberValid ? 'text-emerald-600' : 'text-slate-500'">
                <svg v-if="numberValid" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-emerald-500"
                    viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clip-rule="evenodd" />
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-slate-300" viewBox="0 0 20 20"
                    fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                        clip-rule="evenodd" />
                </svg>
                <span>{{ t('auth.requirements.number', 'One number') }}</span>
            </li>
            <li class="flex items-center gap-2 transition-colors duration-200"
                :class="symbolValid ? 'text-emerald-600' : 'text-slate-500'">
                <svg v-if="symbolValid" xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-emerald-500"
                    viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clip-rule="evenodd" />
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-slate-300" viewBox="0 0 20 20"
                    fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                        clip-rule="evenodd" />
                </svg>
                <span>{{ t('auth.requirements.symbol', 'One special character') }}</span>
            </li>
        </ul>
    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

const props = defineProps<{
    password: string
}>();

const { t } = useI18n();

const lengthValid = computed(() => (props.password || '').length >= 8);
const uppercaseValid = computed(() => /[A-Z]/.test(props.password || ''));
const numberValid = computed(() => /[0-9]/.test(props.password || ''));
const symbolValid = computed(() => /[^A-Za-z0-9]/.test(props.password || ''));

const score = computed(() => {
    let s = 0;
    if (lengthValid.value) s++;
    if (uppercaseValid.value) s++;
    if (numberValid.value) s++;
    if (symbolValid.value) s++;
    return s;
});

const strengthPercentage = computed(() => (score.value / 4) * 100 + 1);

const strengthLabel = computed(() => {
    if (score.value <= 1) return t('auth.strength.weak', 'Weak');
    if (score.value === 2) return t('auth.strength.fair', 'Fair');
    if (score.value === 3) return t('auth.strength.good', 'Good');
    return t('auth.strength.strong', 'Strong');
});

const strengthColorClass = computed(() => {
    if (score.value <= 1) return { text: 'text-rose-600', bg: 'bg-rose-50', bar: 'bg-rose-500' };
    if (score.value === 2) return { text: 'text-amber-500', bg: 'bg-amber-50', bar: 'bg-amber-400' };
    if (score.value === 3) return { text: 'text-blue-500', bg: 'bg-blue-50', bar: 'bg-blue-400' };
    return { text: 'text-emerald-600', bg: 'bg-emerald-50', bar: 'bg-emerald-500' };
});
</script>
