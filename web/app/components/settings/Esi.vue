<script setup lang="ts">
// Settings → ESI section. Extracted verbatim from pages/settings/[[tab]].vue.
// This component only mounts when the ESI tab is first opened (the parent
// keeps it alive afterwards via <KeepAlive>), so `immediate: true` on the
// fetches below is equivalent to the old `immediate: activeSection === 'esi'`.
import type { EsiLogRow } from '~/utils/esiLog'

defineProps<{
    /** `/auth/token-info` payload — fetched eagerly (awaited) by the settings page. */
    tokenInfo: { scopes: string[]; token_expiry: string | null } | null
}>()

const { data: esiData } = useApiFetch('/api/user/esi', {
    immediate: true,
    lazy: true,
})

// ESI logs state
const esiLogPage = ref(1)
const esiLogSource = ref('')
const esiLogStatus = ref('')
const esiLogEndpointType = ref('')
const esiLogAutoReload = ref(true)
const esiDetailRow = ref<EsiLogRow | null>(null)
const esiDetailOpen = computed({
    get: () => !!esiDetailRow.value,
    set: (v: boolean) => { if (!v) esiDetailRow.value = null },
})

type UserEsiLogsData = { rows: EsiLogRow[]; total: number; page: number; limit: number; pages: number; sources: string[] }
const { data: esiLogsData } = useApiFetch<UserEsiLogsData>('/api/user/esi-logs', {
    params: {
        page: esiLogPage,
        source: esiLogSource,
        status: esiLogStatus,
        endpoint_type: esiLogEndpointType,
    },
    immediate: true,
    lazy: true,
    watch: [esiLogPage, esiLogSource, esiLogStatus, esiLogEndpointType],
})

watch([esiLogSource, esiLogStatus, esiLogEndpointType], () => {
    esiLogPage.value = 1
})

// Live tail. Previously an inline interval that kept ticking for the life of
// the page and leaned on an `activeSection` prop to decide whether to do
// anything; the composable stops on deactivate instead, so the prop is gone.
useEsiLogPoll({
    endpoint: '/api/user/esi-logs',
    data: esiLogsData,
    page: esiLogPage,
    enabled: esiLogAutoReload,
    params: () => ({
        source: esiLogSource.value,
        status: esiLogStatus.value,
        endpoint_type: esiLogEndpointType.value,
    }),
})
</script>

<template>
    <div class="space-y-4">
        <!-- Stats cards -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div class="glass-panel p-4 text-center">
                <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">Requests (1h)</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ fmt(esiData?.rateLimit?.request_count) }}</div>
            </div>
            <div class="glass-panel p-4 text-center">
                <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">Avg Response</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ esiData?.responseTime?.avg_ms ?? '—' }}<span class="text-sm text-gray-500 ml-0.5">ms</span></div>
            </div>
            <div class="glass-panel p-4 text-center">
                <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">P95 Response</div>
                <div class="text-2xl font-bold tabular-nums" :class="(esiData?.responseTime?.p95_ms ?? 0) > 3000 ? 'text-yellow-400' : 'text-white'">
                    {{ esiData?.responseTime?.p95_ms ?? '—' }}<span class="text-sm text-gray-500 ml-0.5">ms</span>
                </div>
            </div>
            <div class="glass-panel p-4 text-center">
                <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">Token</div>
                <div class="text-base font-bold" :class="tokenInfo?.token_expiry
                    ? (new Date(tokenInfo.token_expiry) > new Date() ? 'text-green-400' : 'text-yellow-400')
                    : 'text-red-400'">
                    {{ !tokenInfo?.token_expiry ? 'None' : new Date(tokenInfo.token_expiry) > new Date() ? 'Active' : 'Refreshing' }}
                </div>
            </div>
        </div>

        <!-- Volume chart -->
        <EsiVolumeChart :hours="esiData?.volumeByHour ?? []" show-new-items />

        <!-- Scopes -->
        <div class="glass-panel p-5">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">Granted Scopes</h2>
            <div v-if="tokenInfo?.scopes?.length" class="flex flex-wrap gap-1.5">
                <span v-for="scope in tokenInfo.scopes" :key="scope"
                    class="px-2 py-0.5 text-xs rounded bg-blue-500/10 text-blue-400 border border-blue-500/20">
                    {{ scope }}
                </span>
            </div>
            <p v-else class="text-sm text-gray-500">No ESI scopes granted.</p>
        </div>

        <!-- Request log -->
        <div class="glass-panel overflow-hidden">
            <div class="p-4 border-b border-white/[0.06]">
                <div class="flex flex-wrap items-center gap-2">
                    <h3 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">ESI Request Log</h3>
                    <div class="flex-1"></div>
                    <div class="flex flex-wrap items-center gap-2">
                        <SelectMenu
                            v-model="esiLogEndpointType"
                            :options="[
                                { value: '', label: 'All types' },
                                { value: 'character', label: 'Character' },
                                { value: 'corporation', label: 'Corporation' },
                            ]"
                            placeholder="All types"
                        />
                        <SelectMenu
                            v-model="esiLogSource"
                            :options="[
                                { value: '', label: 'All sources' },
                                ...(esiLogsData?.sources?.map(s => ({ value: s, label: s })) ?? []),
                            ]"
                            placeholder="All sources"
                        />
                        <SelectMenu
                            v-model="esiLogStatus"
                            :options="[
                                { value: '', label: 'All status' },
                                { value: 'success', label: 'Success' },
                                { value: 'error', label: 'Error' },
                            ]"
                            placeholder="All status"
                        />
                        <button
                            v-if="esiLogSource || esiLogStatus || esiLogEndpointType"
                            class="text-xs text-gray-500 hover:text-blue-400 transition-colors"
                            @click="esiLogSource = ''; esiLogStatus = ''; esiLogEndpointType = ''; esiLogPage = 1"
                        >Clear</button>
                    </div>
                    <span v-if="esiLogsData" class="text-xs text-gray-600 tabular-nums">{{ fmt(esiLogsData.total) }} rows</span>
                    <button
                        class="w-6 h-6 flex items-center justify-center rounded transition-colors"
                        :class="esiLogAutoReload && esiLogPage === 1
                            ? 'text-green-400 bg-green-500/10'
                            : 'text-gray-600 hover:text-gray-400'"
                        @click="esiLogAutoReload = !esiLogAutoReload"
                        v-tooltip="esiLogPage !== 1 ? 'Auto-reload paused (not on page 1)' : esiLogAutoReload ? 'Auto-reload ON' : 'Auto-reload OFF'"
                    >
                        <Icon :name="esiLogAutoReload ? 'lucide:play' : 'lucide:pause'" class="text-xs" />
                    </button>
                </div>
            </div>

            <EsiLogTable :rows="esiLogsData?.rows ?? []" @select="esiDetailRow = $event" />
        </div>

        <!-- Pagination -->
        <div v-if="esiLogsData && esiLogsData.pages > 1" class="flex items-center justify-between">
            <button
                class="px-3 py-1 text-sm rounded bg-white/[0.06] hover:bg-blue-500/[0.1] transition-colors disabled:opacity-30"
                :disabled="esiLogPage <= 1"
                @click="esiLogPage--"
            >Prev</button>
            <span class="text-xs text-gray-500 tabular-nums">Page {{ esiLogPage }} of {{ fmt(esiLogsData.pages) }}</span>
            <button
                class="px-3 py-1 text-sm rounded bg-white/[0.06] hover:bg-blue-500/[0.1] transition-colors disabled:opacity-30"
                :disabled="esiLogPage >= esiLogsData.pages"
                @click="esiLogPage++"
            >Next</button>
        </div>

        <EsiLogDetail v-model="esiDetailOpen" :row="esiDetailRow" />
    </div>
</template>
