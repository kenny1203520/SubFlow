<script setup lang="ts">const props=defineProps<{amount:number;currency?:string}>();const value=new Intl.NumberFormat('zh-TW',{style:'currency',currency:props.currency??'TWD',maximumFractionDigits:props.currency==='JPY'?0:2}).format(props.amount/100)</script>
<template><span>{{ value }}</span></template>
