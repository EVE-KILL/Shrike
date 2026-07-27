<script setup lang="ts">
const route = useRoute()
const id = route.params.id
const tab = route.params.tab as string | undefined

await navigateTo(tab ? `/item/${id}/${tab}` : `/item/${id}`, { replace: true, redirectCode: 301 })
</script>

<template>
    <div></div>
</template>
