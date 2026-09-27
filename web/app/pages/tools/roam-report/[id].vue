<script setup lang="ts">
import type { BattleMapSystem } from '~/utils/map/battles'
import type { RoamEngagement, RoamReport } from '~/utils/roamReport'

const route = useRoute()
const reportId = String(route.params.id ?? '')
const { data: report, pending, error } = await useApiFetch<RoamReport>(`/api/tools/roam-report/${reportId}`)

useHead({ title: 'Roam Report' })
useSeoMeta({
    description: 'A fleet combat trail with recorded engagements, systems, kills, losses, and pilot contributions.',
    ogTitle: 'Roam Report — EVE-KILL',
})

const copied = ref(false)
const selectedSystem = ref<number | null>(null)
const mapScope = ref('new-eden')
const mapRegion = ref<number | null>(null)
const mapScopes = [
    { id: 'new-eden', label: 'New Eden' },
    { id: 'wormhole', label: 'Wormhole' },
    { id: 'zarzakh', label: 'Zarzakh' },
    { id: 'abyssal', label: 'Abyssal' },
    { id: 'proving', label: 'Proving' },
]
const mapSystems = computed<BattleMapSystem[]>(() => (report.value?.systems ?? []).map(system => ({
    solar_system_id: system.solar_system_id,
    solar_system_name: system.solar_system_name,
    region_id: system.region_id || null,
    region_name: system.region_name,
    battle_count: system.engagements,
    kill_count: system.kills + system.losses,
    total_isk_destroyed: system.isk_destroyed + system.isk_lost,
})))
const selectedSystemName = computed(() => report.value?.systems.find(system => system.solar_system_id === selectedSystem.value)?.solar_system_name)
const visibleEngagements = computed<RoamEngagement[]>(() => (report.value?.engagements ?? []).filter(engagement =>
    selectedSystem.value == null || engagement.solar_system_id === selectedSystem.value,
))
const busiestSystem = computed(() => [...(report.value?.systems ?? [])].sort((a, b) => (b.kills + b.losses) - (a.kills + a.losses))[0])
const mostActivePilot = computed(() => [...(report.value?.roster ?? [])].sort((a, b) =>
    (b.kill_participations + b.losses) - (a.kill_participations + a.losses),
)[0])
const errorMessage = computed(() => (error.value as any)?.data?.error || 'This Roam Report could not be loaded.')

function selectMapSystem(system: BattleMapSystem) {
    selectedSystem.value = system.solar_system_id
    nextTick(() => document.getElementById('roam-engagements')?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}

async function copyLink() {
    try {
        await navigator.clipboard.writeText(window.location.href)
        copied.value = true
        setTimeout(() => { copied.value = false }, 2500)
    } catch {
        copied.value = false
    }
}

function targetName(kill: RoamEngagement['killmails'][number]) {
    return kill.victim_name || kill.victim_corporation_name || 'Unknown target'
}
</script>

<template>
    <div class="pb-16">
        <div v-if="pending" class="glass-panel p-12 text-center text-gray-400">
            <Icon name="lucide:loader-2" class="mr-2 animate-spin" /> Loading Roam Report…
        </div>
        <div v-else-if="error || !report" role="alert" class="glass-panel p-8">
            <h1 class="text-xl font-semibold text-white">Report unavailable</h1>
            <p class="mt-2 text-sm text-gray-400">{{ errorMessage }}</p>
            <NuxtLink to="/tools/roam-report" class="mt-5 inline-block text-sm text-blue-300 hover:underline">Create a new report →</NuxtLink>
        </div>
        <template v-else>
            <div class="mb-6 flex flex-wrap items-start justify-between gap-4">
                <PageHeader
                    title="Roam Report"
                    eyebrow="Fleet combat trail"
                    icon="lucide:route"
                    :description="`${formatEveDateTime(report.start_time, true)} to ${formatEveDateTime(report.end_time, true)} · ${report.roster.length} resolved pilots`"
                />
                <div class="flex gap-2">
                    <button type="button" class="rounded-lg border border-white/10 px-3 py-2 text-sm text-blue-300 hover:bg-white/5" @click="copyLink">
                        <Icon :name="copied ? 'lucide:check' : 'lucide:link'" class="mr-1" /> {{ copied ? 'Copied' : 'Copy link' }}
                    </button>
                    <NuxtLink to="/tools/roam-report" class="rounded-lg border border-white/10 px-3 py-2 text-sm text-gray-300 hover:bg-white/5">New report</NuxtLink>
                </div>
            </div>

            <p class="mb-6 text-xs leading-5 text-gray-500">This is an inferred combat trail from public killmails, not a record of fleet membership or every jump. Affiliations shown below are current. Engagements split when the recorded fighting moves to another system or pauses for more than 30 minutes.</p>
            <div v-if="report.truncated" role="status" class="mb-6 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-200">
                This report reached the 2,000 killmail limit. Totals and engagements cover the first 2,000 matching killmails in the selected window.
            </div>
            <div v-if="report.unresolved.length" class="mb-6 rounded-lg border border-amber-500/20 bg-amber-500/[0.06] p-4 text-sm text-amber-200">
                {{ report.unresolved.length }} {{ report.unresolved.length === 1 ? 'name was' : 'names were' }} not found: {{ report.unresolved.join(', ') }}
            </div>

            <div class="mb-6 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
                <div v-for="metric in [
                    { label: 'Kills', value: formatNumber(report.summary.kills), icon: 'lucide:crosshair', tone: 'text-emerald-300' },
                    { label: 'Losses', value: formatNumber(report.summary.losses), icon: 'lucide:shield-x', tone: 'text-rose-300' },
                    { label: 'ISK destroyed', value: formatIsk(report.summary.isk_destroyed), icon: 'lucide:trending-up', tone: 'text-emerald-300' },
                    { label: 'ISK lost', value: formatIsk(report.summary.isk_lost), icon: 'lucide:trending-down', tone: 'text-rose-300' },
                    { label: 'Efficiency', value: `${report.summary.efficiency.toFixed(2)}%`, icon: 'lucide:gauge', tone: 'text-blue-300' },
                    { label: 'Engagements', value: formatNumber(report.summary.engagements), icon: 'lucide:swords', tone: 'text-amber-300' },
                ]" :key="metric.label" class="glass-panel p-4">
                    <Icon :name="metric.icon" :class="metric.tone" class="text-xl" />
                    <p class="mt-3 text-xl font-semibold text-white tabular-nums">{{ metric.value }}</p>
                    <p class="mt-1 text-xs text-gray-500">{{ metric.label }}</p>
                </div>
            </div>

            <div v-if="report.summary.engagements" class="mb-6 grid gap-4 lg:grid-cols-3">
                <div class="glass-panel p-5">
                    <p class="text-xs font-semibold uppercase tracking-wider text-blue-300">Where you fought</p>
                    <p class="mt-2 text-lg font-semibold text-white">{{ report.summary.systems }} systems · {{ report.summary.regions }} regions</p>
                    <p v-if="busiestSystem" class="mt-2 text-sm leading-6 text-gray-400">Most activity in <strong class="text-gray-200">{{ busiestSystem.solar_system_name }}</strong>: {{ busiestSystem.kills }} kills and {{ busiestSystem.losses }} losses.</p>
                </div>
                <div class="glass-panel p-5">
                    <p class="text-xs font-semibold uppercase tracking-wider text-amber-300">Who you met</p>
                    <p class="mt-2 text-lg font-semibold text-white">{{ report.summary.unique_character_targets }} named targets</p>
                    <p v-if="report.targets[0]" class="mt-2 text-sm leading-6 text-gray-400">Most frequent opposing corporation: <strong class="text-gray-200">{{ report.targets[0].corporation_name || 'Unknown corporation' }}</strong> on {{ report.targets[0].kills }} killmails.</p>
                </div>
                <div class="glass-panel p-5">
                    <p class="text-xs font-semibold uppercase tracking-wider text-emerald-300">Fleet activity</p>
                    <p class="mt-2 text-lg font-semibold text-white">{{ report.summary.active_pilots }} of {{ report.roster.length }} pilots recorded</p>
                    <p class="mt-2 text-sm leading-6 text-gray-400">{{ report.summary.coordinated_kills }} kills show two or more listed pilots. <template v-if="mostActivePilot?.engagements">{{ mostActivePilot.name }} appears in the most killmails.</template></p>
                </div>
            </div>

            <div v-if="report.systems.length" class="mb-7 grid gap-6 xl:grid-cols-[minmax(0,1.5fr)_minmax(18rem,0.5fr)]">
                <section class="glass-panel overflow-hidden p-4 sm:p-5" aria-label="Roam combat map">
                    <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
                        <div>
                            <h2 class="text-lg font-semibold text-white">Combat map</h2>
                            <p class="mt-1 text-xs text-gray-500">Markers show systems with recorded kills or losses. Select a region, then a system.</p>
                        </div>
                        <select v-model="mapScope" class="rounded-lg border border-white/10 bg-[#141414] px-3 py-2 text-sm text-gray-200" @change="mapRegion = null">
                            <option v-for="scope in mapScopes" :key="scope.id" :value="scope.id">{{ scope.label }}</option>
                        </select>
                    </div>
                    <button v-if="mapRegion" class="mb-3 text-xs text-blue-300 hover:underline" @click="mapRegion = null">← All regions</button>
                    <ClientOnly>
                        <MapPixiScopeView
                            :key="mapScope"
                            :type="mapScope"
                            base-layer="geography"
                            activity-layer="none"
                            :hours="24"
                            :show-connections="true"
                            :show-systems="true"
                            :show-labels="true"
                            :battle-systems="mapSystems"
                            :battle-region-id="mapRegion"
                            marker-noun="engagements"
                            marker-value-label="ISK involved"
                            @battle-region="mapRegion = $event"
                            @battle-system="selectMapSystem"
                        />
                    </ClientOnly>
                </section>
                <section class="glass-panel p-5">
                    <h2 class="text-lg font-semibold text-white">Systems encountered</h2>
                    <p class="mt-1 text-xs text-gray-500">First recorded combat order</p>
                    <div class="mt-4 max-h-[35rem] space-y-2 overflow-y-auto pr-1">
                        <button
                            v-for="(system, index) in report.systems"
                            :key="system.solar_system_id"
                            type="button"
                            class="flex w-full items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-white/[0.04]"
                            :class="selectedSystem === system.solar_system_id ? 'border-blue-500/40 bg-blue-500/[0.08]' : 'border-white/[0.06]'"
                            @click="selectedSystem = selectedSystem === system.solar_system_id ? null : system.solar_system_id"
                        >
                            <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-blue-500/10 text-xs font-semibold text-blue-300">{{ index + 1 }}</span>
                            <span class="min-w-0 flex-1">
                                <span class="block truncate text-sm font-medium text-gray-100">{{ system.solar_system_name }}</span>
                                <span class="block truncate text-xs text-gray-500">{{ system.region_name }} · {{ formatEveTime(system.first_seen) }} EVE</span>
                            </span>
                            <span class="text-xs tabular-nums text-gray-400">{{ system.kills }} K / {{ system.losses }} L</span>
                        </button>
                    </div>
                </section>
            </div>

            <section id="roam-engagements" class="mb-7 scroll-mt-6">
                <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
                    <div>
                        <h2 class="text-xl font-semibold text-white">Engagement timeline</h2>
                        <p class="mt-1 text-sm text-gray-400">Each stop contains the recorded killmails from that stretch of fighting.</p>
                    </div>
                    <button v-if="selectedSystem !== null" class="text-sm text-blue-300 hover:underline" @click="selectedSystem = null">Showing {{ selectedSystemName }} · clear filter</button>
                </div>
                <div v-if="!report.engagements.length" class="glass-panel p-8 text-center">
                    <Icon name="lucide:radar" class="text-3xl text-gray-500" />
                    <h3 class="mt-3 font-semibold text-white">No recorded combat in this window</h3>
                    <p class="mx-auto mt-2 max-w-xl text-sm leading-6 text-gray-400">The resolved pilots did not appear on player killmails in the selected time range. Check the names and EVE times, or widen the window.</p>
                </div>
                <div v-else class="space-y-4">
                    <article v-for="engagement in visibleEngagements" :key="engagement.number" class="glass-panel overflow-hidden">
                        <div class="flex flex-wrap items-start gap-4 border-b border-white/[0.07] p-5">
                            <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-blue-500/10 text-sm font-bold text-blue-300">{{ engagement.number }}</span>
                            <div class="min-w-0 flex-1">
                                <div class="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                                    <NuxtLink :to="`/system/${engagement.solar_system_id}`" class="text-lg font-semibold text-white hover:text-blue-300">{{ engagement.solar_system_name }}</NuxtLink>
                                    <span class="text-sm text-gray-500">{{ engagement.region_name }}</span>
                                </div>
                                <p class="mt-1 text-xs text-gray-500">{{ formatEveDateTime(engagement.start_time, true) }}<template v-if="engagement.end_time !== engagement.start_time"> – {{ formatEveTime(engagement.end_time) }} EVE</template> · {{ engagement.pilot_ids.length }} listed pilots recorded</p>
                            </div>
                            <div class="text-right text-sm tabular-nums">
                                <p><span class="text-emerald-300">{{ engagement.kills }} kills</span><span class="mx-2 text-gray-600">/</span><span class="text-rose-300">{{ engagement.losses }} losses</span></p>
                                <p class="mt-1 text-xs text-gray-500">{{ formatIsk(engagement.isk_destroyed) }} destroyed · {{ formatIsk(engagement.isk_lost) }} lost</p>
                            </div>
                        </div>
                        <details class="group px-5 py-3">
                            <summary class="cursor-pointer select-none text-sm text-blue-300">Show {{ engagement.killmails.length }} {{ engagement.killmails.length === 1 ? 'killmail' : 'killmails' }} <Icon name="lucide:chevron-down" class="ml-1 transition-transform group-open:rotate-180" /></summary>
                            <div class="mt-3 divide-y divide-white/[0.06]">
                                <div v-for="kill in engagement.killmails" :key="kill.killmail_id" class="flex flex-wrap items-center gap-x-4 gap-y-1 py-3 text-sm">
                                    <span class="w-14 shrink-0 font-mono text-xs text-gray-500">{{ formatEveTime(kill.killmail_time) }}</span>
                                    <span class="w-10 shrink-0 text-xs font-semibold" :class="kill.role === 'kill' ? 'text-emerald-300' : 'text-rose-300'">{{ kill.role === 'kill' ? 'KILL' : 'LOSS' }}</span>
                                    <NuxtLink :to="`/kill/${kill.killmail_id}`" class="min-w-0 flex-1 truncate text-gray-200 hover:text-blue-300">{{ targetName(kill) }} <span class="text-gray-500">· {{ kill.victim_ship_name || 'Unknown ship' }}</span></NuxtLink>
                                    <span class="text-xs text-gray-400">{{ formatIsk(kill.total_value) }}</span>
                                    <span v-if="kill.friendly_fire" class="text-xs text-amber-300">fleet on both sides</span>
                                </div>
                            </div>
                        </details>
                    </article>
                </div>
            </section>

            <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(20rem,0.55fr)]">
                <section class="glass-panel overflow-hidden">
                    <div class="border-b border-white/[0.07] p-5">
                        <h2 class="text-xl font-semibold text-white">Fleet roster</h2>
                        <p class="mt-1 text-sm text-gray-400">Current corporations and alliances, with each pilot's recorded activity in this window.</p>
                    </div>
                    <div class="max-h-[38rem] divide-y divide-white/[0.06] overflow-y-auto">
                        <div v-for="pilot in report.roster" :key="pilot.character_id" class="flex flex-wrap items-center gap-3 px-5 py-3">
                            <img :src="`/images/characters/${pilot.character_id}/portrait?size=64`" :alt="pilot.name" class="h-10 w-10 rounded-full bg-gray-900" loading="lazy">
                            <div class="min-w-0 flex-1">
                                <NuxtLink :to="`/character/${pilot.character_id}`" class="block truncate text-sm font-medium text-gray-100 hover:text-blue-300">{{ pilot.name }}</NuxtLink>
                                <p class="truncate text-xs text-gray-500">{{ pilot.corporation_name || 'Unknown corporation' }}<template v-if="pilot.alliance_name"> · {{ pilot.alliance_name }}</template></p>
                            </div>
                            <div class="text-right text-xs tabular-nums text-gray-400">
                                <p><span class="text-emerald-300">{{ pilot.kill_participations }} K</span> · <span class="text-rose-300">{{ pilot.losses }} L</span></p>
                                <p class="mt-1">{{ pilot.final_blows }} final blows · {{ pilot.engagements }} fights</p>
                            </div>
                        </div>
                    </div>
                </section>
                <section class="glass-panel p-5">
                    <h2 class="text-xl font-semibold text-white">Opposing corporations</h2>
                    <p class="mt-1 text-sm text-gray-400">Corporations of the ships this fleet destroyed.</p>
                    <div v-if="report.targets.length" class="mt-4 divide-y divide-white/[0.06]">
                        <div v-for="target in report.targets" :key="target.corporation_id" class="flex items-center gap-3 py-3">
                            <img :src="`/images/corporations/${target.corporation_id}/logo?size=64`" :alt="target.corporation_name" class="h-9 w-9 rounded bg-gray-900" loading="lazy">
                            <div class="min-w-0 flex-1">
                                <NuxtLink :to="`/corporation/${target.corporation_id}`" class="block truncate text-sm text-gray-100 hover:text-blue-300">{{ target.corporation_name || `Corporation ${target.corporation_id}` }}</NuxtLink>
                                <p class="truncate text-xs text-gray-500">{{ target.alliance_name || 'No alliance recorded' }}</p>
                            </div>
                            <div class="text-right text-xs text-gray-400">
                                <p>{{ target.kills }} kills</p>
                                <p>{{ formatIsk(target.isk_destroyed) }}</p>
                            </div>
                        </div>
                    </div>
                    <p v-else class="mt-4 text-sm text-gray-500">No opposing corporations recorded.</p>
                </section>
            </div>
        </template>
    </div>
</template>
