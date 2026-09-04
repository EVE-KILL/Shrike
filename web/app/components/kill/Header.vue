<script setup lang="ts">
import type { KillToolContext } from '~/composables/useKillTools'

/**
 * Killmail headline.
 *
 * The page previously opened with eleven outbound tool links and no title —
 * the least important thing on the page held the most prominent position, and
 * nothing actually stated what happened. You had to assemble it from a
 * label/value grid. This says it in a sentence, puts the money and the state
 * badges where the eye lands first, and tucks the external tools behind a
 * toggle.
 */
const props = defineProps<{
    kill: {
        killmail_id: number
        killmail_time: string
        solar_system_id: number
        solar_system_name: string | null
        solar_system_security: number | null
        region_id: number | null
        region_name: string | null
        total_value: number
        destroyed_value: number
        dropped_value: number
        fitted_value: number
        total_damage: number
        attacker_count: number
        points: number | null
        is_solo: boolean
        is_npc: boolean
        victim: {
            ship_type_id: number | null
            ship_name: string | null
            character_name: string | null
            corporation_name: string | null
        }
    }
    finalBlow?: { character_name: string | null } | null
    battleLink?: string | null
    /** Other killmails from the same engagement, if any. */
    siblings?: { killmail_id: number; ship_type_id: number | null; ship_name: string | null; total_value: number }[]
    /** Everything the "Open in" menu needs to build its destinations. */
    toolContext: KillToolContext
}>()

const siblingsOpen = ref(false)
const siblingList = computed(() => props.siblings ?? [])

const victimName = computed(() =>
    props.kill.victim.character_name || props.kill.victim.corporation_name || 'Unknown pilot')

/** The one-line version of the whole page. */
const summary = computed(() => {
    const ship = props.kill.victim.ship_name || 'Unknown ship'
    const where = props.kill.solar_system_name || 'unknown space'
    const n = props.kill.attacker_count
    const by = props.kill.is_solo
        ? `solo by ${props.finalBlow?.character_name || 'one attacker'}`
        : `by ${n} attacker${n === 1 ? '' : 's'}`
    return `${ship} destroyed in ${where} ${by}`
})

const secColor = (sec: number | null): string => {
    if (sec === null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-blue-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

// EVE time, spelled out. The info box shows a relative stamp; the headline
// should carry the absolute one so the page states when this happened without
// a hover.
const absoluteTime = computed(() => {
    const d = new Date(props.kill.killmail_time)
    if (Number.isNaN(d.getTime())) return ''
    const month = d.toLocaleString('en-US', { month: 'short', timeZone: 'UTC' })
    return `${d.getUTCDate()} ${month} ${d.getUTCFullYear()}, ${formatEveTime(d)} EVE`
})

const relativeTime = computed(() => {
    const diff = Date.now() - new Date(props.kill.killmail_time).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 60) return `${mins}m ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours}h ago`
    return `${Math.floor(hours / 24)}d ago`
})

/**
 * Destroyed and dropped are the split that sums to the loss; fitted is a
 * different cut of the same hull (what was bolted on, destroyed or not), so
 * it is shown but not treated as a third slice of the bar.
 */
const tiles = computed(() => [
    { label: 'Total value', value: `${formatIsk(props.kill.total_value)} ISK`, color: 'text-white', icon: 'lucide:coins' },
    { label: 'Destroyed', value: formatIsk(props.kill.destroyed_value), color: 'text-red-400', icon: 'lucide:flame' },
    { label: 'Dropped', value: formatIsk(props.kill.dropped_value), color: 'text-isk', icon: 'lucide:package' },
    { label: 'Fitted', value: formatIsk(props.kill.fitted_value), color: 'text-gray-300', icon: 'lucide:wrench' },
    { label: 'Damage', value: formatNumber(props.kill.total_damage), color: 'text-orange-400', icon: 'lucide:zap' },
    { label: 'Attackers', value: formatNumber(props.kill.attacker_count), color: 'text-blue-400', icon: 'lucide:users' },
])

const lossTotal = computed(() => props.kill.destroyed_value + props.kill.dropped_value)
</script>

<template>
    <div class="hero-surface glass-panel p-4 mb-4">
        <div class="flex items-start justify-between gap-4 flex-wrap">
            <div class="min-w-0">
                <div class="min-w-0">
                    <h1 class="text-lg font-bold text-white leading-tight">{{ summary }}</h1>
                    <div class="flex items-center gap-x-3 gap-y-1 mt-1.5 flex-wrap text-fine text-gray-500">
                        <span class="flex items-center gap-1.5 min-w-0">
                            <Icon name="lucide:user-round" class="text-fine text-gray-600 flex-shrink-0" />
                            <span class="text-gray-400 truncate">{{ victimName }}</span>
                        </span>
                        <NuxtLink :to="`/system/${kill.solar_system_id}`" class="flex items-center gap-1.5 hover:text-blue-400 transition-colors">
                            <Icon name="lucide:map-pin" class="text-fine text-gray-600" />
                            <span class="tabular-nums" :class="secColor(kill.solar_system_security)">{{ kill.solar_system_security?.toFixed(1) }}</span>
                            <span class="text-gray-400" :class="pochvenClass(kill.region_id)">{{ kill.solar_system_name }}</span>
                            <span v-if="kill.region_name" class="text-gray-600">· {{ kill.region_name }}</span>
                        </NuxtLink>
                        <span class="flex items-center gap-1.5 tabular-nums" v-tooltip="relativeTime">
                            <Icon name="lucide:clock" class="text-fine text-gray-600" />
                            {{ absoluteTime }}
                            <span class="text-gray-600">({{ relativeTime }})</span>
                        </span>
                    </div>
                </div>
            </div>

            <div class="flex items-center gap-1.5 flex-wrap min-w-0">
                <span v-if="kill.is_solo" class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-medium bg-amber-500/15 text-amber-400">
                    <Icon name="lucide:swords" class="text-fine" /> Solo
                </span>
                <span v-if="kill.is_npc" class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-medium bg-white/[0.06] text-gray-400">
                    <Icon name="lucide:bot" class="text-fine" /> NPC
                </span>
                <!-- Spell out the unit: a bare star and a number reads as a rating
                     or a favourite, and a tooltip is not an explanation. -->
                <Tooltip v-if="kill.points" inline-flex
                    text="Killboard points — how much this kill is worth, scaled by ship class and the odds faced">
                    <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-fine font-medium bg-purple-500/15 text-purple-300 cursor-help">
                        <Icon name="lucide:star" class="text-fine" />
                        {{ formatNumber(kill.points) }} {{ kill.points === 1 ? 'pt' : 'pts' }}
                    </span>
                </Tooltip>
                <NuxtLink v-if="battleLink" :to="battleLink"
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-fine font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors">
                    <Icon name="lucide:swords" class="text-fine" /> Battle report
                </NuxtLink>
                <!-- Other killmails from the same engagement -->
                <NuxtLink v-if="siblingList.length === 1"
                    :to="`/kill/${siblingList[0]!.killmail_id}`"
                    class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-fine font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors">
                    <img v-if="siblingList[0]!.ship_type_id"
                        :src="`/images/types/${siblingList[0]!.ship_type_id}/icon?size=32`"
                        class="w-3.5 h-3.5 rounded-sm" loading="lazy" alt="">
                    {{ siblingList[0]!.ship_name || 'Related kill' }}
                </NuxtLink>
                <div v-else-if="siblingList.length > 1" class="relative">
                    <button type="button"
                        class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-fine font-medium border transition-colors cursor-pointer"
                        :class="siblingsOpen
                            ? 'bg-blue-500/15 text-blue-400 border-blue-500/30'
                            : 'text-gray-400 bg-white/[0.04] border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400'"
                        @click="siblingsOpen = !siblingsOpen">
                        <Icon name="lucide:layers" class="text-fine" />
                        Other kills
                        <span class="text-gray-600 tabular-nums">{{ siblingList.length }}</span>
                        <Icon name="lucide:chevron-down" class="text-fine transition-transform" :class="siblingsOpen ? 'rotate-180' : ''" />
                    </button>
                    <div v-if="siblingsOpen"
                        class="absolute right-0 z-50 mt-1 w-64 max-h-80 overflow-y-auto rounded-lg border border-white/[0.10] bg-[#141414]/97 backdrop-blur-xl shadow-2xl shadow-black/60 p-1"
                        @click="siblingsOpen = false">
                        <NuxtLink v-for="s in siblingList" :key="s.killmail_id" :to="`/kill/${s.killmail_id}`"
                            class="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-blue-500/[0.06] transition-colors">
                            <img v-if="s.ship_type_id" :src="`/images/types/${s.ship_type_id}/icon?size=32`"
                                class="w-5 h-5 rounded flex-shrink-0 bg-gray-900" loading="lazy" alt="">
                            <span class="text-xs text-gray-300 truncate flex-1">{{ s.ship_name || 'Unknown' }}</span>
                            <span class="text-fine text-yellow-400/80 tabular-nums flex-shrink-0">{{ formatIsk(s.total_value) }}</span>
                        </NuxtLink>
                    </div>
                </div>

                <KillToolsMenu :context="toolContext" />
            </div>
        </div>

        <!-- Where the ISK went -->
        <Tooltip v-if="lossTotal > 0" block class="mt-3"
            :text="`Destroyed ${formatIsk(kill.destroyed_value)} ISK · Dropped ${formatIsk(kill.dropped_value)} ISK`">
            <span class="flex h-1 rounded-full overflow-hidden bg-white/[0.06] cursor-help">
                <span class="bg-red-500/70" :style="{ width: `${(kill.destroyed_value / lossTotal) * 100}%` }" />
                <span class="bg-isk/70" :style="{ width: `${(kill.dropped_value / lossTotal) * 100}%` }" />
            </span>
        </Tooltip>

        <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2 mt-3">
            <div v-for="tile in tiles" :key="tile.label"
                class="rounded-lg bg-white/[0.03] border border-white/[0.06] px-3 py-2 text-center">
                <div class="text-sm font-bold tabular-nums" :class="tile.color">{{ tile.value }}</div>
                <div class="flex items-center justify-center gap-1.5 text-fine text-gray-500 uppercase tracking-wider mt-0.5">
                    <Icon :name="tile.icon" class="text-fine text-gray-600" />{{ tile.label }}
                </div>
            </div>
        </div>

    </div>
</template>
