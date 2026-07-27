<script setup lang="ts">
defineProps<{
    content: string
}>()
</script>

<template>
    <div class="glass-panel p-4">
        <p class="text-sm text-gray-300 whitespace-pre-wrap" v-text="content"></p>
    </div>
</template>
