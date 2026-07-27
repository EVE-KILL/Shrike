<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const id = Number(route.params.id)
if (!Number.isInteger(id) || id < 1 || id > 2147483647) {
    throw createError({ statusCode: 404, statusMessage: 'Battle not found' })
}

// Support killmail-based on-the-fly battle generation via ?killmail=<id>
const killmailId = Number(route.query.killmail) || null
const isKillmailBattle = killmailId && killmailId > 0

const apiUrl = isKillmailBattle
    ? `/api/battle/killmail/${killmailId}`
    : `/api/battle/${id}`

const { data, pending, error } = await useApiFetch<any>(apiUrl)

// If the killmail-based API found a saved battle, redirect to it
if (isKillmailBattle && data.value?.redirect && data.value?.battle_id) {
    await navigateTo(`/battle/${data.value.battle_id}`, { redirectCode: 302, replace: true })
}

if (error.value && !isKillmailBattle) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: 'Battle not found',
    })
}

const battle = computed(() => data.value)

// Corp-palette team accents; null per team → keep the default red/blue
const teamAccents = computed(() => battleTeamAccents((battle.value?.teams ?? []).map((t: any) => t.dominant_corp_palette)))

// A killmail-mode fetch is "failed" if the API errored OR returned a body
// without a proper teams array. The composer endpoint can legitimately
// return a shape like { teams: undefined } when it can't construct two
// opposing sides from a lone killmail — without this guard the template's
// `battle.teams.length` crashes SSR with "undefined is not an object".
// The { redirect, battle_id } response is handled by the navigateTo above
// so we don't need to match it here.
const battleFailed = computed(() => {
    if (!isKillmailBattle) return false
    if (error.value) return true
    return !!battle.value && !Array.isArray(battle.value.teams)
})

// Tabs — same pattern as character page
const tabs = computed(() => {
    const base: Array<{ id: string, label: string, icon: string }> = [
        { id: 'summary', label: 'Summary', icon: 'lucide:layout-list' },
        { id: 'kills', label: 'Kills', icon: 'lucide:target' },
        { id: 'timeline', label: 'Timeline', icon: 'lucide:clock' },
        { id: 'composition', label: 'Composition', icon: 'lucide:users' },
        { id: 'intel', label: 'Intel', icon: 'lucide:scan-eye' },
    ]
    // Comments are only available for saved battles (need a stable target_id)
    if (!isKillmailBattle) {
        base.push({ id: 'comments', label: 'Comments', icon: 'lucide:message-square' })
    }
    return base
})
type TabId = 'summary' | 'kills' | 'timeline' | 'composition' | 'intel' | 'comments'
const tabIds = computed(() => new Set(tabs.value.map(t => t.id as TabId)))

// Unknown tab segments 404 rather than falling back to the summary tab, which
// would answer 200 at a URL whose canonical points to /battle/<id> — unbounded
// URLs collapsing onto one canonical. `tabs` only depends on isKillmailBattle,
// which is settled at setup, so this is safe to evaluate here.
const battleTabParam = route.params.tab
if (typeof battleTabParam === 'string' && battleTabParam.length > 0 && !tabIds.value.has(battleTabParam as TabId)) {
    throw createError({ statusCode: 404, statusMessage: 'Battle tab not found' })
}

// Force remount on id change — `id`/`apiUrl` are captured at setup, not reactive.
definePageMeta({
    key: route => `/battle/${route.params.id}`,
})

const activeTab = computed<TabId>(() => {
    const param = route.params.tab as string
    if (param && tabIds.value.has(param as TabId)) return param as TabId
    return 'summary'
})

const setTab = (tabId: string) => {
    if (!tabIds.value.has(tabId as TabId)) return
    const basePath = tabId === 'summary' ? `/battle/${id}` : `/battle/${id}/${tabId}`
    const query = isKillmailBattle ? `?killmail=${killmailId}` : ''
    useAnalytics().track('tab.change', { entity: 'battle', tab: tabId })
    router.push(basePath + query)
}

// Formatting
const formatDate = (iso: string): string => {
    const d = new Date(iso)
    const y = d.getUTCFullYear()
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
    const m = months[d.getUTCMonth()]
    const day = d.getUTCDate()
    const hh = String(d.getUTCHours()).padStart(2, '0')
    const mm = String(d.getUTCMinutes()).padStart(2, '0')
    return `${m} ${day}, ${y}, ${hh}:${mm} EVE`
}

const formatDuration = (mins: number): string => {
    if (mins < 60) return `${mins}m`
    const h = Math.floor(mins / 60)
    const m = mins % 60
    return m > 0 ? `${h}h ${m}m` : `${h}h`
}

const teamTopName = (teamIdx: number): string => {
    const team = battle.value?.teams?.[teamIdx]
    if (!team?.alliances?.length) return `Team ${teamIdx + 1}`
    const first = team.alliances[0]
    return first.alliance_name || first.corporations?.[0]?.corporation_name || `Team ${teamIdx + 1}`
}

const battleTitle = computed(() => battle.value
    ? `Battle in ${battle.value.solar_system_name} — ${formatDate(battle.value.start_time)}`
    : 'Battle')

/**
 * Hand this report to the battle generator with its system and window already
 * filled in, landing on side assignment.
 *
 * Both kinds of battle get it, for the same reason: their sides are inferred
 * from the killmails rather than chosen, and the inference is wrong often
 * enough to be worth correcting by hand. The difference is where the result
 * goes — a killmail report has nothing to update, so saving creates a battle;
 * a saved battle passes its id and gets its sides rewritten in place.
 */
const editSidesLink = computed(() => {
    if (!battle.value) return null
    return {
        path: '/battlegenerator',
        query: {
            system: String(battle.value.solar_system_id),
            systemName: battle.value.solar_system_name,
            start: battle.value.start_time,
            end: battle.value.end_time,
            step: 'sides',
            ...(isKillmailBattle ? {} : { battle: String(id) }),
        },
    }
})

const activeTabLabel = computed(() => {
    if (activeTab.value === 'summary') return null
    return tabs.value.find(t => t.id === activeTab.value)?.label ?? null
})
useHead({ title: computed(() => {
    return activeTabLabel.value ? `${battleTitle.value} (${activeTabLabel.value})` : battleTitle.value
}) })
useSeoMeta({
    description: computed(() => {
        const b = battle.value
        if (!b) return 'View battle report on EVE-KILL.'
        const kills = b.kill_count ?? 0
        const isk = b.total_isk_destroyed ? formatIsk(b.total_isk_destroyed) : '0'
        const duration = b.duration_minutes ? formatDuration(b.duration_minutes) : ''
        const t0 = b.teams?.[0] ? teamTopName(0) : null
        const t1 = b.teams?.[1] ? teamTopName(1) : null
        let desc = `Battle in ${b.solar_system_name} — ${kills} kills, ${isk} ISK destroyed`
        if (duration) desc += `, ${duration} long`
        if (t0 && t1) desc += `. ${t0} vs ${t1}`
        desc += '. Full battle report on EVE-KILL.'
        return desc
    }),
    ogTitle: computed(() => {
        const b = battle.value
        if (!b) return 'Battle — EVE-KILL'
        const t0 = b.teams?.[0] ? teamTopName(0) : null
        const t1 = b.teams?.[1] ? teamTopName(1) : null
        if (t0 && t1) return `${t0} vs ${t1} — ${b.solar_system_name}`
        return `Battle in ${b.solar_system_name} — EVE-KILL`
    }),
    ogDescription: computed(() => {
        const b = battle.value
        if (!b) return 'View battle report on EVE-KILL.'
        const kills = b.kill_count ?? 0
        const isk = b.total_isk_destroyed ? formatIsk(b.total_isk_destroyed) : '0'
        return `${kills} kills, ${isk} ISK destroyed in ${b.solar_system_name} — battle report on EVE-KILL.`
    }),
    ogType: 'website',
    twitterCard: 'summary',
    twitterTitle: computed(() => battleTitle.value ? `${battleTitle.value} — EVE-KILL` : 'Battle — EVE-KILL'),
    twitterDescription: computed(() => {
        const b = battle.value
        if (!b) return 'View battle report on EVE-KILL.'
        const kills = b.kill_count ?? 0
        const isk = b.total_isk_destroyed ? formatIsk(b.total_isk_destroyed) : '0'
        return `${kills} kills, ${isk} ISK destroyed in ${b.solar_system_name}.`
    }),
    ogImage: computed(() => {
        const team = battle.value?.teams?.[0]
        const ally = team?.alliances?.[0]
        if (ally?.alliance_id) return `/images/alliances/${ally.alliance_id}/logo?size=256`
        const corp = ally?.corporations?.[0]
        if (corp?.corporation_id) return `/images/corporations/${corp.corporation_id}/logo?size=256`
        return 'https://eve-kill.com/og-default.png'
    }),
    twitterImage: computed(() => {
        const team = battle.value?.teams?.[0]
        const ally = team?.alliances?.[0]
        if (ally?.alliance_id) return `/images/alliances/${ally.alliance_id}/logo?size=256`
        const corp = ally?.corporations?.[0]
        if (corp?.corporation_id) return `/images/corporations/${corp.corporation_id}/logo?size=256`
        return 'https://eve-kill.com/og-default.png'
    }),
})

useSchemaOrg([
    defineBreadcrumb({
        itemListElement: computed(() => [
            { name: 'Home', item: '/' },
            { name: 'Battles', item: '/battles' },
            { name: battle.value ? `Battle in ${battle.value.solar_system_name}` : 'Battle', item: `/battle/${id}` },
        ]),
    }),
    {
        '@type': 'Event',
        'name': computed(() => battleTitle.value),
        'url': `https://eve-kill.com/battle/${id}`,
        'startDate': computed(() => battle.value?.start_time || undefined),
        'endDate': computed(() => battle.value?.end_time || undefined),
        'eventStatus': 'https://schema.org/EventCompleted',
        'eventAttendanceMode': 'https://schema.org/OnlineEventAttendanceMode',
        'location': computed(() => {
            const b = battle.value
            if (!b?.solar_system_name) return { '@type': 'VirtualLocation', url: 'https://www.eveonline.com' }
            return {
                '@type': 'Place',
                'name': b.solar_system_name,
                'url': `https://eve-kill.com/system/${b.solar_system_id}`,
                'containedInPlace': b.region_name ? {
                    '@type': 'Place',
                    'name': b.region_name,
                    'url': `https://eve-kill.com/region/${b.region_id}`,
                } : undefined,
            }
        }),
        'attendee': computed(() => {
            const teams = battle.value?.teams
            if (!Array.isArray(teams)) return undefined
            const list: { '@type': string; name: string; url: string }[] = []
            for (const team of teams) {
                for (const ally of team?.alliances || []) {
                    if (ally?.alliance_id && ally.alliance_name) {
                        list.push({
                            '@type': 'Organization',
                            name: ally.alliance_name,
                            url: `https://eve-kill.com/alliance/${ally.alliance_id}`,
                        })
                    }
                }
            }
            return list.length ? list.slice(0, 20) : undefined
        }),
    },
])

const securityColor = (sec: number | null): string => {
    if (sec == null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-green-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

const efficiency = computed(() => {
    if (!battle.value?.teams?.length || battle.value.teams.length < 2) return 50
    const t0 = battle.value.teams[0].total_isk_destroyed || 0
    const t1 = battle.value.teams[1].total_isk_destroyed || 0
    const total = t0 + t1
    if (total === 0) return 50
    return Math.round((t0 / total) * 100)
})

const team0Corps = computed(() => battle.value?.team_entities?.[0]?.corps || [])
const team0Alliances = computed(() => battle.value?.team_entities?.[0]?.alliances || [])
const team1Corps = computed(() => battle.value?.team_entities?.[1]?.corps || [])
const team1Alliances = computed(() => battle.value?.team_entities?.[1]?.alliances || [])

const teamTopImage = (teamIdx: number): string | null => {
    const team = battle.value?.teams?.[teamIdx]
    if (!team?.alliances?.length) return null
    const first = team.alliances[0]
    if (first.alliance_id) return `/images/alliances/${first.alliance_id}/logo?size=128`
    if (first.corporations?.[0]) return `/images/corporations/${first.corporations[0].corporation_id}/logo?size=128`
    return null
}

// API endpoints — switch based on killmail vs saved battle mode
const killlistEndpoint = computed(() =>
    isKillmailBattle ? `/api/battle/killmail/${killmailId}/killlist` : `/api/battle/${id}/killlist`
)
const mostValuableEndpoint = computed(() =>
    isKillmailBattle ? `/api/battle/killmail/${killmailId}/most-valuable` : `/api/battle/${id}/most-valuable`
)
const mostValuableParams = computed(() => battle.value ? {
    systemId: battle.value.solar_system_id,
    start: battle.value.start_time,
    end: battle.value.end_time,
} : {})
const compositionEndpoint = computed(() =>
    isKillmailBattle ? `/api/battle/killmail/${killmailId}/composition` : `/api/battle/${id}/composition`
)
const compositionParams = computed(() => isKillmailBattle && battle.value ? {
    systemId: battle.value.solar_system_id,
    start: battle.value.start_time,
    end: battle.value.end_time,
} : undefined)
const intelEndpoint = computed(() =>
    isKillmailBattle ? `/api/battle/killmail/${killmailId}/intel` : `/api/battle/${id}/intel`
)
const intelParams = computed(() => isKillmailBattle && battle.value ? {
    systemId: battle.value.solar_system_id,
    start: battle.value.start_time,
    end: battle.value.end_time,
} : undefined)
const timelineEndpoint = computed(() => {
    if (isKillmailBattle) {
        const b = battle.value
        if (!b) return ''
        return `/api/battle/killmail/${killmailId}/timeline?systemId=${b.solar_system_id}&start=${encodeURIComponent(b.start_time)}&end=${encodeURIComponent(b.end_time)}`
    }
    return `/api/battle/${id}/timeline`
})
</script>

<template>
    <div>
        <div v-if="pending" class="h-64 rounded-lg bg-white/[0.04] animate-pulse"></div>

        <div v-else-if="battleFailed" class="glass-panel p-8 text-center">
            <Icon name="lucide:swords" class="text-3xl text-gray-600 mb-3" />
            <p class="text-sm text-gray-400 mb-2">Could not generate a battle report from this killmail</p>
            <p class="text-xs text-gray-600 mb-4">There may not be enough opposing sides in this area to form a battle.</p>
            <NuxtLink v-if="killmailId" :to="`/kill/${killmailId}`"
                class="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium bg-white/[0.06] text-gray-300 border border-white/[0.08] hover:bg-blue-500/[0.1] transition-colors">
                <Icon name="lucide:arrow-left" class="text-sm" />
                Back to killmail
            </NuxtLink>
        </div>

        <div v-else-if="battle">
            <h1 class="sr-only">{{ battleTitle }}</h1>

            <!-- Generated badge for killmail-based battles -->
            <div v-if="isKillmailBattle" class="flex flex-wrap items-center gap-2 mb-3 text-xs text-gray-500">
                <span class="px-2 py-0.5 rounded bg-amber-500/20 text-amber-400 font-medium text-fine">Generated</span>
                <span>Battle report generated from</span>
                <NuxtLink :to="`/kill/${killmailId}`" class="text-blue-400 hover:text-blue-300">
                    killmail #{{ killmailId }}
                </NuxtLink>
                <!-- Sides here are inferred from one killmail; this is the way
                     to redo them properly and save the result. -->
                <NuxtLink v-if="editSidesLink" :to="editSidesLink"
                    class="ml-auto inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md font-medium border border-white/[0.08] bg-white/[0.04] text-gray-300 hover:bg-blue-500/[0.12] hover:text-blue-300 hover:border-blue-500/30 transition-colors"
                    v-tooltip="'Reassign the sides in the battle generator and save it as a real battle'">
                    <Icon name="lucide:users" class="text-fine" />
                    Edit sides
                </NuxtLink>
            </div>

            <!-- Header: System + Stats -->
            <div class="glass-panel p-5 mb-4">
                <div class="flex items-center gap-3 mb-4 pb-3 border-b border-white/[0.06]">
                    <img :src="`/images/systems/${battle.solar_system_id}?size=64`" alt="" class="w-10 h-10 rounded-lg flex-shrink-0" loading="eager">
                    <div>
                        <span class="text-lg font-bold text-white" :class="pochvenClass(battle.region_id)">{{ battle.solar_system_name }}</span>
                        <span class="ml-2 text-sm tabular-nums" :class="securityColor(battle.solar_system_security)">
                            {{ battle.solar_system_security?.toFixed(1) }}
                        </span>
                        <span class="ml-2 text-sm text-gray-500" :class="pochvenClass(battle.region_id)">{{ battle.region_name }}</span>
                    </div>
                    <div class="ml-auto flex items-center gap-2">
                        <span v-if="battle.is_multi_party" class="px-2 py-0.5 text-fine rounded bg-amber-500/20 text-amber-400 font-medium">Multi-party</span>
                        <span v-if="battle.is_custom" class="px-2 py-0.5 text-fine rounded bg-purple-500/20 text-purple-400 font-medium">Custom</span>
                        <!-- Killmail reports carry this up in the Generated row
                             instead, next to the killmail they came from. -->
                        <NuxtLink v-if="!isKillmailBattle && editSidesLink" :to="editSidesLink"
                            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium border border-white/[0.08] bg-white/[0.04] text-gray-300 hover:bg-blue-500/[0.12] hover:text-blue-300 hover:border-blue-500/30 transition-colors"
                            v-tooltip="'Reassign the sides — anyone signed in can correct them'">
                            <Icon name="lucide:users" class="text-fine" />
                            Edit sides
                        </NuxtLink>
                    </div>
                </div>

                <!-- When it happened — a line of prose beats three label/value rows -->
                <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-fine text-gray-500 mb-3">
                    <span class="flex items-center gap-1.5">
                        <Icon name="lucide:calendar-range" class="text-fine text-gray-600" />
                        <span class="tabular-nums">{{ formatDate(battle.start_time) }}</span>
                        <span class="text-gray-600">→</span>
                        <span class="tabular-nums">{{ formatDate(battle.end_time) }}</span>
                    </span>
                    <span class="flex items-center gap-1.5">
                        <Icon name="lucide:clock" class="text-fine text-gray-600" />
                        <span class="text-gray-300">{{ formatDuration(battle.duration_minutes) }}</span>
                    </span>
                </div>

                <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
                    <div v-for="tile in [
                        { label: 'Ships Lost', icon: 'lucide:crosshair', value: formatNumber(battle.kill_count), color: 'text-white' },
                        { label: 'ISK Destroyed', icon: 'lucide:coins', value: formatIsk(battle.total_isk_destroyed), color: 'text-yellow-400' },
                        { label: 'Damage', icon: 'lucide:zap', value: formatNumber(battle.total_damage), color: 'text-gray-300' },
                        { label: 'Characters', icon: 'lucide:user-round', value: formatNumber(battle.characters_involved), color: 'text-blue-400' },
                        { label: 'Corporations', icon: 'lucide:building-2', value: formatNumber(battle.corporations_involved), color: 'text-purple-400' },
                        { label: 'Alliances', icon: 'lucide:shield', value: formatNumber(battle.alliances_involved), color: 'text-indigo-400' },
                    ]" :key="tile.label" class="rounded-lg bg-white/[0.03] border border-white/[0.06] p-3 text-center">
                        <div class="text-lg font-bold tabular-nums" :class="tile.color">{{ tile.value }}</div>
                        <div class="flex items-center justify-center gap-1.5 text-fine text-gray-500 uppercase tracking-wider mt-0.5">
                            <Icon :name="tile.icon" class="text-fine text-gray-600" />{{ tile.label }}
                        </div>
                    </div>
                </div>
            </div>

            <!-- Team Summary Cards -->
            <div v-if="battle.teams?.length >= 2" class="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] gap-4 mb-6 items-center">
                <div class="rounded-lg bg-white/[0.04] border border-red-500/20 p-4"
                    :style="teamAccents[0] ? { borderColor: teamAccents[0].border } : undefined">
                    <div class="mb-3">
                        <div class="text-sm font-bold text-red-400 mb-1.5" :style="teamAccents[0] ? { color: teamAccents[0].accent } : undefined">Team A</div>
                        <div class="flex items-center gap-2 min-w-0">
                            <img v-if="teamTopImage(0)" :src="teamTopImage(0)!" class="w-6 h-6 rounded bg-gray-900 flex-shrink-0" loading="eager">
                            <span class="text-xs text-gray-300 truncate">{{ teamTopName(0) }}</span>
                            <span v-if="battle.teams[0].alliance_count > 1 || battle.teams[0].corp_count > 1" class="text-fine text-gray-500 flex-shrink-0">
                                +{{ Math.max(0, battle.teams[0].alliance_count - 1) }} alli · +{{ Math.max(0, battle.teams[0].corp_count - 1) }} corps
                            </span>
                        </div>
                    </div>
                    <div class="grid grid-cols-2 gap-2 text-xs">
                        <div>
                            <span class="text-green-400 tabular-nums font-medium">{{ formatNumber(battle.teams[0].total_kills) }}</span>
                            <span v-if="battle.unsided?.kills" class="text-fine text-gray-600 ml-1" v-tooltip="`Up to ${formatNumber(battle.unsided.kills)} more from unsided participants`">±{{ formatNumber(battle.unsided.kills) }}</span>
                            <div class="text-gray-500">kills</div>
                        </div>
                        <div>
                            <span class="text-red-400 tabular-nums font-medium">{{ formatNumber(battle.teams[0].total_losses) }}</span>
                            <span v-if="battle.unsided?.losses" class="text-fine text-gray-600 ml-1" v-tooltip="`Up to ${formatNumber(battle.unsided.losses)} more from unsided participants`">±{{ formatNumber(battle.unsided.losses) }}</span>
                            <div class="text-gray-500">losses</div>
                        </div>
                        <div>
                            <span class="text-green-400 tabular-nums font-medium">{{ formatIsk(battle.teams[0].total_isk_destroyed) }}</span>
                            <span v-if="battle.unsided?.isk_destroyed" class="text-fine text-gray-600 ml-1" v-tooltip="`Up to ${formatIsk(battle.unsided.isk_destroyed)} more from unsided participants`">±{{ formatIsk(battle.unsided.isk_destroyed) }}</span>
                            <div class="text-gray-500">destroyed</div>
                        </div>
                        <div>
                            <span class="text-red-400 tabular-nums font-medium">{{ formatIsk(battle.teams[0].total_isk_lost) }}</span>
                            <span v-if="battle.unsided?.isk_lost" class="text-fine text-gray-600 ml-1" v-tooltip="`Up to ${formatIsk(battle.unsided.isk_lost)} more from unsided participants`">±{{ formatIsk(battle.unsided.isk_lost) }}</span>
                            <div class="text-gray-500">lost</div>
                        </div>
                    </div>
                </div>

                <div class="flex flex-col items-center gap-2 px-4">
                    <div class="text-2xl font-bold text-gray-600">VS</div>
                    <div class="w-32 h-2 bg-blue-500/30 rounded-full overflow-hidden"
                        :style="teamAccents[1] ? { backgroundColor: teamAccents[1].border } : undefined">
                        <div class="h-full bg-red-500 rounded-full" :style="{ width: `${efficiency}%`, ...(teamAccents[0] ? { backgroundColor: teamAccents[0].accent } : {}) }"></div>
                    </div>
                    <div class="text-fine text-gray-500 tabular-nums">{{ efficiency }}% — {{ 100 - efficiency }}%</div>
                </div>

                <div class="rounded-lg bg-white/[0.04] border border-blue-500/20 p-4"
                    :style="teamAccents[1] ? { borderColor: teamAccents[1].border } : undefined">
                    <div class="mb-3">
                        <div class="text-sm font-bold text-blue-400 mb-1.5" :style="teamAccents[1] ? { color: teamAccents[1].accent } : undefined">Team B</div>
                        <div class="flex items-center gap-2 min-w-0">
                            <img v-if="teamTopImage(1)" :src="teamTopImage(1)!" class="w-6 h-6 rounded bg-gray-900 flex-shrink-0" loading="eager">
                            <span class="text-xs text-gray-300 truncate">{{ teamTopName(1) }}</span>
                            <span v-if="battle.teams[1].alliance_count > 1 || battle.teams[1].corp_count > 1" class="text-fine text-gray-500 flex-shrink-0">
                                +{{ Math.max(0, battle.teams[1].alliance_count - 1) }} alli · +{{ Math.max(0, battle.teams[1].corp_count - 1) }} corps
                            </span>
                        </div>
                    </div>
                    <div class="grid grid-cols-2 gap-2 text-xs">
                        <div>
                            <span class="text-green-400 tabular-nums font-medium">{{ formatNumber(battle.teams[1].total_kills) }}</span>
                            <span v-if="battle.unsided?.kills" class="text-fine text-gray-600 ml-1" v-tooltip="`Up to ${formatNumber(battle.unsided.kills)} more from unsided participants`">±{{ formatNumber(battle.unsided.kills) }}</span>
                            <div class="text-gray-500">kills</div>
                        </div>
                        <div>
                            <span class="text-red-400 tabular-nums font-medium">{{ formatNumber(battle.teams[1].total_losses) }}</span>
                            <span v-if="battle.unsided?.losses" class="text-fine text-gray-600 ml-1" v-tooltip="`Up to ${formatNumber(battle.unsided.losses)} more from unsided participants`">±{{ formatNumber(battle.unsided.losses) }}</span>
                            <div class="text-gray-500">losses</div>
                        </div>
                        <div>
                            <span class="text-green-400 tabular-nums font-medium">{{ formatIsk(battle.teams[1].total_isk_destroyed) }}</span>
                            <span v-if="battle.unsided?.isk_destroyed" class="text-fine text-gray-600 ml-1" v-tooltip="`Up to ${formatIsk(battle.unsided.isk_destroyed)} more from unsided participants`">±{{ formatIsk(battle.unsided.isk_destroyed) }}</span>
                            <div class="text-gray-500">destroyed</div>
                        </div>
                        <div>
                            <span class="text-red-400 tabular-nums font-medium">{{ formatIsk(battle.teams[1].total_isk_lost) }}</span>
                            <span v-if="battle.unsided?.isk_lost" class="text-fine text-gray-600 ml-1" v-tooltip="`Up to ${formatIsk(battle.unsided.isk_lost)} more from unsided participants`">±{{ formatIsk(battle.unsided.isk_lost) }}</span>
                            <div class="text-gray-500">lost</div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Most Valuable: per-team kills inflicted on the other side -->
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
                <div>
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-red-400/80 mb-1.5 px-1">
                        Team A's most valuable kills
                    </div>
                    <MostValuable
                        compact
                        :api-endpoint="mostValuableEndpoint"
                        :extra-params="{ ...mostValuableParams, team: 0 }"
                    />
                </div>
                <div>
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1.5 px-1">
                        Team B's most valuable kills
                    </div>
                    <MostValuable
                        compact
                        :api-endpoint="mostValuableEndpoint"
                        :extra-params="{ ...mostValuableParams, team: 1 }"
                    />
                </div>
            </div>

            <!-- Tab Bar -->
            <div class="flex overflow-x-auto border-b border-white/[0.08] mb-4 scrollbar-hide">
                <button v-for="t in tabs" :key="t.id"
                    class="flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors border-b-2 cursor-pointer"
                    :class="activeTab === t.id
                        ? 'text-white border-white'
                        : 'text-gray-500 border-transparent hover:text-blue-400'"
                    @click="setTab(t.id)">
                    <Icon :name="t.icon" class="text-base" />
                    {{ t.label }}
                </button>
            </div>

            <!-- Tab Content -->
            <BattleSummary v-if="activeTab === 'summary'" :teams="battle.teams" :unsided="battle.unsided" />

            <Suspense v-if="activeTab === 'kills'">
                <KillList
                    :api-endpoint="killlistEndpoint"
                    :extra-params="{
                        systemId: battle.solar_system_id,
                        start: battle.start_time,
                        end: battle.end_time,
                    }"
                    :war-aggressor-corps="team0Corps"
                    :war-aggressor-alliances="team0Alliances"
                    :war-defender-corps="team1Corps"
                    :war-defender-alliances="team1Alliances"
                    :key="`battle-kills-${id}`"
                />
                <template #fallback>
                    <div class="flex items-center justify-center py-12">
                        <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                    </div>
                </template>
            </Suspense>

            <div v-if="activeTab === 'comments'" class="mt-2">
                <CommentsCommentList :target-type="7" :target-id="id" />
            </div>

            <Suspense v-if="activeTab === 'composition'">
                <BattleComposition
                    :api-endpoint="compositionEndpoint"
                    :extra-params="compositionParams"
                    :key="`battle-comp-${isKillmailBattle ? `km-${killmailId}` : id}`"
                />
                <template #fallback>
                    <div class="flex items-center justify-center py-12">
                        <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                    </div>
                </template>
            </Suspense>

            <Suspense v-if="activeTab === 'intel'">
                <BattleIntel
                    :api-endpoint="intelEndpoint"
                    :extra-params="intelParams"
                    :key="`battle-intel-${isKillmailBattle ? `km-${killmailId}` : id}`"
                />
                <template #fallback>
                    <div class="flex items-center justify-center py-12">
                        <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                    </div>
                </template>
            </Suspense>

            <Suspense v-if="activeTab === 'timeline'">
                <BattleTimeline
                    :battle-id="isKillmailBattle ? null : id"
                    :solar-system-id="battle.solar_system_id"
                    :start-time="battle.start_time"
                    :end-time="battle.end_time"
                    :teams="battle.teams"
                    :team-entities="battle.team_entities"
                    :timeline-endpoint="isKillmailBattle ? timelineEndpoint : undefined"
                />
                <template #fallback>
                    <div class="flex items-center justify-center py-12">
                        <Icon name="lucide:loader-2" class="w-5 h-5 text-gray-500 animate-spin" />
                    </div>
                </template>
            </Suspense>
        </div>
    </div>
</template>
