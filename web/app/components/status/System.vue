<script setup lang="ts">
defineProps<{ data: any }>()
</script>

<template>
    <div class="glass-panel p-4">
        <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">System</div>
        <div class="grid grid-cols-2 md:grid-cols-5 gap-2">
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Platform</div>
                <div class="text-sm text-white font-medium">{{ data.system?.platform }} ({{ data.system?.arch }})</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">CPUs</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ data.system?.cpus }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Memory</div>
                <div class="text-sm text-white font-medium">{{ data.system?.memory_total }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Free Memory</div>
                <div class="text-sm text-white font-medium">{{ data.system?.memory_free }}</div>
            </div>
            <div class="rounded-md bg-white/[0.03] border border-white/[0.06] px-3 py-2">
                <div class="text-xs text-gray-400">Load Average</div>
                <div class="text-sm text-white font-medium tabular-nums">{{ data.system?.load_average?.map((l: number) => l.toFixed(2)).join(' / ') }}</div>
            </div>
        </div>
    </div>
</template>
