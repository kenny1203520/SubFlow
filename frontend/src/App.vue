<script setup lang="ts">
import GlobalDialogs from './components/GlobalDialogs.vue';
</script>

<template>
    <router-view />
    <GlobalDialogs />
</template>

<style>
/* Global styles can go here */
body {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
}
</style>
