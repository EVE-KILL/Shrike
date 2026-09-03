<script setup lang="ts">
const route = useRoute()
const typeId = Number(route.params.id)

if (!Number.isInteger(typeId) || typeId <= 0) {
    throw createError({ statusCode: 404, statusMessage: 'Market item not found' })
}
</script>

<template>
    <MarketItemPanel :type-id="typeId" />
</template>
