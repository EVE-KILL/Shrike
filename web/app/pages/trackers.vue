<script setup lang="ts">
import type { KilllistLossEntities } from '#shared/utils/killlistRow'
import type { EntityTracker, TrackerTargetType } from '~/composables/useTrackerNotifications'

interface PickedTarget {
    type: TrackerTargetType
    id: number
    name: string
    ticker: string | null
}

const { isAuthenticated } = useAuth()
const { data: authCheck } = await useApiFetch('/auth/me')
useState('auth-user').value = authCheck.value?.user ?? null

const {
    trackers,
    trackerLimit,
    trackersLoading,
    browserAlertsEnabled,
    browserPermission,
    refreshTrackers,
    setBrowserAlerts,
} = useTrackerNotifications()

useHead({ title: 'Entity Trackers' })
useSeoMeta({ robots: 'noindex, nofollow' })

const selectedID = ref<number | null>(null)
const creating = ref(false)
const mutatingID = ref<number | null>(null)
const errorMessage = ref('')

const selected = computed(() => trackers.value.find(tracker => tracker.id === selectedID.value) ?? null)

watch(trackers, current => {
    if (selectedID.value && current.some(tracker => tracker.id === selectedID.value)) return
    selectedID.value = current[0]?.id ?? null
}, { immediate: true })

const pickedAlready = (type: TrackerTargetType, id: number) =>
    trackers.value.some(tracker => tracker.target_type === type && tracker.target_id === id)

async function createTracker(target: PickedTarget) {
    if (creating.value || pickedAlready(target.type, target.id)) return
    creating.value = true
    errorMessage.value = ''
    try {
        const created = await apiFetch<EntityTracker>('/api/me/trackers', {
            method: 'POST',
            body: {
                target_type: target.type,
                target_id: target.id,
                notifications_enabled: false,
            },
        })
        await refreshTrackers()
        selectedID.value = created.id
    } catch (error: any) {
        errorMessage.value = error?.data?.error || error?.message || 'Could not create tracker.'
    } finally {
        creating.value = false
    }
}

async function updateTracker(tracker: EntityTracker, changes: Partial<Pick<EntityTracker, 'enabled' | 'notifications_enabled'>>) {
    if (mutatingID.value !== null) return
    mutatingID.value = tracker.id
    errorMessage.value = ''
    try {
        await apiFetch(`/api/me/trackers/${tracker.id}`, {
            method: 'PATCH',
            body: changes,
        })
        await refreshTrackers()
    } catch (error: any) {
        errorMessage.value = error?.data?.error || error?.message || 'Could not update tracker.'
    } finally {
        mutatingID.value = null
    }
}

async function removeTracker(tracker: EntityTracker) {
    if (!window.confirm(`Delete the tracker for ${tracker.target_name}? Its recorded activity and alerts will also be removed.`)) return
    mutatingID.value = tracker.id
    errorMessage.value = ''
    try {
        await apiFetch(`/api/me/trackers/${tracker.id}`, { method: 'DELETE' })
        await refreshTrackers()
    } catch (error: any) {
        errorMessage.value = error?.data?.error || error?.message || 'Could not delete tracker.'
    } finally {
        mutatingID.value = null
    }
}

async function toggleBrowserAlerts() {
    await setBrowserAlerts(!browserAlertsEnabled.value)
}

const selectedTopics = computed(() => {
    const tracker = selected.value
    if (!tracker?.enabled) return null
    if (['character', 'corporation', 'alliance'].includes(tracker.target_type)) {
        return [`victim.${tracker.target_id}`, `attacker.${tracker.target_id}`]
    }
    return [`${tracker.target_type}.${tracker.target_id}`]
})

const selectedLosses = computed<KilllistLossEntities | undefined>(() => {
    const tracker = selected.value
    if (!tracker) return undefined
    if (tracker.target_type === 'character') return { characterIds: [tracker.target_id] }
    if (tracker.target_type === 'corporation') return { corporationIds: [tracker.target_id] }
    if (tracker.target_type === 'alliance') return { allianceIds: [tracker.target_id] }
    return undefined
})

function entityPath(tracker: EntityTracker): string {
    return `/${tracker.target_type}/${tracker.target_id}`
}

function entityImage(tracker: EntityTracker): string | null {
    if (tracker.target_type === 'character') return `/images/characters/${tracker.target_id}/portrait?size=64`
    if (tracker.target_type === 'corporation') return `/images/corporations/${tracker.target_id}/logo?size=64`
    if (tracker.target_type === 'alliance') return `/images/alliances/${tracker.target_id}/logo?size=64`
    return null
}

function targetIcon(type: TrackerTargetType): string {
    if (type === 'character') return 'lucide:user'
    if (type === 'corporation') return 'lucide:building-2'
    if (type === 'alliance') return 'lucide:flag'
    if (type === 'region') return 'lucide:map'
    if (type === 'constellation') return 'lucide:orbit'
    return 'lucide:map-pin'
}

function activityTime(value: string | null): string {
    if (!value) return 'No activity yet'
    return new Date(value).toLocaleString()
}
</script>

<template>
    <div>
        <PageHeader
            class="mb-4"
            title="Entity Trackers"
            eyebrow="Your watchlist"
            icon="lucide:radar"
            description="Record new killmail activity for entities and locations. Alerts are a separate, optional choice for each tracker."
        >
            <template #actions>
                <div v-if="isAuthenticated" class="flex flex-wrap items-center gap-2">
                    <NuxtLink to="/tracker" class="inline-flex items-center gap-2 rounded-md border border-blue-400/20 bg-blue-500/10 px-3 py-2 text-xs font-semibold text-blue-300 hover:bg-blue-500/20">
                        <Icon name="lucide:scan-eye" /> Tracker
                    </NuxtLink>
                    <button
                        type="button"
                        class="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-xs font-semibold transition-colors"
                        :class="browserAlertsEnabled
                            ? 'border-emerald-400/30 bg-emerald-500/10 text-emerald-300'
                            : 'border-white/10 bg-white/[0.04] text-gray-300 hover:border-blue-400/30 hover:text-blue-300'"
                        :disabled="browserPermission === 'unsupported' || browserPermission === 'denied'"
                        @click="toggleBrowserAlerts"
                    >
                        <Icon :name="browserAlertsEnabled ? 'lucide:bell-ring' : 'lucide:bell'" />
                        {{ browserAlertsEnabled ? 'Browser alerts on' : 'Browser alerts while open' }}
                    </button>
                </div>
            </template>
        </PageHeader>

        <LoginGate v-if="!isAuthenticated" message="Log in to create and manage entity trackers." />

        <template v-else>
            <div class="mb-4 rounded-lg border border-blue-400/15 bg-blue-500/[0.05] px-4 py-3 text-xs text-gray-400">
                <div class="flex items-start gap-2">
                    <Icon name="lucide:shield-check" class="mt-0.5 flex-shrink-0 text-blue-400" />
                    <p>
                        Browser alerts use this page's live connection only. They can appear while EVE-KILL is open in a tab,
                        but there is no push subscription and nothing is delivered after the site is closed.
                        <span v-if="browserPermission === 'denied'" class="text-amber-300">Notifications are currently blocked by your browser.</span>
                    </p>
                </div>
            </div>

            <div v-if="errorMessage" class="mb-4 rounded-lg border border-red-400/20 bg-red-500/[0.08] px-4 py-3 text-sm text-red-300">
                {{ errorMessage }}
            </div>

            <div class="grid gap-5 lg:grid-cols-[300px_minmax(0,1fr)]">
                <aside class="space-y-3">
                    <div class="glass-panel p-3">
                        <div class="mb-2 flex items-center justify-between">
                            <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Add a tracker</div>
                            <span class="text-fine tabular-nums text-gray-600">{{ trackers.length }}/{{ trackerLimit }}</span>
                        </div>
                        <SearchPicker
                            :types="['alliance', 'corporation', 'character', 'region', 'constellation', 'system']"
                            placeholder="Search entities or locations..."
                            :disabled="creating || trackers.length >= trackerLimit"
                            :is-picked="pickedAlready"
                            @select="createTracker"
                        />
                        <p class="mt-2 text-fine leading-relaxed text-gray-600">
                            Tracking starts now. Historical killmails are not backfilled.
                        </p>
                    </div>

                    <div class="glass-panel overflow-hidden">
                        <div v-if="trackersLoading && trackers.length === 0" class="px-4 py-10 text-center text-sm text-gray-500">
                            <Icon name="lucide:loader-2" class="mr-1 animate-spin" /> Loading trackers…
                        </div>
                        <div v-else-if="trackers.length === 0" class="px-4 py-10 text-center">
                            <Icon name="lucide:radar" class="mb-2 text-3xl text-gray-700" />
                            <p class="text-sm text-gray-400">No trackers yet</p>
                            <p class="mt-1 text-xs text-gray-600">Search above to start a watchlist.</p>
                        </div>
                        <button
                            v-for="tracker in trackers"
                            :key="tracker.id"
                            type="button"
                            class="relative flex w-full items-center gap-3 border-b border-white/[0.05] px-3 py-3 text-left transition-colors last:border-0 hover:bg-blue-500/[0.05]"
                            :class="selectedID === tracker.id ? 'bg-blue-500/[0.08]' : ''"
                            @click="selectedID = tracker.id"
                        >
                            <EveImage
                                v-if="entityImage(tracker)"
                                :src="entityImage(tracker)!"
                                :size="64"
                                :alt="tracker.target_name"
                                class="h-10 w-10 flex-shrink-0 rounded-md ring-1 ring-white/10"
                            />
                            <span v-else class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-md bg-white/[0.05] text-gray-500">
                                <Icon :name="targetIcon(tracker.target_type)" />
                            </span>
                            <span class="min-w-0 flex-1">
                                <span class="block truncate text-sm font-semibold" :class="tracker.enabled ? 'text-gray-200' : 'text-gray-500'">
                                    {{ tracker.target_name }}
                                </span>
                                <span class="mt-0.5 flex items-center gap-2 text-fine capitalize text-gray-600">
                                    {{ tracker.target_type }}
                                    <span v-if="tracker.target_ticker" class="font-mono">[{{ tracker.target_ticker }}]</span>
                                    <Icon v-if="tracker.notifications_enabled" name="lucide:bell" class="text-amber-400" v-tooltip="'Notifications enabled'" />
                                </span>
                            </span>
                            <span v-if="tracker.unread_notifications" class="rounded-full bg-red-500 px-1.5 py-0.5 text-fine font-bold text-white">
                                {{ tracker.unread_notifications > 99 ? '99+' : tracker.unread_notifications }}
                            </span>
                            <span v-if="selectedID === tracker.id" class="absolute inset-y-2 left-0 w-0.5 rounded-full bg-blue-400" />
                        </button>
                    </div>
                </aside>

                <main v-if="selected" class="min-w-0 space-y-4">
                    <div class="glass-panel p-4">
                        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                            <div class="flex min-w-0 items-center gap-3">
                                <EveImage
                                    v-if="entityImage(selected)"
                                    :src="entityImage(selected)!"
                                    :size="64"
                                    :alt="selected.target_name"
                                    class="h-14 w-14 flex-shrink-0 rounded-lg ring-1 ring-white/10"
                                />
                                <span v-else class="flex h-14 w-14 flex-shrink-0 items-center justify-center rounded-lg bg-white/[0.05] text-xl text-gray-500">
                                    <Icon :name="targetIcon(selected.target_type)" />
                                </span>
                                <div class="min-w-0">
                                    <NuxtLink :to="entityPath(selected)" class="truncate text-lg font-bold text-white hover:text-blue-300">
                                        {{ selected.target_name }}
                                    </NuxtLink>
                                    <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500">
                                        <span class="capitalize">{{ selected.target_type }}</span>
                                        <span>{{ selected.event_count.toLocaleString() }} recorded kills</span>
                                        <span>{{ activityTime(selected.last_event_at) }}</span>
                                    </div>
                                </div>
                            </div>

                            <div class="flex flex-wrap items-center gap-2">
                                <button
                                    type="button"
                                    class="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-xs font-semibold transition-colors"
                                    :class="selected.enabled
                                        ? 'border-emerald-400/25 bg-emerald-500/10 text-emerald-300'
                                        : 'border-white/10 bg-white/[0.03] text-gray-500'"
                                    :disabled="mutatingID !== null"
                                    @click="updateTracker(selected, { enabled: !selected.enabled })"
                                >
                                    <Icon :name="selected.enabled ? 'lucide:radio' : 'lucide:pause'" />
                                    {{ selected.enabled ? 'Tracking' : 'Paused' }}
                                </button>
                                <button
                                    type="button"
                                    class="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-xs font-semibold transition-colors"
                                    :class="selected.notifications_enabled
                                        ? 'border-amber-400/25 bg-amber-500/10 text-amber-300'
                                        : 'border-white/10 bg-white/[0.03] text-gray-500'"
                                    :disabled="mutatingID !== null"
                                    @click="updateTracker(selected, { notifications_enabled: !selected.notifications_enabled })"
                                >
                                    <Icon :name="selected.notifications_enabled ? 'lucide:bell-ring' : 'lucide:bell-off'" />
                                    {{ selected.notifications_enabled ? 'Notifying' : 'Notifications off' }}
                                </button>
                                <button
                                    type="button"
                                    class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-red-400/15 text-red-400/70 transition-colors hover:bg-red-500/10 hover:text-red-300"
                                    :disabled="mutatingID !== null"
                                    aria-label="Delete tracker"
                                    @click="removeTracker(selected)"
                                >
                                    <Icon name="lucide:trash-2" />
                                </button>
                            </div>
                        </div>
                        <div class="mt-3 border-t border-white/[0.06] pt-3 text-xs text-gray-500">
                            <span v-if="selected.notifications_enabled">
                                New matches are recorded here and also added to your notification bell.
                            </span>
                            <span v-else>
                                New matches are recorded here without creating notifications. Turn alerts on only when this watch is important enough.
                            </span>
                        </div>
                    </div>

                    <KillList
                        :key="`${selected.id}-${selected.updated_at}`"
                        :entity-endpoint="`/api/me/trackers/${selected.id}/killmails`"
                        :stream-topics="selectedTopics"
                        :loss-entities="selectedLosses"
                    />
                </main>
            </div>
        </template>
    </div>
</template>
