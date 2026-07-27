<script setup lang="ts">
// overview + esiData are fetched by the parent admin page (shared between the
// Overview and ESI sections) and passed down as props.
defineProps<{
    overview?: any
    esiData?: any
}>()

// ── ESI Logs state ──────────────────────────────────────────────────────────
const esiLogSearch = ref('')
const esiLogPage = ref(1)

// Deep link from a user's admin page ("View in ESI logs →"), which passes the
// character it wants pre-filtered. Seeded once at setup rather than watched:
// after landing here the filter belongs to the operator, not to the URL.
const route = useRoute()
const seedEntity = () => {
    const id = Number(route.query.character_id)
    if (!id || Number.isNaN(id)) return null
    return { id, name: String(route.query.character_name ?? id), type: 'character' as const }
}
const esiLogEntity = ref<{ id: number; name: string; type: 'character' | 'corporation' } | null>(seedEntity())
const esiLogSource = ref('')
const esiLogStatus = ref('')
const esiLogEndpointType = ref('')
const esiLogHasNew = ref(false)
const esiLogAutoReload = ref(true)
const esiDetailRow = ref<any>(null)
const esiDetailOpen = computed({
    get: () => !!esiDetailRow.value,
    set: (v: boolean) => { if (!v) esiDetailRow.value = null },
})

const debouncedEsiLogSearch = refDebounced(esiLogSearch, 300)

// Derive character_id / corporation_id from selected entity
const esiLogCharacterId = computed(() => esiLogEntity.value?.type === 'character' ? String(esiLogEntity.value.id) : '')
const esiLogCorporationId = computed(() => esiLogEntity.value?.type === 'corporation' ? String(esiLogEntity.value.id) : '')

type EsiLogsData = { rows: any[]; total: number; page: number; limit: number; pages: number; sources: string[] }
const esiLogHasNewParam = computed(() => esiLogHasNew.value ? '1' : '')
const { data: esiLogsData, refresh: refreshEsiLogs } = useApiFetch<EsiLogsData>('/api/admin/esi-logs', {
    query: {
        search: debouncedEsiLogSearch,
        page: esiLogPage,
        character_id: esiLogCharacterId,
        corporation_id: esiLogCorporationId,
        source: esiLogSource,
        status: esiLogStatus,
        endpoint_type: esiLogEndpointType,
        has_new: esiLogHasNewParam,
        limit: 50,
    },
    immediate: true,
    lazy: true,
    watch: [esiLogPage, esiLogCharacterId, esiLogCorporationId, esiLogSource, esiLogStatus, esiLogEndpointType, esiLogHasNewParam, debouncedEsiLogSearch],
})

// Reset page when filters change
watch([debouncedEsiLogSearch, esiLogEntity, esiLogSource, esiLogStatus, esiLogEndpointType, esiLogHasNew], () => {
    esiLogPage.value = 1
})

// Live tail — starts/stops with the tab, shared with the settings ESI section.
useEsiLogPoll({
    endpoint: '/api/admin/esi-logs',
    data: esiLogsData,
    page: esiLogPage,
    enabled: esiLogAutoReload,
    params: () => ({
        character_id: esiLogCharacterId.value,
        corporation_id: esiLogCorporationId.value,
        search: debouncedEsiLogSearch.value,
        source: esiLogSource.value,
        status: esiLogStatus.value,
        endpoint_type: esiLogEndpointType.value,
        has_new: esiLogHasNewParam.value,
    }),
})

// The parent keeps this component alive (<KeepAlive>) across tab switches.
// Matching the old tab-switch watcher: on re-activation, refetch the log only
// if no data ever loaded (e.g. the first fetch errored).
let activatedOnce = false
onActivated(() => {
    if (!activatedOnce) { activatedOnce = true; return }
    if (!esiLogsData.value) refreshEsiLogs()
})
</script>

<template>
    <!-- ═══════════════ ESI MONITORING ═══════════════ -->
    <div class="space-y-4">
        <!-- Summary cards -->
        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Requests (1h)</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ fmt(esiData?.rateLimit?.request_count) }}</div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Requests (24h)</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ fmt(overview?.esi?.requests24h) }}</div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Errors (24h)</div>
                <div class="text-2xl font-bold tabular-nums" :class="(overview?.esi?.errors24h ?? 0) > 100 ? 'text-red-400' : 'text-white'">
                    {{ fmt(overview?.esi?.errors24h) }}
                </div>
                <div class="text-xs text-gray-500 mt-1">{{ overview?.esi?.errorRate }}%</div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Avg Response</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ esiData?.responseTime?.avg_ms ?? '—' }}<span class="text-sm text-gray-500 ml-0.5">ms</span></div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">P95 Response</div>
                <div class="text-2xl font-bold tabular-nums" :class="(esiData?.responseTime?.p95_ms ?? 0) > 3000 ? 'text-yellow-400' : 'text-white'">
                    {{ esiData?.responseTime?.p95_ms ?? '—' }}<span class="text-sm text-gray-500 ml-0.5">ms</span>
                </div>
            </div>
        </div>

        <!-- Volume chart -->
        <EsiVolumeChart :hours="esiData?.volumeByHour ?? []" show-new-items />

        <!-- ── Request Log ── -->
        <!-- Filter bar is outside overflow-hidden so dropdowns aren't clipped -->
        <div class="rounded-t-lg bg-white/[0.04] border border-b-0 border-white/[0.08] px-5 pt-5 pb-3 flex flex-col md:flex-row md:items-center gap-3 relative z-10">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 flex-shrink-0">Request Log</h2>

            <div class="flex-1 flex flex-wrap items-center gap-2">
                    <div class="flex items-center gap-2 bg-white/[0.04] rounded-lg px-3 py-1.5 flex-1 min-w-[160px]">
                        <Icon name="lucide:search" class="text-gray-600 text-sm" />
                        <input
                            v-model="esiLogSearch"
                            type="text"
                            placeholder="Filter endpoint..."
                            class="bg-transparent text-xs text-white placeholder-gray-600 outline-none w-full"
                        >
                    </div>

                    <EntitySearchSelect v-model="esiLogEntity" placeholder="Character / Corp..." />

                    <SelectMenu
                        v-model="esiLogEndpointType"
                        placeholder="All endpoints"
                        :options="[
                            { value: '', label: 'All endpoints' },
                            { value: 'character', label: 'Character' },
                            { value: 'corporation', label: 'Corporation' },
                        ]"
                    />

                    <SelectMenu
                        v-model="esiLogSource"
                        placeholder="All sources"
                        :options="[
                            { value: '', label: 'All sources' },
                            ...(esiLogsData?.sources?.map(s => ({ value: s, label: s })) ?? []),
                        ]"
                    />

                    <SelectMenu
                        v-model="esiLogStatus"
                        placeholder="All status"
                        :options="[
                            { value: '', label: 'All status' },
                            { value: 'success', label: 'Success' },
                            { value: 'error', label: 'Errors' },
                        ]"
                    />

                    <button
                        class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer border"
                        :class="esiLogHasNew
                            ? 'bg-blue-500/15 text-blue-400 border-blue-500/20'
                            : 'bg-white/[0.04] text-gray-400 border-white/[0.06] hover:text-blue-400'"
                        @click="esiLogHasNew = !esiLogHasNew"
                        v-tooltip="esiLogHasNew ? 'Showing only requests with new data' : 'Show only requests with new data'"
                    >
                        <Icon name="lucide:sparkles" class="text-xs" />
                        Has new
                    </button>
                </div>

                <div class="flex items-center gap-3 flex-shrink-0">
                    <button
                        v-if="esiLogSearch || esiLogEntity || esiLogSource || esiLogStatus || esiLogEndpointType || esiLogHasNew"
                        class="flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors cursor-pointer"
                        @click="esiLogSearch = ''; esiLogEntity = null; esiLogSource = ''; esiLogStatus = ''; esiLogEndpointType = ''; esiLogHasNew = false; esiLogPage = 1"
                    >
                        <Icon name="lucide:x" class="text-xs" />
                        Clear
                    </button>
                    <span v-if="esiLogsData" class="text-xs text-gray-600 tabular-nums">{{ fmt(esiLogsData.total) }} rows</span>
                    <button
                        class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer"
                        :class="esiLogAutoReload && esiLogPage === 1
                            ? 'bg-green-500/15 text-green-400 border border-green-500/20'
                            : 'bg-white/[0.04] text-gray-500 border border-white/[0.06]'"
                        @click="esiLogAutoReload = !esiLogAutoReload"
                        v-tooltip="esiLogPage !== 1 ? 'Auto-reload paused (not on page 1)' : esiLogAutoReload ? 'Auto-reload ON' : 'Auto-reload OFF'"
                    >
                        <Icon :name="esiLogAutoReload ? 'lucide:play' : 'lucide:pause'" class="text-xs" />
                        Live
                    </button>
                </div>
        </div>
        <div class="rounded-b-lg bg-white/[0.04] border border-t-0 border-white/[0.08] overflow-hidden">
            <EsiLogTable :rows="esiLogsData?.rows ?? []" show-entity @select="esiDetailRow = $event" />
        </div>

        <!-- Pagination -->
        <div v-if="esiLogsData && esiLogsData.pages > 1" class="flex items-center justify-between">
            <button
                class="px-3 py-1.5 rounded text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                :disabled="esiLogPage <= 1"
                @click="esiLogPage--"
            >Previous</button>
            <span class="text-xs text-gray-500 tabular-nums">Page {{ esiLogPage }} of {{ fmt(esiLogsData.pages) }}</span>
            <button
                class="px-3 py-1.5 rounded text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                :disabled="esiLogPage >= esiLogsData.pages"
                @click="esiLogPage++"
            >Next</button>
        </div>

        <EsiLogDetail v-model="esiDetailOpen" :row="esiDetailRow" show-entity />
    </div>
</template>
