<script setup lang="ts">defineProps<{ title: string; description: string }>()</script>
<template>
    <div class="empty">
        <div class="empty-mark">◇</div>
        <h3>{{ title }}</h3>
        <p>{{ description }}</p>
    </div>
</template>
