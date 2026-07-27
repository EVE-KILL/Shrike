<script setup lang="ts">
// All-attackers list — header, one compact row per attacker, and the
// show-more button. Shared by the desktop right column and the mobile Kill
// tab.
//
// Was a 5-cell image strip (portrait / corp / alliance / ship / weapon) above a
// three-line text block, ~110px per attacker. Empty cells rendered as blank
// squares, which is the gap on the right of every row that had no alliance,
// and 32 attackers ran to ~3,500px. One row carries the same information:
// portrait, ship, name, affiliation, damage — with the weapon on the ship's
// tooltip and the damage share as a background fill.
interface AttackerRow {
    character_id: number | null
    character_name: string | null
    corporation_id: number | null
    corporation_name: string | null
    alliance_id: number | null
    alliance_name: string | null
    ship_type_id: number | null
    ship_name: string | null
    weapon_type_id: number | null
    weapon_name: string | null
    damage_done: number
    final_blow: boolean
}

const props = defineProps<{
    attackers: AttackerRow[]
    attackerCount: number
    totalDamage: number
    hasMore: boolean
    remaining: number
    interactive?: boolean
}>()

defineEmits<{ (e: 'show-more'): void }>()

const damagePercent = (dmg: number) => {
    if (!props.totalDamage) return 0
    return ((dmg / props.totalDamage) * 100)
}

/**
 * Bar width is relative to the biggest contributor, not to total damage —
 * scaled against the total, a 30-attacker kill gives everyone a sliver and
 * the bars stop distinguishing anyone.
 */
const topDamageDone = computed(() =>
    Math.max(1, ...props.attackers.map(a => a.damage_done || 0)))

const damageBarWidth = (dmg: number) => `${Math.max(2, ((dmg || 0) / topDamageDone.value) * 100)}%`

</script>

<template>
    <!-- All Attackers list -->
    <div class="px-2 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
        Attackers ({{ attackerCount }})
    </div>
    <div class="space-y-px">
        <div
            v-for="(attacker, idx) in attackers"
            :key="idx"
            class="relative rounded-md overflow-hidden"
            :class="attacker.final_blow ? 'ring-1 ring-amber-500/30' : ''"
        >
            <!-- Damage share as a background fill. It reads as a bar chart
                 without spending a row on one, and it is what makes the list
                 scannable when there are thirty attackers. -->
            <div class="absolute inset-y-0 left-0 pointer-events-none transition-[width] duration-300"
                :class="attacker.final_blow ? 'bg-amber-400/[0.10]' : 'bg-orange-400/[0.07]'"
                :style="{ width: damageBarWidth(attacker.damage_done) }" />

            <div class="relative flex items-center gap-2 px-2 py-1.5"
                :class="attacker.final_blow ? 'bg-amber-500/[0.04]' : 'hover:bg-white/[0.03]'">
                <!-- Pilot -->
                <NuxtLink
                    :to="attacker.character_id ? `/character/${attacker.character_id}` : attacker.corporation_id ? `/corporation/${attacker.corporation_id}` : '#'"
                    class="w-12 h-12 rounded flex-shrink-0 overflow-hidden bg-white/[0.04]">
                    <img v-if="attacker.character_id" :src="`/images/characters/${attacker.character_id}/portrait?size=64`"
                        :alt="attacker.character_name || 'Attacker'" class="w-full h-full object-cover" loading="lazy">
                    <img v-else-if="attacker.ship_type_id" :src="`/images/types/${attacker.ship_type_id}/icon?size=64`"
                        :alt="attacker.ship_name || 'NPC'" class="w-full h-full object-cover" loading="lazy">
                    <span v-else class="w-full h-full flex items-center justify-center">
                        <Icon name="lucide:crosshair" class="text-base text-gray-600" />
                    </span>
                </NuxtLink>

                <!-- Ship, with the weapon on hover rather than its own cell -->
                <Tooltip :text="attacker.weapon_name && attacker.weapon_name !== attacker.ship_name
                    ? `${attacker.ship_name || 'Unknown ship'} — ${attacker.weapon_name}`
                    : (attacker.ship_name || 'Unknown ship')" inline-flex>
                    <NuxtLink :to="attacker.ship_type_id ? `/item/${attacker.ship_type_id}` : '#'"
                        class="w-12 h-12 rounded flex-shrink-0 overflow-hidden bg-white/[0.04] block">
                        <img v-if="attacker.ship_type_id" :src="`/images/types/${attacker.ship_type_id}/icon?size=64`"
                            :alt="attacker.ship_name || 'Ship'" class="w-full h-full object-cover" loading="lazy">
                    </NuxtLink>
                </Tooltip>

                <div class="min-w-0 flex-1">
                    <div class="text-xs truncate">
                        <NuxtLink v-if="attacker.character_id" :to="`/character/${attacker.character_id}`"
                            class="text-gray-200 hover:text-blue-400 transition-colors">{{ attacker.character_name || 'Unknown' }}</NuxtLink>
                        <NuxtLink v-else-if="attacker.corporation_id" :to="`/corporation/${attacker.corporation_id}`"
                            class="text-gray-200 hover:text-blue-400 transition-colors">{{ attacker.corporation_name || 'Unknown' }}</NuxtLink>
                        <span v-else class="text-npc/80">{{ attacker.ship_name || 'NPC' }}</span>
                        <span v-if="attacker.final_blow" class="text-fine text-amber-400 font-bold ml-1">FB</span>
                    </div>
                    <div class="text-fine text-gray-500 truncate">
                        <span v-if="attacker.ship_name" class="text-gray-400">{{ attacker.ship_name }}</span>
                        <template v-if="attacker.character_name && attacker.corporation_name">
                            <span v-if="attacker.ship_name" class="text-gray-700"> · </span>
                            <NuxtLink :to="`/corporation/${attacker.corporation_id}`"
                                class="hover:text-blue-400 transition-colors">{{ attacker.corporation_name }}</NuxtLink>
                        </template>
                    </div>
                    <!-- Alliance gets its own line: on a row this narrow it was
                         the first thing to be truncated away. -->
                    <div v-if="attacker.alliance_name" class="text-fine text-gray-600 truncate">
                        <NuxtLink :to="`/alliance/${attacker.alliance_id}`"
                            class="hover:text-blue-400 transition-colors">{{ attacker.alliance_name }}</NuxtLink>
                    </div>
                </div>

                <div class="flex-shrink-0 text-right">
                    <div class="text-xs text-gray-300 font-medium tabular-nums leading-tight">{{ formatNumber(attacker.damage_done) }}</div>
                    <div class="text-fine text-gray-500 tabular-nums">{{ damagePercent(attacker.damage_done).toFixed(1) }}%</div>
                </div>
            </div>
        </div>

        <!-- Show more button -->
        <button v-if="hasMore" @click="$emit('show-more')"
            class="w-full py-2.5 mt-2 rounded-lg text-xs font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.08] hover:text-blue-400 transition-colors">
            Show more ({{ remaining }} remaining)
        </button>
    </div>
</template>
