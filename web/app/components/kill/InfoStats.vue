<script setup lang="ts">
// Killmail info-box stats + ISK value breakdown — shared by the desktop info
// box (portrait card, right of the fitting wheel) and the mobile character
// information card. `interactive` enables the desktop-only extras: hover link
// styling, the market-group link, and the time-ago ↔ timestamp tooltip.
const props = defineProps<{
    kill: {
        killmail_time: string
        solar_system_id: number
        solar_system_name: string | null
        solar_system_security: number | null
        region_id: number | null
        region_name: string | null
        location: { item_name: string | null; distance: number } | null
        attacker_count: number
        total_value: number
        fitted_value: number
        dropped_value: number
        destroyed_value: number
        victim: {
            ship_type_id: number | null
            ship_name: string | null
            ship_group_name: string | null
            ship_market_path?: string | null
            ship_price: number
            damage_taken: number
        }
    }
    /**
     * Value of everything in a fitting slot — deliberately NOT the killmail's
     * `fitted_value`, which folds the hull in and so made the Fitted row read
     * as far more than the modules were worth. Passed in rather than derived
     * here so this and the item table's footer quote the same figure; the two
     * are computed from item prices, which differ slightly from the killmail's
     * stored aggregates.
     */
    fittedItemsValue: number
    interactive?: boolean
}>()

const formatDistance = (meters: number) => {
    const AU = 149_597_870_700
    if (meters >= AU / 10) return `${(meters / AU).toLocaleString('en-US', { maximumFractionDigits: 2 })} AU`
    return `${Math.round(meters / 1000).toLocaleString('en-US')} km`
}

const secColor = (sec: number | null): string => {
    if (sec === null) return 'text-gray-500'
    if (sec >= 0.5) return 'text-blue-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

const timeAgo = (dateStr: string): string => {
    const diff = Date.now() - new Date(dateStr).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 60) return `${mins}m ago`
    const hours = Math.floor(mins / 60)
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    return `${days}d ago`
}

const formatTimestamp = (dateStr: string): string => {
    const d = new Date(dateStr)
    const day = d.getUTCDate()
    const month = d.toLocaleString('en-US', { month: 'short', timeZone: 'UTC' })
    const year = d.getUTCFullYear()
    const hh = String(d.getUTCHours()).padStart(2, '0')
    const mm = String(d.getUTCMinutes()).padStart(2, '0')
    const ss = String(d.getUTCSeconds()).padStart(2, '0')
    return `${day} ${month} ${year}, ${hh}:${mm}:${ss} EVE`
}

// Destroyed and dropped are the two halves of what came off the wreck; the rule
// between the two blocks below is drawn in their proportion, so the divider
// doubles as a reading of the numbers underneath it.
const lossTotal = computed(() => props.kill.destroyed_value + props.kill.dropped_value)
const destroyedPct = computed(() => lossTotal.value > 0 ? (props.kill.destroyed_value / lossTotal.value) * 100 : 100)
</script>

<template>
    <!-- Stats -->
    <div class="bg-white/[0.04] px-4 py-3 space-y-2 text-xs">
        <div class="flex justify-between items-start gap-2">
            <span class="text-gray-500 flex-shrink-0">Ship</span>
            <span class="flex flex-wrap justify-end items-center gap-x-1 min-w-0 text-right">
                <img :src="`/images/types/${kill.victim.ship_type_id}/icon?size=32`" alt=""
                    class="w-4 h-4 rounded flex-shrink-0" loading="lazy">
                <NuxtLink :to="`/item/${kill.victim.ship_type_id}`" :class="interactive ? 'text-gray-300 hover:text-blue-400 transition-colors break-words' : 'text-gray-300 break-words'">{{ kill.victim.ship_name }}</NuxtLink>
                <template v-if="kill.victim.ship_group_name">
                    <NuxtLink v-if="interactive" :to="kill.victim.ship_market_path || '/market/ships'" class="text-gray-500 hover:text-blue-400 transition-colors break-words">({{ kill.victim.ship_group_name }})</NuxtLink>
                    <span v-else class="text-gray-500 break-words">({{ kill.victim.ship_group_name }})</span>
                </template>
            </span>
        </div>
        <div class="flex justify-between items-center"><span class="text-gray-500">System</span><span class="flex items-center gap-1.5"><img :src="`/images/systems/${kill.solar_system_id}?size=32`" alt="" class="w-4 h-4 rounded" loading="lazy"><NuxtLink :to="`/system/${kill.solar_system_id}`" :class="interactive ? 'hover:text-blue-400 transition-colors' : undefined"><span :class="secColor(kill.solar_system_security)" class="font-medium tabular-nums">{{ kill.solar_system_security?.toFixed(1) }}</span><span class="text-gray-300 ml-1" :class="pochvenClass(kill.region_id)">{{ kill.solar_system_name }}</span></NuxtLink></span></div>
        <div class="flex justify-between items-center"><span class="text-gray-500">Region</span><span class="flex items-center gap-1.5"><img v-if="kill.region_id" :src="`/images/regions/${kill.region_id}?size=32`" alt="" class="w-4 h-4 rounded" loading="lazy"><NuxtLink v-if="kill.region_id" :to="`/region/${kill.region_id}`" :class="[interactive ? 'text-gray-300 hover:text-blue-400 transition-colors' : 'text-gray-300', pochvenClass(kill.region_id)]">{{ kill.region_name }}</NuxtLink><span v-else class="text-gray-300">{{ kill.region_name }}</span></span></div>
        <div v-if="kill.location?.item_name" class="flex justify-between items-start gap-2">
            <span class="text-gray-500 flex-shrink-0">Location</span>
            <Tooltip :text="kill.location.item_name" class="min-w-0 text-right">
                <span class="text-gray-300 break-words">{{ kill.location.item_name }}</span>
            </Tooltip>
        </div>
        <div v-if="kill.location?.item_name" class="flex justify-between items-start gap-2"><span class="text-gray-500 flex-shrink-0">Distance</span><span class="text-gray-300 text-right tabular-nums">{{ formatDistance(kill.location.distance) }}</span></div>
        <div class="flex justify-between">
            <span class="text-gray-500">Time</span>
            <!-- Desktop showed the exact stamp only while the cursor sat on the
                 row, which nothing advertised. The tooltip says the same thing
                 and looks like it has something to say. -->
            <Tooltip v-if="interactive" :text="formatTimestamp(kill.killmail_time)">
                <span class="text-gray-300 cursor-help decoration-dotted decoration-gray-600 underline underline-offset-4">{{ timeAgo(kill.killmail_time) }}</span>
            </Tooltip>
            <span v-else class="text-gray-300">{{ formatTimestamp(kill.killmail_time) }}</span>
        </div>
        <div class="flex justify-between"><span class="text-gray-500">Damage Taken</span><span class="text-red-400 tabular-nums">{{ formatNumber(kill.victim.damage_taken) }}</span></div>
        <div class="flex justify-between"><span class="text-gray-500">Attackers</span><span class="text-gray-300 tabular-nums">{{ kill.attacker_count }}</span></div>
    </div>

    <!-- Divider drawn in the destroyed/dropped proportion -->
    <Tooltip block :disabled="lossTotal <= 0"
        :text="`Destroyed ${formatIsk(kill.destroyed_value)} ISK · Dropped ${formatIsk(kill.dropped_value)} ISK`">
        <span class="flex h-0.5 w-full bg-white/[0.08]" :class="lossTotal > 0 ? 'cursor-help' : ''">
            <span v-if="lossTotal > 0" class="bg-red-400/50" :style="{ width: `${destroyedPct}%` }" />
            <span v-if="lossTotal > 0" class="bg-isk/50 flex-1" />
        </span>
    </Tooltip>

    <div class="bg-white/[0.04] px-4 py-3 space-y-1.5 text-xs">
        <div class="flex justify-between items-center">
            <span class="flex items-center gap-1.5 text-gray-600"><span class="w-1.5 h-1.5 rounded-full bg-red-400/50" />Destroyed</span>
            <span class="text-red-400/80 tabular-nums">{{ formatIsk(kill.destroyed_value) }}</span>
        </div>
        <div class="flex justify-between items-center">
            <span class="flex items-center gap-1.5 text-gray-600"><span class="w-1.5 h-1.5 rounded-full bg-isk/50" />Dropped</span>
            <span class="text-isk/80 tabular-nums">{{ formatIsk(kill.dropped_value) }}</span>
        </div>
        <!-- Hidden when empty: a pod has neither, and "Fitted 0 / Ship 0" is
             noise on the kill where the block matters least. -->
        <div v-if="fittedItemsValue" class="flex justify-between items-center">
            <!-- No dots below: these are a different cut of the loss, not
                 further slices of the destroyed/dropped rule above. The spacer
                 keeps the labels aligned without claiming otherwise. -->
            <span class="flex items-center gap-1.5 text-gray-600"><span class="w-1.5 h-1.5" />Fitted</span>
            <span class="text-gray-400 tabular-nums">{{ formatIsk(fittedItemsValue) }}</span>
        </div>
        <div v-if="kill.victim.ship_price" class="flex justify-between items-center">
            <span class="flex items-center gap-1.5 text-gray-600"><span class="w-1.5 h-1.5" />Ship</span>
            <span class="text-gray-400 tabular-nums">{{ formatIsk(kill.victim.ship_price) }}</span>
        </div>
        <div class="flex justify-between border-t border-white/[0.08] pt-1.5 mt-1.5">
            <span class="text-gray-400 font-medium">Total Value</span>
            <span class="text-white font-semibold tabular-nums">{{ formatIsk(kill.total_value) }} ISK</span>
        </div>
    </div>
</template>
