<script setup lang="ts">
import type { RoamReport } from '~/utils/roamReport'

const route = useRoute()
const reportId = String(route.params.id ?? '')
if (!/^[0-9a-f]{32}$/.test(reportId)) {
    throw createError({ statusCode: 404, statusMessage: 'Roam Report not found' })
}
definePageMeta({ key: route => `/tools/roam-report/${route.params.id}` })

const tabs = [
    { id: 'information', label: 'Information', icon: 'lucide:layout-dashboard' },
    { id: 'kills', label: 'Kills', icon: 'lucide:crosshair' },
    { id: 'systems', label: 'Systems', icon: 'lucide:map' },
    { id: 'intel', label: 'Intel', icon: 'lucide:scan-eye' },
    { id: 'composition', label: 'Composition', icon: 'lucide:users' },
] as const
type TabId = typeof tabs[number]['id']
const tabParam = route.params.tab
if (typeof tabParam === 'string' && tabParam && !tabs.some(tab => tab.id === tabParam)) {
    throw createError({ statusCode: 404, statusMessage: 'Roam Report tab not found' })
}
const activeTab = computed<TabId>(() => {
    const tab = route.params.tab
    return typeof tab === 'string' && tabs.some(item => item.id === tab) ? tab as TabId : 'information'
})
const basePath = `/tools/roam-report/${reportId}`
const tabPath = (tab: TabId) => tab === 'information' ? basePath : `${basePath}/${tab}`
const { data: report, pending, error } = await useApiFetch<RoamReport>(`/api/tools/roam-report/${reportId}`)
const errorMessage = computed(() => (error.value as any)?.data?.error || 'This Roam Report could not be loaded.')
const copied = ref(false)
const editing = ref(false)
const reportRevision = ref(0)

useHead({ title: computed(() => activeTab.value === 'information' ? 'Roam Report' : `Roam Report — ${tabs.find(tab => tab.id === activeTab.value)?.label}`) })
useSeoMeta({
    description: 'A fleet combat report with recorded kills, systems, pilot composition and intel.',
    ogTitle: 'Roam Report — EVE-KILL',
})

const metrics = computed(() => report.value ? [
    { label: 'Systems', value: formatNumber(report.value.summary.systems), detail: `${report.value.summary.regions} regions`, icon: 'lucide:map-pin', tone: 'text-blue-300' },
    { label: 'Pilots', value: formatNumber(report.value.roster.length), detail: `${report.value.summary.active_pilots} active`, icon: 'lucide:users', tone: 'text-sky-300' },
    { label: 'Kills', value: formatNumber(report.value.summary.kills), detail: `${report.value.summary.coordinated_kills} coordinated`, icon: 'lucide:crosshair', tone: 'text-emerald-300' },
    { label: 'Losses', value: formatNumber(report.value.summary.losses), detail: `${report.value.summary.friendly_fire_losses} friendly fire`, icon: 'lucide:shield-x', tone: 'text-rose-300' },
    { label: 'ISK destroyed', value: formatIsk(report.value.summary.isk_destroyed), detail: `${report.value.summary.efficiency.toFixed(2)}% efficiency`, icon: 'lucide:trending-up', tone: 'text-emerald-300' },
    { label: 'ISK lost', value: formatIsk(report.value.summary.isk_lost), detail: 'Recorded losses', icon: 'lucide:trending-down', tone: 'text-rose-300' },
    { label: 'Engagements', value: formatNumber(report.value.summary.engagements), detail: 'Recorded fights', icon: 'lucide:swords', tone: 'text-amber-300' },
] : [])
const heroPilots = computed(() => [...(report.value?.roster ?? [])]
    .sort((a, b) => (b.kill_participations + b.losses) - (a.kill_participations + a.losses))
    .slice(0, 4))

async function copyLink() {
    try {
        await navigator.clipboard.writeText(window.location.href)
        copied.value = true
        setTimeout(() => { copied.value = false }, 2500)
    } catch {
        copied.value = false
    }
}

function startEdit() {
    editing.value = true
    nextTick(() => document.getElementById('roam-edit')?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}

function saved(updated: RoamReport) {
    report.value = updated
    reportRevision.value++
    editing.value = false
}
</script>

<template>
    <div class="pb-16">
        <EntityHeader v-if="pending" loading />
        <div v-else-if="error || !report" role="alert" class="glass-panel p-8">
            <h1 class="text-xl font-semibold text-white">Report unavailable</h1>
            <p class="mt-2 text-sm text-gray-400">{{ errorMessage }}</p>
            <NuxtLink to="/tools/roam-report" class="mt-5 inline-block text-sm text-blue-300 hover:underline">Create a new report →</NuxtLink>
        </div>
        <template v-else>
            <EntityHeader accent="#60a5fa" :background-image="report.systems[0] ? `/images/systems/${report.systems[0].solar_system_id}?size=1024` : null">
                <template #image>
                    <div class="grid h-32 w-32 shrink-0 grid-cols-2 gap-1 overflow-hidden rounded-lg border border-white/10 bg-blue-500/10 shadow-lg md:h-40 md:w-40">
                        <template v-for="pilot in heroPilots" :key="pilot.character_id">
                            <EveImage :src="`/images/characters/${pilot.character_id}/portrait?size=128`" :alt="pilot.name" class="h-full w-full object-cover" loading="eager" />
                        </template>
                        <div v-if="!report.roster.length" class="col-span-2 flex items-center justify-center"><Icon name="lucide:route" class="text-5xl text-blue-300" /></div>
                    </div>
                </template>
                <div class="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-blue-300"><Icon name="lucide:route" /> Fleet combat trail</div>
                <h1 class="text-2xl font-bold text-white md:text-3xl">Roam Report</h1>
                <div class="mt-4 flex flex-wrap items-center gap-2 text-sm text-gray-300">
                    <Icon name="lucide:calendar-range" class="text-gray-500" />
                    <span class="tabular-nums">{{ formatEveDateTime(report.start_time, true) }}</span>
                    <Icon name="lucide:arrow-right" class="text-gray-600" />
                    <span class="tabular-nums">{{ formatEveDateTime(report.end_time, true) }}</span>
                </div>
                <p class="mt-3 text-xs text-gray-400">{{ report.roster.length }} listed pilots · {{ report.summary.systems }} systems with recorded combat · {{ report.summary.engagements }} engagements</p>
                <template #right>
                    <div class="flex flex-wrap justify-center gap-2 md:max-w-36 md:flex-col md:items-stretch">
                        <button type="button" class="rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-sm text-blue-300 hover:bg-white/5" @click="copyLink"><Icon :name="copied ? 'lucide:check' : 'lucide:link'" class="mr-1" /> {{ copied ? 'Copied' : 'Copy link' }}</button>
                        <button type="button" class="rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-sm text-blue-300 hover:bg-white/5" @click="startEdit"><Icon name="lucide:pencil" class="mr-1" /> Edit report</button>
                        <NuxtLink to="/tools/roam-report" class="rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-center text-sm text-gray-300 hover:bg-white/5">New report</NuxtLink>
                    </div>
                </template>
                <template #stats>
                    <div class="grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7">
                        <div v-for="metric in metrics" :key="metric.label" class="min-w-0 rounded-lg border border-white/[0.06] bg-white/[0.03] p-3 text-center">
                            <p class="truncate text-lg font-bold tabular-nums text-white" :title="metric.value">{{ metric.value }}</p>
                            <p class="mt-1 flex items-center justify-center gap-1.5 text-fine font-medium uppercase tracking-wider text-gray-400"><Icon :name="metric.icon" :class="metric.tone" /> {{ metric.label }}</p>
                            <p class="mt-1 truncate text-fine text-gray-500">{{ metric.detail }}</p>
                        </div>
                    </div>
                </template>
            </EntityHeader>

            <div v-if="report.truncated" role="status" class="mb-5 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-200">
                This report reached the 2,000 killmail limit. Totals cover the first 2,000 matching killmails in the window.
            </div>
            <div v-if="report.unresolved.length" class="mb-5 rounded-lg border border-amber-500/20 bg-amber-500/[0.06] p-4 text-sm text-amber-200">
                {{ report.unresolved.length }} {{ report.unresolved.length === 1 ? 'name was' : 'names were' }} not found: {{ report.unresolved.join(', ') }}
            </div>
            <RoamEdit v-if="editing" id="roam-edit" class="mb-7 scroll-mt-6" :report="report" @saved="saved" @cancel="editing = false" />

            <nav aria-label="Roam Report sections" class="mb-6 flex overflow-x-auto border-b border-white/[0.08] scrollbar-hide">
                <NuxtLink v-for="tab in tabs" :key="tab.id" :to="tabPath(tab.id)"
                    class="flex shrink-0 items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors"
                    :class="activeTab === tab.id ? 'border-white text-white' : 'border-transparent text-gray-500 hover:text-blue-400'"
                    :aria-current="activeTab === tab.id ? 'page' : undefined">
                    <Icon :name="tab.icon" class="text-base" /> {{ tab.label }}
                </NuxtLink>
            </nav>

            <RoamInformation v-if="activeTab === 'information'" :report="report" />
            <RoamKills v-else-if="activeTab === 'kills'" :key="reportRevision" :report="report" />
            <RoamSystems v-else-if="activeTab === 'systems'" :report="report" />
            <RoamIntel v-else-if="activeTab === 'intel'" :report="report" />
            <RoamComposition v-else-if="activeTab === 'composition'" :report="report" />
        </template>
    </div>
</template>
