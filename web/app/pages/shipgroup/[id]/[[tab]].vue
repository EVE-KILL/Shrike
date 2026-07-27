<script setup lang="ts">
const route = useRoute()
const id = Number(route.params.id)

// Redirect old shipgroup URLs to the market browser
await navigateTo('/market/ships', { replace: true, redirectCode: 301 })
</script>

<template>
    <div></div>
</template>
