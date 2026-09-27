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
    editing.value = false
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
                    :description="`${formatEveDateTime(report.start_time, true)} to ${formatEveDateTime(report.end_time, true)}`"
                />
                <div class="flex flex-wrap gap-2">
                    <button type="button" class="rounded-lg border border-white/10 px-3 py-2 text-sm text-blue-300 hover:bg-white/5" @click="copyLink">
                        <Icon :name="copied ? 'lucide:check' : 'lucide:link'" class="mr-1" /> {{ copied ? 'Copied' : 'Copy link' }}
                    </button>
                    <button type="button" class="rounded-lg border border-white/10 px-3 py-2 text-sm text-blue-300 hover:bg-white/5" @click="startEdit">
                        <Icon name="lucide:pencil" class="mr-1" /> Edit report
                    </button>
                    <NuxtLink to="/tools/roam-report" class="rounded-lg border border-white/10 px-3 py-2 text-sm text-gray-300 hover:bg-white/5">New report</NuxtLink>
                </div>
            </div>

            <div v-if="report.truncated" role="status" class="mb-5 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-200">
                This report reached the 2,000 killmail limit. Totals cover the first 2,000 matching killmails in the window.
            </div>
            <div v-if="report.unresolved.length" class="mb-5 rounded-lg border border-amber-500/20 bg-amber-500/[0.06] p-4 text-sm text-amber-200">
                {{ report.unresolved.length }} {{ report.unresolved.length === 1 ? 'name was' : 'names were' }} not found: {{ report.unresolved.join(', ') }}
            </div>
            <RoamEdit v-if="editing" id="roam-edit" class="mb-7 scroll-mt-6" :report="report" @saved="saved" @cancel="editing = false" />

            <div class="mb-7 grid grid-cols-2 gap-3 md:grid-cols-4 xl:grid-cols-7">
                <div v-for="metric in metrics" :key="metric.label" class="glass-panel min-w-0 p-4">
                    <Icon :name="metric.icon" :class="metric.tone" class="text-xl" />
                    <p class="mt-3 truncate text-xl font-semibold text-white tabular-nums" :title="metric.value">{{ metric.value }}</p>
                    <p class="mt-1 text-xs font-medium text-gray-300">{{ metric.label }}</p>
                    <p class="mt-1 truncate text-xs text-gray-500">{{ metric.detail }}</p>
                </div>
            </div>

            <nav aria-label="Roam Report sections" class="mb-6 flex overflow-x-auto border-b border-white/[0.08] scrollbar-hide">
                <NuxtLink v-for="tab in tabs" :key="tab.id" :to="tabPath(tab.id)"
                    class="flex shrink-0 items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors"
                    :class="activeTab === tab.id ? 'border-white text-white' : 'border-transparent text-gray-500 hover:text-blue-400'"
                    :aria-current="activeTab === tab.id ? 'page' : undefined">
                    <Icon :name="tab.icon" class="text-base" /> {{ tab.label }}
                </NuxtLink>
            </nav>

            <RoamInformation v-if="activeTab === 'information'" :report="report" />
            <RoamKills v-else-if="activeTab === 'kills'" :report="report" />
            <RoamSystems v-else-if="activeTab === 'systems'" :report="report" />
            <RoamIntel v-else-if="activeTab === 'intel'" :report="report" />
            <RoamComposition v-else-if="activeTab === 'composition'" :report="report" />
        </template>
    </div>
</template>
