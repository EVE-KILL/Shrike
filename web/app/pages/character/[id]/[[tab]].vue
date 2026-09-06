<script setup lang="ts">
// Tab system with route segments — /character/123/kills
const tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: 'lucide:layout-dashboard' },
    { id: 'combined', label: 'Combined', icon: 'lucide:layers' },
    { id: 'kills', label: 'Kills', icon: 'lucide:trophy' },
    { id: 'losses', label: 'Losses', icon: 'lucide:skull' },
    { id: 'top', label: 'Top', icon: 'lucide:trending-up' },
    { id: 'battles', label: 'Battles', icon: 'lucide:swords' },
    { id: 'wars', label: 'Wars', icon: 'lucide:flag' },
    { id: 'campaigns', label: 'Campaigns', icon: 'lucide:target' },
    { id: 'history', label: 'History', icon: 'lucide:history' },
    { id: 'achievements', label: 'Achievements', icon: 'lucide:award' },
] as const

// Keep the same page instance when only the tab param changes
definePageMeta({
    key: route => `/character/${route.params.id}`,
})

const characterId = Number(useRoute().params.id)
if (!Number.isInteger(characterId) || characterId < 1 || characterId > 2147483647) {
    throw createError({ statusCode: 404, statusMessage: 'Character not found' })
}
const characterRequest = await useApiFetch<any>(`/api/character/${characterId}`)

const {
    id, data, pending,
    entity: char, stats, recentStats, accent,
    activeTab, setTab, killlistRole, killlistExtraParams, topLists,
    formatDate, ageYears: characterAgeYears, effWidth, iskEffWidth, dangerRatio,
} = useEntityPage('character', {
    tabs,
    titleBase: c => c?.name || null,
}, characterRequest)

const corpHistory = computed(() => data.value?.corporationHistory || [])
const corpHistoryQueued = computed(() => data.value?.corporationHistoryQueued || false)
const allTimeRanking = computed(() => data.value?.rankings?.all_time || null)

useSeoMeta({
    description: computed(() => {
        const c = char.value
        if (!c?.name) return 'View character kill statistics on EVE-KILL.'
        const s = stats.value
        let desc = `${c.name}`
        if (c.corporation_name) desc += ` — ${c.corporation_name}`
        if (c.alliance_name) desc += ` [${c.alliance_name}]`
        if (s?.kills || s?.losses) desc += `. ${s.kills ?? 0} kills, ${s.losses ?? 0} losses`
        desc += ' — EVE Online pilot stats on EVE-KILL.'
        return desc
    }),
    ogTitle: computed(() => {
        const c = char.value
        if (!c?.name) return 'Character — EVE-KILL'
        let t = c.name
        if (c.corporation_name) t += ` | ${c.corporation_name} [${c.corporation_ticker}]`
        if (c.alliance_name) t += ` | ${c.alliance_name} [${c.alliance_ticker}]`
        return t
    }),
    ogDescription: computed(() => {
        const c = char.value
        if (!c?.name) return 'View character stats on EVE-KILL.'
        return `${c.name} — kills, losses, and combat stats in EVE Online.`
    }),
    ogImage: computed(() => char.value ? `/images/characters/${char.value.character_id}/portrait?size=512` : ''),
    ogType: 'profile',
    twitterCard: 'summary',
    twitterTitle: computed(() => char.value?.name ? `${char.value.name} — EVE-KILL` : 'Character — EVE-KILL'),
    twitterDescription: computed(() => {
        const c = char.value
        if (!c?.name) return 'View character stats on EVE-KILL.'
        return `${c.name} — kills, losses, and combat stats in EVE Online.`
    }),
    twitterImage: computed(() => char.value ? `/images/characters/${char.value.character_id}/portrait?size=512` : ''),
})

useSchemaOrg([
    defineBreadcrumb(computed(() => ({
        itemListElement: (() => {
            const crumbs: { name: string; item: string }[] = [{ name: 'Home', item: '/' }]
            const c = char.value
            if (c?.alliance_id && c.alliance_name) {
                crumbs.push({ name: `${c.alliance_name}`, item: `/alliance/${c.alliance_id}` })
            }
            if (c?.corporation_id && c.corporation_name) {
                crumbs.push({ name: `${c.corporation_name}`, item: `/corporation/${c.corporation_id}` })
            }
            crumbs.push({ name: c?.name || 'Character', item: `/character/${id}` })
            return crumbs
        })(),
    }))),
    {
        '@type': 'ProfilePage',
        'mainEntity': {
            '@type': 'Person',
            'name': computed(() => char.value?.name || 'Unknown'),
            'url': `https://eve-kill.com/character/${id}`,
            'image': computed(() => char.value ? `/images/characters/${char.value.character_id}/portrait?size=512` : ''),
            'memberOf': computed(() => {
                const parts: { '@type': string; name: string; url: string }[] = []
                const c = char.value
                if (c?.corporation_name) parts.push({ '@type': 'Organization', name: c.corporation_name, url: `https://eve-kill.com/corporation/${c.corporation_id}` })
                if (c?.alliance_name) parts.push({ '@type': 'Organization', name: c.alliance_name, url: `https://eve-kill.com/alliance/${c.alliance_id}` })
                return parts.length ? parts : undefined
            }),
        },
    },
])

// Old portrait — characters born before 2011-01-18 (Incursion 1.1 new character creator)
const hasOldPortrait = computed(() => {
    if (!char.value?.birthday) return false
    return new Date(char.value.birthday) < new Date('2011-01-18')
})

// Security status color
const secColor = (sec: number): string => {
    if (sec >= 5) return 'text-cyan-300'
    if (sec >= 3) return 'text-green-400'
    if (sec >= 1) return 'text-green-300'
    if (sec >= 0) return 'text-yellow-400'
    if (sec >= -2) return 'text-orange-400'
    if (sec >= -5) return 'text-red-400'
    return 'text-red-600'
}

const secBg = (sec: number): string => {
    if (sec >= 5) return 'bg-cyan-500/20 border-cyan-500/30'
    if (sec >= 3) return 'bg-green-500/20 border-green-500/30'
    if (sec >= 1) return 'bg-green-500/15 border-green-500/20'
    if (sec >= 0) return 'bg-yellow-500/20 border-yellow-500/30'
    if (sec >= -2) return 'bg-orange-500/20 border-orange-500/30'
    if (sec >= -5) return 'bg-red-500/20 border-red-500/30'
    return 'bg-red-600/25 border-red-600/40'
}

// Next birthday
const nextBirthday = computed(() => {
    if (!char.value?.birthday) return null
    const born = new Date(char.value.birthday)
    const now = new Date()
    const thisYear = new Date(now.getFullYear(), born.getMonth(), born.getDate())
    const next = now > thisYear
        ? new Date(now.getFullYear() + 1, born.getMonth(), born.getDate())
        : thisYear
    const days = Math.ceil((next.getTime() - now.getTime()) / 86400000)
    return { date: next.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }), days }
})

// Time ago for last active
const timeAgo = (iso: string | null): string => {
    if (!iso) return 'Unknown'
    const diff = Date.now() - new Date(iso).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'Just now'
    if (mins < 60) return `${mins}m ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    if (days < 30) return `${days}d ago`
    return formatDate(iso)
}

const soloKillRatio = computed(() => {
    const s = stats.value
    if (!s || s.kills === 0) return 0
    return Math.round(s.solo_kills / s.kills * 100)
})

const avgKillsPerDay = computed(() => {
    const s = stats.value
    if (!s || !char.value?.birthday) return '0'
    const days = Math.max(1, Math.floor((Date.now() - new Date(char.value.birthday).getTime()) / 86400000))
    return (s.kills / days).toFixed(2)
})

const activityLevel = computed(() => {
    const r = recentStats.value
    if (!r) return { label: 'Unknown', color: 'text-gray-500' }
    const rate = r.kills / 90
    if (rate >= 2) return { label: 'Very High', color: 'text-green-400' }
    if (rate >= 0.5) return { label: 'High', color: 'text-green-300' }
    if (rate >= 0.1) return { label: 'Medium', color: 'text-yellow-400' }
    if (rate > 0) return { label: 'Low', color: 'text-orange-400' }
    return { label: 'Inactive', color: 'text-red-400' }
})
</script>

<template>
    <div>
        <!-- Loading / Header -->
        <EntityHeader v-if="pending" loading />

        <div v-else-if="char">
            <!-- ===== CHARACTER HEADER — MOBILE ===== -->
            <div class="hero-surface glass-panel md:hidden overflow-hidden mb-6"
                :style="accent ? { backgroundImage: `linear-gradient(to bottom, ${accent.soft}, transparent 60%)`, boxShadow: `inset 0 2px 0 0 ${accent.accent}` } : undefined">
                <div class="p-4">
                    <!-- Portrait + name + meta in one compact row -->
                    <div class="flex gap-3">
                        <EntityImageExpand
                            :full-src="`/images/characters/${char.character_id}/portrait?size=512`"
                            :alt="char.name"
                            :alt-src="hasOldPortrait ? `/images/oldcharacters/${char.character_id}` : undefined"
                            primary-label="Current"
                            alt-label="Legacy"
                        >
                            <div class="relative group/portrait w-20 h-20 flex-shrink-0">
                                <EveImage v-if="hasOldPortrait" :src="`/images/oldcharacters/${char.character_id}`" :size="128" :alt="`${char.name} — legacy portrait`" class="absolute inset-0 w-full h-full rounded-lg shadow-lg object-cover z-[1]" sizes="80px" />
                                <EveImage :src="`/images/characters/${char.character_id}/portrait?size=256`" :size="128" :alt="char.name" class="absolute inset-0 w-full h-full rounded-lg shadow-lg object-cover z-[2]" loading="eager" sizes="80px" fetchpriority="high" />
                                <div class="absolute bottom-0.5 right-0.5 px-1 py-0.5 rounded text-fine font-mono font-bold border text-center z-10" :class="[secColor(char.security_status), secBg(char.security_status)]">{{ char.security_status.toFixed(2) }}</div>
                            </div>
                        </EntityImageExpand>
                        <div class="flex-1 min-w-0">
                            <h1 class="text-lg font-bold text-white leading-tight mb-1">{{ char.name }}</h1>
                            <div class="space-y-0.5">
                                <NuxtLink v-if="char.corporation_id" :to="`/corporation/${char.corporation_id}`" class="flex items-center gap-1.5 text-gray-300 hover:text-blue-400 transition-colors">
                                    <EveImage :src="`/images/corporations/${char.corporation_id}/logo?size=64`" :size="16" :alt="char.corporation_name ?? ''" class="w-4 h-4 rounded" />
                                    <span class="text-fine truncate">{{ char.corporation_name || 'Unknown' }}</span>
                                    <span v-if="char.corporation_ticker" class="text-fine text-gray-600">[{{ char.corporation_ticker }}]</span>
                                </NuxtLink>
                                <NuxtLink v-if="char.alliance_id" :to="`/alliance/${char.alliance_id}`" class="flex items-center gap-1.5 text-gray-300 hover:text-blue-400 transition-colors">
                                    <EveImage :src="`/images/alliances/${char.alliance_id}/logo?size=64`" :size="16" :alt="char.alliance_name ?? ''" class="w-4 h-4 rounded" />
                                    <span class="text-fine truncate">{{ char.alliance_name || 'Unknown' }}</span>
                                    <span v-if="char.alliance_ticker" class="text-fine text-gray-600">[{{ char.alliance_ticker }}]</span>
                                </NuxtLink>
                                <NuxtLink v-if="char.faction_id" :to="`/faction/${char.faction_id}`" class="flex items-center gap-1.5 text-gray-400 hover:text-blue-400 transition-colors">
                                    <Icon name="lucide:flag" class="w-4 h-4" />
                                    <span class="text-fine">{{ char.faction_name }}</span>
                                </NuxtLink>
                            </div>
                        </div>
                    </div>

                    <!-- Activity meta — compact row below -->
                    <div class="flex flex-wrap gap-x-4 gap-y-1 mt-3 text-fine">
                        <div v-if="char.last_active" class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:clock" class="w-3 h-3" />
                            <span class="text-gray-300">{{ timeAgo(char.last_active) }}</span>
                        </div>
                        <div v-if="char.birthday" class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:calendar" class="w-3 h-3" />
                            <span class="text-gray-300">{{ formatDate(char.birthday) }}</span>
                            <span v-if="characterAgeYears !== null" class="text-gray-600">({{ characterAgeYears }}y)</span>
                        </div>
                        <div v-if="nextBirthday" class="flex items-center gap-1 text-gray-500">
                            <Icon name="lucide:cake" class="w-3 h-3" />
                            <span class="text-gray-300">{{ nextBirthday.date }}</span>
                            <span class="text-gray-600">({{ nextBirthday.days }}d)</span>
                        </div>
                    </div>
                </div>

                <!-- Stats -->
                <div v-if="stats" class="px-4 pb-4 pt-2 border-t border-white/[0.04]">
                    <EntityStatGrid variant="boxed">
                        <EntityStatBox icon="lucide:swords" icon-color="text-red-500" title="Combat">
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Kills</span><span class="text-fine text-green-400 tabular-nums">{{ stats.kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ stats.losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Efficiency</span><span class="text-fine text-white tabular-nums">{{ stats.efficiency }}%</span></div>
                            <div class="h-1 bg-red-500/20 rounded-full overflow-hidden"><div class="h-full bg-green-500 rounded-full" :style="{ width: effWidth }"></div></div>
                        </EntityStatBox>
                        <EntityStatBox icon="lucide:coins" icon-color="text-yellow-500" title="ISK">
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Destroyed</span><span class="text-fine text-green-400 tabular-nums">{{ formatIsk(stats.isk_destroyed) }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(stats.isk_lost) }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">ISK Eff.</span><span class="text-fine text-white tabular-nums">{{ stats.isk_efficiency }}%</span></div>
                            <div class="h-1 bg-red-500/20 rounded-full overflow-hidden"><div class="h-full bg-green-500 rounded-full" :style="{ width: iskEffWidth }"></div></div>
                        </EntityStatBox>
                        <EntityStatBox icon="lucide:user" icon-color="text-blue-500" title="Solo">
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Solo Kills</span><span class="text-fine text-green-400 tabular-nums">{{ stats.solo_kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Solo Ratio</span><span class="text-fine text-white tabular-nums">{{ soloKillRatio }}%</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Final Blows</span><span class="text-fine text-white tabular-nums">{{ stats.final_blows.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Combat Points</span><span class="text-fine text-white tabular-nums">{{ stats.points.toLocaleString('en-US') }}</span></div>
                        </EntityStatBox>
                        <EntityStatBox icon="lucide:bar-chart-2" icon-color="text-purple-500" title="Other">
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">NPC Losses</span><span class="text-fine text-white tabular-nums">{{ stats.npc_losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">NPC Loss Ratio</span><span class="text-fine text-white tabular-nums">{{ stats.losses > 0 ? Math.round(stats.npc_losses / stats.losses * 100) : 0 }}%</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Avg Kills/Day</span><span class="text-fine text-white tabular-nums">{{ avgKillsPerDay }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Activity</span><span class="text-fine tabular-nums" :class="activityLevel.color">{{ activityLevel.label }}</span></div>
                        </EntityStatBox>
                    </EntityStatGrid>
                </div>
            </div>

            <!-- ===== CHARACTER HEADER — DESKTOP ===== -->
            <div class="hidden md:block">
            <EntityHeader :accent="accent?.accent" :background-image="`/images/characters/${char.character_id}/portrait?size=1024`">
                <template #image>
                    <EntityImageExpand
                        :full-src="`/images/characters/${char.character_id}/portrait?size=512`"
                        :alt="char.name"
                        :alt-src="hasOldPortrait ? `/images/oldcharacters/${char.character_id}` : undefined"
                        primary-label="Current"
                        alt-label="Legacy"
                    >
                        <div class="relative group/portrait w-40 h-40">
                            <EveImage v-if="hasOldPortrait" :src="`/images/oldcharacters/${char.character_id}`" :size="256" :alt="`${char.name} — legacy portrait`" class="absolute inset-0 w-full h-full rounded-lg shadow-lg object-cover z-[1]" sizes="160px" />
                            <EveImage :src="`/images/characters/${char.character_id}/portrait?size=256`" :size="256" :alt="char.name" class="absolute inset-0 w-full h-full rounded-lg shadow-lg object-cover z-[2] transition-opacity duration-300" :class="hasOldPortrait ? 'group-hover/portrait:opacity-0' : ''" loading="lazy" sizes="160px" />
                            <div v-if="char.race_name" class="absolute bottom-1.5 left-1.5 px-2 py-0.5 rounded text-fine font-medium bg-black/80 border border-white/10 text-gray-300 z-10">{{ char.race_name }}</div>
                            <div v-if="char.bloodline_name" class="absolute bottom-8 left-1.5 px-2 py-0.5 rounded text-fine font-medium bg-black/80 border border-white/10 text-gray-300 z-10">{{ char.bloodline_name }}</div>
                            <div class="absolute bottom-1.5 right-1.5 px-2 py-0.5 rounded text-xs font-mono font-bold border min-w-[52px] text-center z-10" :class="[secColor(char.security_status), secBg(char.security_status)]">{{ char.security_status.toFixed(2) }}</div>
                            <div v-if="hasOldPortrait" class="absolute top-1.5 right-1.5 px-1.5 py-0.5 rounded text-fine font-medium bg-amber-500/15 border border-amber-500/20 text-amber-400/70 z-10 group-hover/portrait:opacity-0 transition-opacity duration-300"><Icon name="lucide:history" class="text-fine" /></div>
                        </div>
                    </EntityImageExpand>
                </template>

                <!-- Middle: Character info + Activity -->
                <div class="flex flex-row gap-6">
                    <div class="flex-1 min-w-0">
                        <h1 class="text-3xl font-bold text-white mb-2">{{ char.name }}</h1>
                        <div class="space-y-1.5 mb-4">
                            <NuxtLink v-if="char.corporation_id" :to="`/corporation/${char.corporation_id}`" class="flex items-center gap-2 text-gray-300 hover:text-blue-400 transition-colors">
                                <EveImage :src="`/images/corporations/${char.corporation_id}/logo?size=64`" :size="32" :alt="char.corporation_name ?? ''" class="w-5 h-5 rounded" />
                                <span class="text-xs">{{ char.corporation_name || 'Unknown' }}</span>
                                <span v-if="char.corporation_ticker" class="text-fine text-gray-600">[{{ char.corporation_ticker }}]</span>
                            </NuxtLink>
                            <NuxtLink v-if="char.alliance_id" :to="`/alliance/${char.alliance_id}`" class="flex items-center gap-2 text-gray-300 hover:text-blue-400 transition-colors">
                                <EveImage :src="`/images/alliances/${char.alliance_id}/logo?size=64`" :size="32" :alt="char.alliance_name ?? ''" class="w-5 h-5 rounded" />
                                <span class="text-xs">{{ char.alliance_name || 'Unknown' }}</span>
                                <span v-if="char.alliance_ticker" class="text-fine text-gray-600">[{{ char.alliance_ticker }}]</span>
                            </NuxtLink>
                            <NuxtLink v-if="char.faction_id" :to="`/faction/${char.faction_id}`" class="flex items-center gap-2 text-gray-400 hover:text-blue-400 transition-colors">
                                <Icon name="lucide:flag" class="w-5 h-5" />
                                <span class="text-xs">{{ char.faction_name }}</span>
                            </NuxtLink>
                            <div v-if="char.title" class="flex items-center gap-2 text-gray-500 italic">
                                <Icon name="lucide:award" class="w-5 h-5" />
                                <span class="text-xs">{{ char.title }}</span>
                            </div>
                        </div>
                    </div>
                    <div class="flex-shrink-0 min-w-[200px] space-y-2.5 text-xs">
                        <div v-if="char.last_active">
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs"><Icon name="lucide:clock" class="w-3.5 h-3.5" /> Last Active</div>
                            <div class="text-fine text-gray-300 ml-5">{{ timeAgo(char.last_active) }}</div>
                        </div>
                        <div v-if="char.birthday">
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs"><Icon name="lucide:calendar" class="w-3.5 h-3.5" /> Birthday</div>
                            <div class="text-fine text-gray-300 ml-5">{{ formatDate(char.birthday) }} <span v-if="characterAgeYears !== null" class="text-fine text-gray-500">({{ characterAgeYears }} years old)</span></div>
                        </div>
                        <div v-if="nextBirthday">
                            <div class="flex items-center gap-1.5 text-gray-500 text-xs"><Icon name="lucide:cake" class="w-3.5 h-3.5" /> Next Birthday</div>
                            <div class="text-fine text-gray-300 ml-5">{{ nextBirthday.date }} <span class="text-fine text-gray-500">({{ nextBirthday.days }} days)</span></div>
                        </div>
                    </div>
                </div>

                <template #right>
                    <div class="flex flex-col gap-2 items-end">
                        <EntityRankingBadge :ranking="allTimeRanking" />
                        <NuxtLink v-if="char.corporation_id" :to="`/corporation/${char.corporation_id}`" class="block hover:opacity-80 transition-opacity">
                            <EveImage :src="`/images/corporations/${char.corporation_id}/logo?size=128`" :alt="char.corporation_name ?? ''" class="w-20 h-20 rounded-lg shadow-md" sizes="80px" />
                        </NuxtLink>
                        <NuxtLink v-if="char.alliance_id" :to="`/alliance/${char.alliance_id}`" class="block hover:opacity-80 transition-opacity">
                            <EveImage :src="`/images/alliances/${char.alliance_id}/logo?size=128`" :alt="char.alliance_name ?? ''" class="w-20 h-20 rounded-lg shadow-md" sizes="80px" />
                        </NuxtLink>
                        <NuxtLink v-if="char.faction_id" :to="`/faction/${char.faction_id}`" class="block hover:opacity-80 transition-opacity">
                            <EveImage :src="`/images/corporations/${char.faction_id}/logo?size=128`" :alt="char.faction_name ?? ''" class="w-20 h-20 rounded-lg shadow-md" sizes="80px" />
                        </NuxtLink>
                    </div>
                </template>

                <template v-if="stats" #stats>
                    <EntityStatGrid variant="boxed">
                        <EntityStatBox icon="lucide:swords" icon-color="text-red-500" title="Combat">
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Kills</span><span class="text-fine text-green-400 tabular-nums">{{ stats.kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Losses</span><span class="text-fine text-red-400 tabular-nums">{{ stats.losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Efficiency</span><span class="text-fine text-white tabular-nums">{{ stats.efficiency }}%</span></div>
                            <div class="h-1 bg-red-500/20 rounded-full overflow-hidden"><div class="h-full bg-green-500 rounded-full" :style="{ width: effWidth }"></div></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Danger Ratio</span><span class="text-fine tabular-nums" :class="dangerRatio >= 50 ? 'text-red-400' : 'text-yellow-400'">{{ dangerRatio }}%</span></div>
                        </EntityStatBox>
                        <EntityStatBox icon="lucide:coins" icon-color="text-yellow-500" title="ISK">
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Destroyed</span><span class="text-fine text-green-400 tabular-nums">{{ formatIsk(stats.isk_destroyed) }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Lost</span><span class="text-fine text-red-400 tabular-nums">{{ formatIsk(stats.isk_lost) }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">ISK Efficiency</span><span class="text-fine text-white tabular-nums">{{ stats.isk_efficiency }}%</span></div>
                            <div class="h-1 bg-red-500/20 rounded-full overflow-hidden"><div class="h-full bg-green-500 rounded-full" :style="{ width: iskEffWidth }"></div></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Balance</span><span class="text-fine tabular-nums" :class="stats.isk_destroyed > stats.isk_lost ? 'text-green-400' : 'text-red-400'">{{ stats.isk_destroyed > stats.isk_lost ? '+' : '' }}{{ formatIsk(stats.isk_destroyed - stats.isk_lost) }}</span></div>
                        </EntityStatBox>
                        <EntityStatBox icon="lucide:user" icon-color="text-blue-500" title="Solo">
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Solo Kills</span><span class="text-fine text-green-400 tabular-nums">{{ stats.solo_kills.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Solo Ratio</span><span class="text-fine text-white tabular-nums">{{ soloKillRatio }}%</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Final Blows</span><span class="text-fine text-white tabular-nums">{{ stats.final_blows.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Combat Points</span><span class="text-fine text-white tabular-nums">{{ stats.points.toLocaleString('en-US') }}</span></div>
                        </EntityStatBox>
                        <EntityStatBox icon="lucide:bar-chart-2" icon-color="text-purple-500" title="Other">
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">NPC Losses</span><span class="text-fine text-white tabular-nums">{{ stats.npc_losses.toLocaleString('en-US') }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">NPC Loss Ratio</span><span class="text-fine text-white tabular-nums">{{ stats.losses > 0 ? Math.round(stats.npc_losses / stats.losses * 100) : 0 }}%</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Avg Kills/Day</span><span class="text-fine text-white tabular-nums">{{ avgKillsPerDay }}</span></div>
                            <div class="flex justify-between items-center"><span class="text-fine text-gray-500">Activity</span><span class="text-fine tabular-nums" :class="activityLevel.color">{{ activityLevel.label }}</span></div>
                        </EntityStatBox>
                    </EntityStatGrid>
                </template>
            </EntityHeader>
            </div>
            <MostValuable :api-endpoint="`/api/entity/character/${id}/most-valuable`" />

            <EntityShipClasses entity-type="character" :entity-id="id" />

            <!-- ===== TAB BAR ===== -->
            <EntityTabBar :tabs="tabs" :active-id="activeTab" :accent="accent" @select="setTab" />

            <!-- Dashboard -->
            <div v-if="activeTab === 'dashboard'">
                <LazyCharacterDashboard :character-id="id" :description="char.description" :custom-description-html="char.custom_description_html"
                    :last-active="char.last_active" :lifetime-events="(stats?.kills ?? 0) + (stats?.losses ?? 0)" />
            </div>

            <!-- Top -->
            <div v-if="activeTab === 'top'">
                <LazyEntityTop entity-type="character" :entity-id="id" />
            </div>

            <!-- Corporation History -->
            <div v-if="activeTab === 'history'">
                <LazyCharacterCorporationHistory :history="corpHistory" :queued="corpHistoryQueued" />
            </div>

            <!-- Achievements -->
            <div v-if="activeTab === 'achievements'">
                <LazyCharacterAchievements :character-id="id" />
            </div>

            <div v-if="activeTab === 'battles'">
                <LazyEntityBattles :character-id="id" />
            </div>

            <div v-if="activeTab === 'wars'">
                <LazyEntityWars :character-id="id" />
            </div>

            <div v-if="activeTab === 'campaigns'">
                <LazyEntityCampaigns :character-id="id" />
            </div>

            <!-- Kill tabs (combined/kills/losses) -->
            <LazyEntityKillTabsLayout v-if="['combined', 'kills', 'losses'].includes(activeTab)"
                kind="character" :entity-id="id" :active-tab="activeTab"
                :top-lists="topLists" :killlist-role="killlistRole" :extra-params="killlistExtraParams" />
        </div>
    </div>
</template>
