<script setup lang="ts">
import type { MapActivityHours, MapActivityLayer, MapRenderBaseLayer } from '~/utils/map/layers'
import { isMapActivityHours, isMapActivityLayer, isMapRenderBaseLayer, isMapLayer, MAP_ACTIVITY_WINDOWS } from '~/utils/map/layers'
import { primeAIIDAudio } from '~/utils/map/aiidAudio'

useHead({ title: 'Map' })
useSeoMeta({
    description: 'Interactive map of New Eden and the rest of EVE Online — explore geography, security, activity, danger, and traffic.',
    ogTitle: 'New Eden Map',
})

const scopes = [
    { id: 'new-eden', label: 'New Eden' },
    { id: 'zarzakh', label: 'Zarzakh' },
    { id: 'wormhole', label: 'Wormhole' },
    { id: 'abyssal', label: 'Abyssal' },
    { id: 'proving', label: 'Proving' },
]
const route = useRoute()
const router = useRouter()
const scopeIds = new Set(scopes.map(scope => scope.id))
const requestedScope = typeof route.query.scope === 'string' ? route.query.scope : 'new-eden'
const activeScope = ref(scopeIds.has(requestedScope) ? requestedScope : 'new-eden')
const watchedSystemIds = ref(
    (typeof route.query.watch === 'string' ? route.query.watch.split(',') : [])
        .map(Number)
        .filter((id, index, ids) => Number.isInteger(id) && id > 0 && ids.indexOf(id) === index)
        .slice(0, 8),
)
const legacyLayer = isMapLayer(route.query.layer) ? route.query.layer : undefined
const requestedBase = isMapRenderBaseLayer(route.query.base) ? route.query.base : legacyLayer === 'security' ? 'security' : 'geography'
const specialBases = new Set<MapRenderBaseLayer>(['sovereignty', 'live', 'aiid'])
const baseLayer = ref<MapRenderBaseLayer>(activeScope.value === 'new-eden' ? requestedBase : specialBases.has(requestedBase) ? 'geography' : requestedBase)
const legacyActivityLayer: MapActivityLayer = isMapActivityLayer(legacyLayer) ? legacyLayer : 'none'
const activityLayer = ref<MapActivityLayer>(isMapActivityLayer(route.query.activity) ? route.query.activity : legacyActivityLayer)
const requestedHours = Number(route.query.hours)
const activityHours = ref<MapActivityHours>(isMapActivityHours(requestedHours) ? requestedHours : 24)
const showConnections = ref(route.query.routes !== '0')
const showSystems = ref(route.query.systems !== '0')
const showLabels = ref(route.query.labels !== '0')
const showChanges = ref(route.query.changes === '1')
const nearAlarmEnabled = ref(false)
const outerAlarmEnabled = ref(false)

function toggleAIIDAlarm(band: 'near' | 'outer') {
    const alarm = band === 'near' ? nearAlarmEnabled : outerAlarmEnabled
    alarm.value = !alarm.value
    if (alarm.value) void primeAIIDAudio()
}

watch(activeScope, (scope) => {
    if (scope !== 'new-eden' && specialBases.has(baseLayer.value)) baseLayer.value = 'geography'
})

watch(
    [activeScope, baseLayer, activityLayer, activityHours, showConnections, showSystems, showLabels, showChanges, watchedSystemIds],
    ([scope, base, activity, hours, connections, systems, labels, changes, watched]) => router.replace({ query: {
        scope, base, activity, hours,
        routes: connections ? undefined : '0',
        systems: systems ? undefined : '0',
        labels: labels ? undefined : '0',
        changes: base === 'sovereignty' && changes ? '1' : undefined,
        watch: base === 'aiid' && watched.length ? watched.join(',') : undefined,
    } }),
    { deep: true },
)
</script>

<template>
    <div>
        <div class="glass-panel mb-4 overflow-hidden">
            <MapPixiLayerControls
                v-model:base-layer="baseLayer"
                v-model:activity-layer="activityLayer"
                v-model:hours="activityHours"
                v-model:show-connections="showConnections"
                v-model:show-systems="showSystems"
                v-model:show-labels="showLabels"
                :allow-sovereignty="activeScope === 'new-eden'"
                compact
            />
            <div class="flex items-end gap-1 overflow-x-auto px-2 pt-2">
                <button v-for="scope in scopes" :key="scope.id" type="button" class="whitespace-nowrap rounded-t px-3 py-1.5 text-sm transition-colors" :class="activeScope === scope.id ? 'border border-b-transparent border-white/[0.08] bg-[#0a0a0f] text-white' : 'text-gray-400 hover:bg-white/[0.04] hover:text-white'" @click="activeScope = scope.id">{{ scope.label }}</button>
                <div class="mb-1 ml-auto flex shrink-0 items-center gap-1">
                    <span class="mr-1 text-[9px] font-bold uppercase tracking-[0.16em] text-gray-600">Window</span>
                    <button v-for="window in MAP_ACTIVITY_WINDOWS" :key="window.value" type="button" class="rounded px-2 py-1 text-[10px] transition-colors" :class="activityHours === window.value ? 'bg-white/[0.09] text-white' : 'text-gray-600 hover:bg-white/[0.04] hover:text-gray-300'" :disabled="activityLayer === 'none'" @click="activityHours = window.value">{{ window.label }}</button>
                    <span class="ml-3 mr-1 text-[9px] font-bold uppercase tracking-[0.16em] text-gray-600">Show</span>
                    <button type="button" class="rounded px-2 py-1 text-[10px] transition-colors" :class="showConnections ? 'bg-white/[0.08] text-gray-200' : 'text-gray-600'" @click="showConnections = !showConnections">Routes</button>
                    <button type="button" class="rounded px-2 py-1 text-[10px] transition-colors" :class="showSystems ? 'bg-white/[0.08] text-gray-200' : 'text-gray-600'" @click="showSystems = !showSystems">Systems</button>
                    <button type="button" class="rounded px-2 py-1 text-[10px] transition-colors" :class="showLabels ? 'bg-white/[0.08] text-gray-200' : 'text-gray-600'" @click="showLabels = !showLabels">Labels</button>
                    <template v-if="baseLayer === 'aiid'">
                        <span class="ml-2 mr-1 text-[9px] font-bold uppercase tracking-[0.16em] text-gray-600">Alarms</span>
                        <button type="button" class="flex items-center gap-1 rounded border px-2 py-1 text-[10px] transition-colors" :class="nearAlarmEnabled ? 'border-rose-400/30 bg-rose-400/10 text-rose-200' : 'border-white/[0.08] text-gray-600 hover:text-gray-300'" title="Sound an urgent alarm for kills fewer than five jumps away" @click="toggleAIIDAlarm('near')"><Icon :name="nearAlarmEnabled ? 'lucide:volume-2' : 'lucide:volume-x'" class="text-[11px]" />Near &lt;5</button>
                        <button type="button" class="flex items-center gap-1 rounded border px-2 py-1 text-[10px] transition-colors" :class="outerAlarmEnabled ? 'border-amber-400/30 bg-amber-400/10 text-amber-200' : 'border-white/[0.08] text-gray-600 hover:text-gray-300'" title="Sound a scanner alert for kills five to ten jumps away" @click="toggleAIIDAlarm('outer')"><Icon :name="outerAlarmEnabled ? 'lucide:volume-2' : 'lucide:volume-x'" class="text-[11px]" />Outer 5–10</button>
                    </template>
                    <button v-if="baseLayer === 'sovereignty'" type="button" class="ml-2 shrink-0 rounded border px-2.5 py-1 text-[10px] transition-colors" :class="showChanges ? 'border-yellow-400/30 bg-yellow-400/10 text-yellow-200' : 'border-white/[0.08] text-gray-500 hover:bg-white/[0.04] hover:text-gray-300'" @click="showChanges = !showChanges">Recent changes</button>
                </div>
            </div>
        </div>
        <ClientOnly>
            <MapPixiScopeView
                :key="`${activeScope}-${specialBases.has(baseLayer) ? baseLayer : 'map'}`"
                :type="activeScope"
                :base-layer="baseLayer"
                :activity-layer="activityLayer"
                :hours="activityHours"
                :show-connections="showConnections"
                :show-systems="showSystems"
                :show-labels="showLabels"
                :mode="baseLayer === 'sovereignty' ? 'sovereignty' : baseLayer === 'live' ? 'live' : baseLayer === 'aiid' ? 'aiid' : 'map'"
                :show-changes="showChanges"
                :watched-system-ids="watchedSystemIds"
                :near-alarm-enabled="nearAlarmEnabled"
                :outer-alarm-enabled="outerAlarmEnabled"
                @update:watched-system-ids="watchedSystemIds = $event"
            />
            <template #fallback><div class="flex h-[78vh] items-center justify-center rounded-lg border border-white/[0.08] bg-[#08090d] text-sm text-gray-500 animate-pulse">Loading map...</div></template>
        </ClientOnly>
    </div>
</template>
