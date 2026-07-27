<script setup lang="ts">
const props = defineProps<{ data: any }>()
const formatSize = (kb: number): string => {
    if (kb >= 1048576) return `${(kb / 1048576).toFixed(1)} GB`
    if (kb >= 1024) return `${(kb / 1024).toFixed(1)} MB`
    return `${kb} KB`
}

const folders = computed(() => {
    if (!props.data?.folderStats) return []
    return Object.entries(props.data.folderStats as Record<string, any>)
        .filter(([name]) => name.toLowerCase() !== 'types')
        .map(([name, s]: [string, any]) => ({ name, ...s }))
        .sort((a: any, b: any) => b.fileCount - a.fileCount)
})

const totalFiles = computed(() => folders.value.reduce((s: number, f: any) => s + f.fileCount, 0))
const totalSizeKB = computed(() => folders.value.reduce((s: number, f: any) => s + f.sizeKB, 0))
</script>

<template>
    <div v-if="data" class="glass-panel p-4">
        <div class="flex items-center justify-between mb-3">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">Image Server</div>
            <div class="flex items-center gap-3 text-xs text-gray-500">
                <span class="tabular-nums">{{ formatNumber(totalFiles) }} files</span>
                <span class="text-gray-700">&middot;</span>
                <span class="tabular-nums">{{ formatSize(totalSizeKB) }}</span>
            </div>
        </div>
        <div class="grid grid-cols-2 md:grid-cols-5 gap-2">
            <div v-for="folder in folders" :key="folder.name"
                class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400 capitalize">{{ folder.name }}</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ formatNumber(folder.fileCount) }}</div>
                <div class="text-fine text-gray-600 tabular-nums">{{ formatSize(folder.sizeKB) }}</div>
                <div v-if="folder.metadataFileCount" class="text-fine text-gray-700 tabular-nums">{{ formatNumber(folder.metadataFileCount) }} metadata</div>
            </div>
        </div>
        <div v-if="data.cacheValidation" class="mt-3 pt-3 border-t border-white/[0.06] grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
            <div>
                <div class="text-gray-500">Validation Runs</div>
                <div class="text-white tabular-nums">{{ data.cacheValidation.totalValidationRuns }}</div>
            </div>
            <div>
                <div class="text-gray-500">Images Validated</div>
                <div class="text-white tabular-nums">{{ formatNumber(data.cacheValidation.totalImagesValidated) }}</div>
            </div>
            <div>
                <div class="text-gray-500">Images Removed</div>
                <div class="text-white tabular-nums">{{ formatNumber(data.cacheValidation.totalImagesRemoved) }}</div>
            </div>
            <div>
                <div class="text-gray-500">Validation Errors</div>
                <div class="tabular-nums" :class="data.cacheValidation.validationErrors > 0 ? 'text-amber-400' : 'text-white'">{{ formatNumber(data.cacheValidation.validationErrors) }}</div>
            </div>
        </div>
    </div>
</template>
