<script setup lang="ts">
const route = useRoute()
const hash = route.params.hash as string

const { data: result, error } = await useAsyncData(`localscan-${hash}`, () =>
    apiFetch(`/api/tools/localscan/${hash}`)
)

useHead({ title: 'Local Scan Result' })
useSeoMeta({
    description: 'Shared local scan analysis from EVE Online — pilot affiliations and threat levels.',
    ogTitle: 'Local Scan Result — EVE-KILL',
})
</script>

<template>
    <div>
        <div class="flex items-center justify-between mb-4">
            <h1 class="text-xl font-bold text-white">Local Scan Result</h1>
            <NuxtLink to="/tools/localscan" class="text-xs text-blue-400/60 hover:text-blue-400 hover:underline">
                New Scan
            </NuxtLink>
        </div>

        <div v-if="error" class="glass-panel p-6 text-center">
            <p class="text-gray-500 text-sm">Scan not found.</p>
            <NuxtLink to="/tools/localscan" class="text-xs text-blue-400/60 hover:text-blue-400 hover:underline mt-2 inline-block">
                Create a new scan
            </NuxtLink>
        </div>

        <ToolsLocalscanResult v-else-if="result" :result="result as any" />
    </div>
</template>
