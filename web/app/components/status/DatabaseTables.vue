<script setup lang="ts">
const props = defineProps<{ data: any }>()

const topTables = computed(() => {
    if (!props.data?.database?.tables) return []
    return Object.entries(props.data.database.tables as Record<string, any>)
        .map(([name, t]: [string, any]) => ({ name, ...t }))
        .sort((a, b) => b.rows - a.rows)
        .slice(0, 15)
})
</script>

<template>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="glass-panel p-4">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Database Tables (1-8)</div>
            <table class="w-full text-sm">
                <thead>
                    <tr class="text-fine uppercase tracking-wider text-gray-500">
                        <th class="text-left pb-2 font-medium">Table</th>
                        <th class="text-right pb-2 font-medium">Rows</th>
                        <th class="text-right pb-2 font-medium">Data</th>
                        <th class="text-right pb-2 font-medium">Total</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="t in topTables.slice(0, 8)" :key="t.name" class="border-t border-white/[0.03]">
                        <td class="py-1.5 text-gray-300 truncate max-w-[180px]">{{ t.name }}</td>
                        <td class="py-1.5 text-white tabular-nums text-right font-medium">{{ formatNumber(t.rows) }}</td>
                        <td class="py-1.5 text-gray-400 text-right">{{ t.data_size }}</td>
                        <td class="py-1.5 text-gray-400 text-right">{{ t.total_size }}</td>
                    </tr>
                </tbody>
            </table>
        </div>
        <div class="glass-panel p-4">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-3">Database Tables (9-15)</div>
            <table class="w-full text-sm">
                <thead>
                    <tr class="text-fine uppercase tracking-wider text-gray-500">
                        <th class="text-left pb-2 font-medium">Table</th>
                        <th class="text-right pb-2 font-medium">Rows</th>
                        <th class="text-right pb-2 font-medium">Data</th>
                        <th class="text-right pb-2 font-medium">Total</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="t in topTables.slice(8, 15)" :key="t.name" class="border-t border-white/[0.03]">
                        <td class="py-1.5 text-gray-300 truncate max-w-[180px]">{{ t.name }}</td>
                        <td class="py-1.5 text-white tabular-nums text-right font-medium">{{ formatNumber(t.rows) }}</td>
                        <td class="py-1.5 text-gray-400 text-right">{{ t.data_size }}</td>
                        <td class="py-1.5 text-gray-400 text-right">{{ t.total_size }}</td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>
</template>
