<script setup lang="ts">
import type { HomepageWidgets } from '~/composables/useDomainConfig'

interface TrackedDashboardConfig {
    configured: boolean
    widgets: HomepageWidgets
}

interface TrackedSummary {
    tracker_count: number
    active_tracker_count: number
    kill_count: number
    total_value: number
    last_kill_at: string | null
    window_days: number
}

const { isAuthenticated } = useAuth()
const isDesktop = useIsDesktop()
const { data: authCheck } = await useApiFetch('/auth/me')
useState('auth-user').value = authCheck.value?.user ?? null

const {
    trackers,
    trackerLimit,
    trackersLoading,
    refreshTrackers,
} = useTrackerNotifications()

useHead({ title: 'Tracker' })
useSeoMeta({ robots: 'noindex, nofollow' })

const {
    data: dashboardConfig,
    pending: configLoading,
    refresh: refreshConfig,
} = await useApiFetch<TrackedDashboardConfig>('/api/me/tracked/config', {
    immediate: isAuthenticated.value,
})
const {
    data: summary,
    refresh: refreshSummary,
} = await useApiFetch<TrackedSummary>('/api/me/tracked/summary', {
    params: { days: 7 },
    immediate: isAuthenticated.value,
})

const customizing = ref(false)
const saving = ref(false)
const saveError = ref('')

watch(dashboardConfig, value => {
    if (value && !value.configured) customizing.value = true
}, { immediate: true })

const widgets = computed(() => dashboardConfig.value?.widgets)
const enabledWidgets = (section: 'top' | 'left' | 'right') =>
    (widgets.value?.[section] || []).filter(widget => widget.enabled)

const hasLeft = computed(() => enabledWidgets('left').length > 0)
const hasRight = computed(() => enabledWidgets('right').length > 0)
const columnGridTemplate = computed(() => {
    if (!hasLeft.value || !hasRight.value) return '1fr'
    return (widgets.value?.columnRatio || '250px_1fr').replace('_', ' ')
})

const streamTopics = computed<string[] | null>(() => {
    const topics = trackers.value
        .filter(tracker => tracker.enabled)
        .flatMap(tracker => {
            if (['character', 'corporation', 'alliance'].includes(tracker.target_type)) {
                return [`victim.${tracker.target_id}`, `attacker.${tracker.target_id}`]
            }
            return [`${tracker.target_type}.${tracker.target_id}`]
        })
    return topics.length ? [...new Set(topics)].sort() : null
})

async function saveDashboard(widgets: HomepageWidgets) {
    saving.value = true
    saveError.value = ''
    try {
        await apiFetch('/api/me/tracked/config', {
            method: 'PUT',
            body: { widgets },
        })
        await Promise.all([refreshConfig(), refreshTrackers(), refreshSummary()])
        customizing.value = false
    } catch (error: any) {
        saveError.value = error?.data?.error || error?.message || 'Could not save dashboard.'
    } finally {
        saving.value = false
    }
}

function lastActivity(value: string | null | undefined): string {
    if (!value) return 'Waiting for activity'
    return new Date(value).toLocaleString()
}
</script>

<template>
    <div>
        <PageHeader
            class="mb-4"
            title="Tracker"
            eyebrow="Your private killboard"
            icon="lucide:scan-eye"
            description="A personal EVE-KILL homepage built from the entities and locations you choose to track."
        >
            <template v-if="isAuthenticated && dashboardConfig?.configured" #actions>
                <div class="flex items-center gap-2">
                    <NuxtLink to="/trackers" class="rounded-md border border-white/10 bg-white/[0.03] px-3 py-2 text-xs font-semibold text-gray-300 hover:border-blue-400/25 hover:text-blue-300">
                        Manage trackers
                    </NuxtLink>
                    <button type="button" class="inline-flex items-center gap-1.5 rounded-md bg-blue-500/15 px-3 py-2 text-xs font-semibold text-blue-300 hover:bg-blue-500/25" @click="customizing = true">
                        <Icon name="lucide:layout-dashboard" /> Customize
                    </button>
                </div>
            </template>
        </PageHeader>

        <LoginGate v-if="!isAuthenticated" message="Log in to build your personal Tracker dashboard." />

        <div v-else-if="configLoading || (trackersLoading && !trackers.length)" class="glass-panel py-16 text-center text-sm text-gray-500">
            <Icon name="lucide:loader-2" class="mr-1 animate-spin" /> Loading your killboard…
        </div>

        <template v-else-if="dashboardConfig && widgets">
            <div v-if="saveError" class="mb-4 rounded-lg border border-red-400/20 bg-red-500/[0.08] px-4 py-3 text-sm text-red-300">
                {{ saveError }}
            </div>

            <TrackedDashboardConfigurator
                v-if="customizing"
                :widgets="widgets"
                :trackers="trackers"
                :tracker-limit="trackerLimit"
                :first-visit="!dashboardConfig.configured"
                :saving="saving"
                @save="saveDashboard"
                @cancel="customizing = false"
            />

            <template v-else>
                <div class="mb-4 grid grid-cols-2 gap-2 sm:grid-cols-4">
                    <div class="glass-panel px-4 py-3">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Tracked scope</div>
                        <div class="mt-1 text-lg font-bold tabular-nums text-white">{{ summary?.active_tracker_count ?? 0 }} <span class="text-xs font-normal text-gray-600">active</span></div>
                        <div class="text-fine text-gray-700">{{ summary?.tracker_count ?? trackers.length }} configured</div>
                    </div>
                    <div class="glass-panel px-4 py-3">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Kills recorded</div>
                        <div class="mt-1 text-lg font-bold tabular-nums text-white">{{ (summary?.kill_count ?? 0).toLocaleString() }}</div>
                        <div class="text-fine text-gray-700">Last 7 days</div>
                    </div>
                    <div class="glass-panel px-4 py-3">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Tracked value</div>
                        <div class="mt-1 text-lg font-bold tabular-nums text-isk">{{ formatIsk(summary?.total_value ?? 0) }}</div>
                        <div class="text-fine text-gray-700">Last 7 days</div>
                    </div>
                    <div class="glass-panel px-4 py-3">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Last activity</div>
                        <div class="mt-1 truncate text-sm font-semibold text-white">{{ lastActivity(summary?.last_kill_at) }}</div>
                        <div class="text-fine text-gray-700">Recorded by your trackers</div>
                    </div>
                </div>

                <div v-if="enabledWidgets('top').length" class="mb-4 space-y-4">
                    <TrackedDashboardWidget
                        v-for="(widget, index) in enabledWidgets('top')"
                        :key="`top-${widget.type}-${index}`"
                        :widget="widget"
                        :trackers="trackers"
                        stats-endpoint="/api/me/tracked/stats"
                        killlist-endpoint="/api/me/tracked/killmails"
                        :stream-topics="streamTopics"
                    />
                </div>

                <div v-if="hasLeft || hasRight" class="grid gap-4" :style="isDesktop ? { gridTemplateColumns: columnGridTemplate } : undefined">
                    <div v-if="hasLeft" class="order-2 min-w-0 space-y-0 lg:order-1">
                        <TrackedDashboardWidget
                            v-for="(widget, index) in enabledWidgets('left')"
                            :key="`left-${widget.type}-${index}`"
                            :widget="widget"
                            :trackers="trackers"
                            stats-endpoint="/api/me/tracked/stats"
                            killlist-endpoint="/api/me/tracked/killmails"
                            :stream-topics="streamTopics"
                        />
                    </div>
                    <div v-if="hasRight" class="order-1 min-w-0 space-y-4 lg:order-2">
                        <TrackedDashboardWidget
                            v-for="(widget, index) in enabledWidgets('right')"
                            :key="`right-${widget.type}-${index}`"
                            :widget="widget"
                            :trackers="trackers"
                            stats-endpoint="/api/me/tracked/stats"
                            killlist-endpoint="/api/me/tracked/killmails"
                            :stream-topics="streamTopics"
                        />
                    </div>
                </div>
            </template>
        </template>
    </div>
</template>
