<script setup lang="ts">
interface AllianceGroup {
    alliance_id: number | null
    alliance_name: string | null
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
    corporations: {
        corporation_id: number
        corporation_name: string | null
        kills: number
        losses: number
        isk_destroyed: number
        isk_lost: number
    }[]
}

const props = defineProps<{
    teams: {
        team_index: number
        total_kills: number
        total_losses: number
        total_isk_destroyed: number
        total_isk_lost: number
        alliance_count: number
        corp_count: number
        alliances: AllianceGroup[]
        dominant_corp_palette?: { main_color?: string; secondary_color?: string; tertiary_color?: string } | null
    }[]
    unsided?: {
        kills: number
        losses: number
        isk_destroyed: number
        isk_lost: number
        alliance_count: number
        corp_count: number
        alliances: AllianceGroup[]
    }
}>()

// Alliances are collapsed by default. Track only the keys the user has
// explicitly opened.
const expanded = ref<Set<string>>(new Set())

const toggleAlliance = (key: string) => {
    if (expanded.value.has(key)) expanded.value.delete(key)
    else expanded.value.add(key)
}

const isExpanded = (key: string) => expanded.value.has(key)

// Alliances first (in their existing rank order), unaffiliated/no-alliance
// group last so independent corps sit at the bottom of each team panel.
const sortedAlliances = (alliances: any[]) => {
    const withAlly = alliances.filter(a => a.alliance_id != null)
    const noAlly = alliances.filter(a => a.alliance_id == null)
    return [...withAlly, ...noAlly]
}

const teamColor = (idx: number) => idx === 0 ? 'red' : 'blue'
const teamBorder = (idx: number) => idx === 0 ? 'border-red-500/20' : 'border-blue-500/20'
const teamLabel = (idx: number) => idx === 0 ? 'Team 1' : 'Team 2'

// Corp-palette team accents; null per team → keep the default red/blue
const teamAccents = computed(() => battleTeamAccents(props.teams.map(t => t.dominant_corp_palette)))

const showUnsided = ref(false)

const allianceImage = (a: any): string => {
    if (a.alliance_id) return `/images/alliances/${a.alliance_id}/logo?size=64`
    if (a.corporations?.[0]) return `/images/corporations/${a.corporations[0].corporation_id}/logo?size=64`
    return ''
}

const allianceName = (a: any): string => {
    return a.alliance_name || a.corporations?.[0]?.corporation_name || 'Unknown'
}

const allianceLink = (a: any): string | null => {
    if (a.alliance_id) return `/alliance/${a.alliance_id}`
    return null
}

</script>

<template>
    <div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div v-for="(team, ti) in teams" :key="ti" class="rounded-lg bg-white/[0.04] border" :class="teamBorder(ti)"
            :style="teamAccents[ti] ? { borderColor: teamAccents[ti]!.border } : undefined">
            <!-- Team header -->
            <div class="px-4 py-2 border-b border-white/[0.06] bg-white/[0.02]">
                <div class="flex items-center justify-between">
                    <div class="text-fine font-bold uppercase tracking-[0.15em]"
                        :class="ti === 0 ? 'text-red-400/80' : 'text-blue-400/80'"
                        :style="teamAccents[ti] ? { color: teamAccents[ti]!.accent } : undefined">
                        {{ teamLabel(ti) }} — {{ team.alliance_count }} alliances · {{ team.corp_count }} corps
                    </div>
                    <div class="text-fine text-gray-500 tabular-nums">
                        {{ team.total_kills }} kills · {{ formatIsk(team.total_isk_destroyed) }} destroyed
                    </div>
                </div>
            </div>

            <!-- Alliance list -->
            <div class="divide-y divide-white/[0.04]">
                <template v-for="(ally, ai) in sortedAlliances(team.alliances)" :key="ai">
                    <!-- Alliance row (collapsed by default) -->
                    <div v-if="ally.alliance_id != null">
                        <button class="flex items-center gap-2.5 w-full px-4 py-2.5 text-left hover:bg-blue-500/[0.04] transition-colors"
                            @click="toggleAlliance(`${ti}-${ai}`)">
                            <img :src="allianceImage(ally)" class="w-8 h-8 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                            <div class="flex-1 min-w-0">
                                <div class="text-sm text-gray-200 truncate">
                                    <NuxtLink v-if="allianceLink(ally)" :to="allianceLink(ally)!" class="hover:text-blue-400" @click.stop>
                                        {{ allianceName(ally) }}
                                    </NuxtLink>
                                    <span v-else>{{ allianceName(ally) }}</span>
                                    <span v-if="ally.corporations.length > 1" class="text-xs text-gray-500 ml-1">({{ ally.corporations.length }})</span>
                                </div>
                                <div class="text-fine text-gray-500">
                                    <span class="text-isk/70 tabular-nums">{{ ally.kills }} kills</span>
                                    <span class="text-gray-600 mx-0.5">·</span>
                                    <span class="text-red-400/70 tabular-nums">{{ ally.losses }} losses</span>
                                    <span class="text-gray-600 mx-0.5">·</span>
                                    <span class="tabular-nums">{{ formatIsk(ally.isk_destroyed) }}</span>
                                </div>
                            </div>
                            <Icon :name="isExpanded(`${ti}-${ai}`) ? 'lucide:chevron-down' : 'lucide:chevron-right'" class="text-xs text-gray-600 flex-shrink-0" />
                        </button>

                        <div v-if="isExpanded(`${ti}-${ai}`) && ally.corporations.length > 0" class="bg-white/[0.01]">
                            <NuxtLink v-for="corp in ally.corporations" :key="corp.corporation_id"
                                :to="`/corporation/${corp.corporation_id}`"
                                class="flex items-center gap-2.5 px-4 pl-10 py-1.5 hover:bg-blue-500/[0.04] transition-colors">
                                <img :src="`/images/corporations/${corp.corporation_id}/logo?size=64`"
                                    class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                                <span class="flex-1 text-xs text-gray-400 truncate">{{ corp.corporation_name || 'Unknown' }}</span>
                                <div class="flex items-center gap-2 text-fine text-gray-500 flex-shrink-0 tabular-nums">
                                    <span class="text-isk/60">{{ corp.kills }}</span>
                                    <span class="text-gray-600">/</span>
                                    <span class="text-red-400/60">{{ corp.losses }}</span>
                                    <span class="text-gray-600 mx-0.5">·</span>
                                    <span>{{ formatIsk(corp.isk_destroyed) }}</span>
                                </div>
                            </NuxtLink>
                        </div>
                    </div>

                    <!-- Independent corps (no alliance) — flat list, always shown at bottom -->
                    <div v-else>
                        <div class="px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-500 bg-white/[0.02]">
                            Independent corporations
                        </div>
                        <NuxtLink v-for="corp in ally.corporations" :key="corp.corporation_id"
                            :to="`/corporation/${corp.corporation_id}`"
                            class="flex items-center gap-2.5 px-4 py-1.5 hover:bg-blue-500/[0.04] transition-colors">
                            <img :src="`/images/corporations/${corp.corporation_id}/logo?size=64`"
                                class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                            <span class="flex-1 text-xs text-gray-400 truncate">{{ corp.corporation_name || 'Unknown' }}</span>
                            <div class="flex items-center gap-2 text-fine text-gray-500 flex-shrink-0 tabular-nums">
                                <span class="text-isk/60">{{ corp.kills }}</span>
                                <span class="text-gray-600">/</span>
                                <span class="text-red-400/60">{{ corp.losses }}</span>
                                <span class="text-gray-600 mx-0.5">·</span>
                                <span>{{ formatIsk(corp.isk_destroyed) }}</span>
                            </div>
                        </NuxtLink>
                    </div>
                </template>

                <div v-if="team.alliances.length === 0" class="px-4 py-6 text-center text-sm text-gray-500">
                    No organizations found
                </div>
            </div>
        </div>
    </div>

    <!-- Unsided participants -->
    <div v-if="unsided && (unsided.alliances.length > 0 || unsided.corp_count > 0)"
         class="glass-panel mt-4">
        <button class="flex items-center justify-between w-full px-4 py-2.5 text-left hover:bg-white/[0.02] transition-colors"
                @click="showUnsided = !showUnsided">
            <div class="flex items-center gap-2">
                <Icon name="lucide:help-circle" class="text-base text-gray-500" />
                <div>
                    <div class="text-sm font-bold text-gray-300">Unsided participants</div>
                    <div class="text-fine text-gray-500">
                        {{ unsided.alliance_count }} alliances · {{ unsided.corp_count }} corps — couldn't be assigned to either team
                    </div>
                </div>
            </div>
            <div class="flex items-center gap-3">
                <div class="text-fine text-gray-500 tabular-nums hidden sm:block">
                    {{ unsided.kills }} kills · {{ unsided.losses }} losses · {{ formatIsk(unsided.isk_destroyed) }} destroyed
                </div>
                <Icon :name="showUnsided ? 'lucide:chevron-down' : 'lucide:chevron-right'" class="text-xs text-gray-600" />
            </div>
        </button>

        <div v-if="showUnsided" class="border-t border-white/[0.06] divide-y divide-white/[0.04]">
            <template v-for="(ally, ai) in sortedAlliances(unsided.alliances)" :key="ai">
                <div v-if="ally.alliance_id != null">
                    <button class="flex items-center gap-2.5 w-full px-4 py-2.5 text-left hover:bg-blue-500/[0.04] transition-colors"
                        @click="toggleAlliance(`u-${ai}`)">
                        <img :src="allianceImage(ally)" class="w-8 h-8 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                        <div class="flex-1 min-w-0">
                            <div class="text-sm text-gray-200 truncate">
                                <NuxtLink v-if="allianceLink(ally)" :to="allianceLink(ally)!" class="hover:text-blue-400" @click.stop>
                                    {{ allianceName(ally) }}
                                </NuxtLink>
                                <span v-else>{{ allianceName(ally) }}</span>
                                <span v-if="ally.corporations.length > 1" class="text-xs text-gray-500 ml-1">({{ ally.corporations.length }})</span>
                            </div>
                            <div class="text-fine text-gray-500">
                                <span class="text-isk/70 tabular-nums">{{ ally.kills }} kills</span>
                                <span class="text-gray-600 mx-0.5">·</span>
                                <span class="text-red-400/70 tabular-nums">{{ ally.losses }} losses</span>
                                <span class="text-gray-600 mx-0.5">·</span>
                                <span class="tabular-nums">{{ formatIsk(ally.isk_destroyed) }}</span>
                            </div>
                        </div>
                        <Icon :name="isExpanded(`u-${ai}`) ? 'lucide:chevron-down' : 'lucide:chevron-right'" class="text-xs text-gray-600 flex-shrink-0" />
                    </button>
                    <div v-if="isExpanded(`u-${ai}`) && ally.corporations.length > 0" class="bg-white/[0.01]">
                        <NuxtLink v-for="corp in ally.corporations" :key="corp.corporation_id"
                            :to="`/corporation/${corp.corporation_id}`"
                            class="flex items-center gap-2.5 px-4 pl-10 py-1.5 hover:bg-blue-500/[0.04] transition-colors">
                            <img :src="`/images/corporations/${corp.corporation_id}/logo?size=64`"
                                class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                            <span class="flex-1 text-xs text-gray-400 truncate">{{ corp.corporation_name || 'Unknown' }}</span>
                            <div class="flex items-center gap-2 text-fine text-gray-500 flex-shrink-0 tabular-nums">
                                <span class="text-isk/60">{{ corp.kills }}</span>
                                <span class="text-gray-600">/</span>
                                <span class="text-red-400/60">{{ corp.losses }}</span>
                                <span class="text-gray-600 mx-0.5">·</span>
                                <span>{{ formatIsk(corp.isk_destroyed) }}</span>
                            </div>
                        </NuxtLink>
                    </div>
                </div>
                <div v-else>
                    <div class="px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-500 bg-white/[0.02]">
                        Independent corporations
                    </div>
                    <NuxtLink v-for="corp in ally.corporations" :key="corp.corporation_id"
                        :to="`/corporation/${corp.corporation_id}`"
                        class="flex items-center gap-2.5 px-4 py-1.5 hover:bg-blue-500/[0.04] transition-colors">
                        <img :src="`/images/corporations/${corp.corporation_id}/logo?size=64`"
                            class="w-6 h-6 rounded flex-shrink-0 bg-gray-900" loading="lazy">
                        <span class="flex-1 text-xs text-gray-400 truncate">{{ corp.corporation_name || 'Unknown' }}</span>
                        <div class="flex items-center gap-2 text-fine text-gray-500 flex-shrink-0 tabular-nums">
                            <span class="text-isk/60">{{ corp.kills }}</span>
                            <span class="text-gray-600">/</span>
                            <span class="text-red-400/60">{{ corp.losses }}</span>
                            <span class="text-gray-600 mx-0.5">·</span>
                            <span>{{ formatIsk(corp.isk_destroyed) }}</span>
                        </div>
                    </NuxtLink>
                </div>
            </template>
        </div>
    </div>
    </div>
</template>
