<script setup lang="ts">
const route = useRoute()
const hash = route.params.hash as string

const { data: result, error } = await useAsyncData(`dscan-${hash}`, () =>
    apiFetch(`/api/tools/dscan/${hash}`)
)

useHead({ title: 'D-Scan Result' })
useSeoMeta({
    description: 'Shared D-Scan analysis from EVE Online — fleet composition breakdown.',
    ogTitle: 'D-Scan Result — EVE-KILL',
})
</script>

<template>
    <div>
        <div class="flex items-center justify-between mb-4">
            <h1 class="text-xl font-bold text-white">D-Scan Result</h1>
            <NuxtLink to="/tools/dscan" class="text-xs text-blue-400/60 hover:text-blue-400 hover:underline">
                New Scan
            </NuxtLink>
        </div>

        <div v-if="error" class="glass-panel p-6 text-center">
            <p class="text-gray-500 text-sm">Scan not found.</p>
            <NuxtLink to="/tools/dscan" class="text-xs text-blue-400/60 hover:text-blue-400 hover:underline mt-2 inline-block">
                Create a new scan
            </NuxtLink>
        </div>

        <ToolsDscanResult v-else-if="result" :result="result as any" />
    </div>
</template>
