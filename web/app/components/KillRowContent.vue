<script setup lang="ts">
import type { KilllistRow } from '#shared/utils/killlistRow'
import { killlistClockKey } from '~/utils/killlistClock'

defineProps<{ kill: KilllistRow }>()

// Inject the list's shared clock only when this row hydrates. A changing time
// prop would force every offscreen row to hydrate on the first timer tick.
const now = inject(killlistClockKey, ref(Date.now()))

// Kill time — relative "Xm ago" within the last 30 minutes, otherwise
// the EVE-time clock (UTC HH:MM). Day context comes from the header and
// dividers, so the per-row slot only carries the hour/minute.
const killTime = (dateStr: string): string => {
    const diff = now.value - new Date(dateStr).getTime()
    const mins = Math.floor(diff / 60000)
    if (mins < 1) return 'just now'
    if (mins < 30) return `${mins}m ago`
    return new Date(dateStr).toISOString().slice(11, 16)
}

const secColor = (sec: number | null): string => {
    if (sec === null) return 'text-gray-400'
    if (sec >= 0.5) return 'text-blue-400'
    if (sec > 0.0) return 'text-amber-400'
    return 'text-red-400'
}

const secLabel = (sec: number | null): string => {
    if (sec === null) return '?'
    return sec.toFixed(1)
}
</script>

<template>
    <!-- Stretched row link — covers whitespace so clicks anywhere that
         miss an inner entity link navigate to the killmail. Inner
         NuxtLinks use relative+z-10 to render above and capture clicks. -->
    <NuxtLink
        :to="`/kill/${kill.killmail_id}`"
        class="absolute inset-0 z-0"
        :aria-label="`Killmail: ${kill.ship_name || 'ship'} — ${formatIsk(kill.total_value)} ISK`"
    />

    <!-- Ship -->
    <div class="relative z-10 flex items-center gap-2.5 min-w-0 pointer-events-none [&_a]:pointer-events-auto">
        <NuxtLink v-if="kill.ship_type_id" :to="`/kill/${kill.killmail_id}`" class="flex-shrink-0 w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
            <EveImage :src="`/images/types/${kill.ship_type_id}/icon?size=64`" :alt="kill.ship_name || ''" class="w-full h-full object-cover" loading="lazy" />
        </NuxtLink>
        <div v-else class="flex-shrink-0 w-10 h-10 rounded bg-white/[0.04] flex items-center justify-center">
            <Icon name="lucide:box" class="text-sm text-gray-500" />
        </div>
        <div class="min-w-0">
            <NuxtLink v-if="kill.ship_type_id" :to="`/item/${kill.ship_type_id}`" class="block w-fit max-w-full text-xs text-gray-300 group-hover:text-blue-400 hover:text-blue-400 truncate">{{ kill.ship_name || 'Unknown' }}</NuxtLink>
            <div v-else class="text-xs text-gray-300 truncate">{{ kill.ship_name || 'Unknown' }}</div>
            <NuxtLink v-if="kill.ship_group_id && kill.ship_group_name" :to="`/group/${kill.ship_group_id}`" class="hidden md:block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ kill.ship_group_name }}</NuxtLink>
            <div v-else class="hidden md:block text-fine text-gray-400 truncate">{{ kill.ship_group_name }}</div>
            <div class="hidden md:block text-fine text-isk/70 tabular-nums">{{ formatIsk(kill.total_value) }} ISK</div>
            <div class="md:hidden text-fine text-gray-400 truncate">
                {{ kill.victim_character_name || kill.victim_corporation_name || 'Unknown' }}
                <span v-if="kill.victim_character_name && kill.victim_corporation_name"> · {{ kill.victim_corporation_name }}</span>
            </div>
            <div class="md:hidden text-fine truncate">
                <span :class="secColor(kill.solar_system_security)" class="tabular-nums">{{ secLabel(kill.solar_system_security) }}</span>
                <span class="text-gray-400 ml-0.5" :class="pochvenClass(kill.region_id)">{{ kill.solar_system_name }}</span>
            </div>
        </div>
    </div>

    <LazyKillRowDesktopDetails :kill="kill" hydrate-on-media-query="(min-width: 768px)" />

    <!-- Details -->
    <div class="relative z-10 text-right min-w-0 pointer-events-none [&_a]:pointer-events-auto">
        <div class="md:hidden text-fine text-isk/80 tabular-nums mb-4">{{ formatIsk(kill.total_value) }}</div>
        <div
            class="text-xs text-gray-300 truncate tabular-nums"
            data-allow-mismatch="text"
        >{{ killTime(kill.killmail_time) }}</div>
        <div class="hidden md:flex items-center justify-end gap-1 text-fine text-gray-400">
            <Icon name="lucide:users" class="text-fine" />
            <span class="tabular-nums">{{ kill.attacker_count }}</span>
        </div>
    </div>
</template>
