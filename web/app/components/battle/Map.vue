<script setup lang="ts">
import type { BattleMapSystem } from '~/utils/map/battles'
const props = defineProps<{ filters: Record<string, any> }>()
const emit = defineEmits<{ selectSystem: [system: BattleMapSystem] }>()
const params = computed(() => ({ ...props.filters, page: undefined, limit: undefined, map: 'true' }))
const { data, pending, error, refresh } = useApiFetch<{ systems?: BattleMapSystem[] }>('/api/conflicts/battles', { params })
const systems = computed(() => pending.value || error.value ? [] : data.value?.systems ?? [])
const total = computed(() => systems.value.reduce((sum, system) => sum + system.battle_count, 0))
const regionId = ref<number | null>(null)
const scope = ref('new-eden')
const scopes = [
    { id: 'new-eden', label: 'New Eden' }, { id: 'wormhole', label: 'Wormhole' },
    { id: 'zarzakh', label: 'Zarzakh' }, { id: 'abyssal', label: 'Abyssal' }, { id: 'proving', label: 'Proving' },
]
const regionName = computed(() => systems.value.find(system => system.region_id === regionId.value)?.region_name ?? 'Region')
watch(scope, () => { regionId.value = null })
watch(() => props.filters, () => { regionId.value = null })
</script>

<template>
    <section aria-label="Battle map">
        <div class="mb-3 flex flex-wrap items-center gap-2">
            <label class="flex items-center gap-2 text-xs text-gray-400">Space
                <select v-model="scope" class="rounded-md border border-white/10 bg-[#141414] px-3 py-2 text-gray-200">
                    <option v-for="item in scopes" :key="item.id" :value="item.id">{{ item.label }}</option>
                </select>
            </label>
            <button v-if="regionId" class="rounded-md border border-white/10 px-3 py-2 text-xs text-blue-300 hover:bg-white/5" @click="regionId = null">All regions</button>
            <span v-if="regionId" class="text-sm text-gray-200">{{ regionName }}</span>
            <span class="ml-auto text-xs text-gray-400" role="status">
                <template v-if="pending">Loading matching battles…</template>
                <template v-else-if="!error">{{ formatNumber(total) }} matching battles across all space · {{ formatNumber(systems.length) }} systems</template>
            </span>
        </div>
        <div v-if="error" role="alert" class="glass-panel mb-3 p-4 text-sm text-red-300">
            Unable to load battle markers. <button class="underline" @click="refresh()">Retry</button>
        </div>
        <p v-else-if="!pending && !total" class="glass-panel mb-3 p-4 text-sm text-gray-400">No battles match these filters.</p>
        <p class="mb-3 text-xs text-gray-500">Select a region’s swords to explore its systems, then select a system to browse its matching battles. Markers use each battle’s primary system.</p>
        <ClientOnly>
            <MapPixiScopeView :key="scope" :type="scope" base-layer="geography" activity-layer="none" :hours="24"
                :show-connections="true" :show-systems="true" :show-labels="true"
                :battle-systems="systems" :battle-region-id="regionId"
                @battle-region="regionId = $event" @battle-system="emit('selectSystem', $event)" />
        </ClientOnly>
    </section>
</template>
