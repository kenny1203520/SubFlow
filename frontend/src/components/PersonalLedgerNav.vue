<script setup lang="ts">import { useI18n } from '../i18n';const {tr}=useI18n()</script>
<template><nav class="section-tabs" :aria-label="tr('personalLedger')"><RouterLink :to="{name:'personal-expenses'}">{{tr('expensePersonal')}}</RouterLink><RouterLink :to="{name:'personal-subscriptions'}">{{tr('subscriptionPersonal')}}</RouterLink></nav></template>
