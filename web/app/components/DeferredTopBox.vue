<script setup lang="ts">
const props = defineProps<{ title: string; dataType: string; limit: number; apiEndpoint?: string; killType?: string; days?: number }>()

interface Entry {
    id: number
    name: string
    count: number
    type: string
    region_id?: number
    palette?: EntityPalette | null
}

// Fetch during the page's normal SSR/hydration lifecycle. A fetch inside a
// component hydrated on scroll would run after Nuxt's isHydrating flag clears,
// missing its SSR payload and replacing server-rendered rows with skeletons.
const { data } = await useApiFetch<{ entries: Entry[] }>(computed(() => props.apiEndpoint || '/api/stats'), {
    params: computed(() => ({ dataType: props.dataType, limit: props.limit, days: props.days || 7, ...(props.killType ? { type: props.killType } : {}) })),
    default: () => ({ entries: [] }),
})
</script>

<template>
    <LazyTopBox
        :title="title"
        :data-type="dataType"
        :limit="limit"
        :days="days || 7"
        :entries="data?.entries || []"
        :hydrate-on-visible="{ rootMargin: '200px' }"
    />
</template>
