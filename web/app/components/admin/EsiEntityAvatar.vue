<script setup lang="ts">
// Entity avatar for ESI request-log rows: corporation logo or character portrait.
defineProps<{
    entityType: string
    entityId: number | string
    size: number
}>()
</script>

<template>
    <img
        :src="`/images/${entityType === 'corporation' ? 'corporations' : 'characters'}/${entityId}/${entityType === 'corporation' ? 'logo' : 'portrait'}?size=${size}`"
        loading="lazy"
    >
</template>
