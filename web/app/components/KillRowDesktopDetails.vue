<script setup lang="ts">
import type { KilllistRow } from '#shared/utils/killlistRow'
defineProps<{ kill: KilllistRow }>()

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
    <!-- Preserve crawlable entity links in SSR; hydrate these cells only on desktop. -->
    <div class="hidden md:contents">
        <!-- Victim -->
        <div class="relative z-10 flex items-center gap-2 min-w-0 pointer-events-none [&_a]:pointer-events-auto">
            <NuxtLink v-if="kill.victim_character_id" :to="`/character/${kill.victim_character_id}`" class="flex-shrink-0 w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
                <EveImage :src="`/images/characters/${kill.victim_character_id}/portrait?size=64`" :alt="kill.victim_character_name || ''" class="w-full h-full object-cover" loading="lazy" />
            </NuxtLink>
            <div v-else class="flex-shrink-0 w-10 h-10 rounded bg-white/[0.04] flex items-center justify-center">
                <Icon name="lucide:building" class="text-sm text-gray-500" />
            </div>
            <div v-if="kill.victim_corporation_id || kill.victim_alliance_id" class="flex-shrink-0 flex flex-col gap-[2px]">
                <NuxtLink v-if="kill.victim_corporation_id" :to="`/corporation/${kill.victim_corporation_id}`" class="w-[19px] h-[19px] rounded-sm overflow-hidden bg-white/[0.04]" :aria-label="`Corporation: ${kill.victim_corporation_name || kill.victim_corporation_id}`">
                    <EveImage :src="`/images/corporations/${kill.victim_corporation_id}/logo?size=32`" :alt="kill.victim_corporation_name || ''" class="w-full h-full object-cover" loading="lazy" />
                </NuxtLink>
                <NuxtLink v-if="kill.victim_alliance_id" :to="`/alliance/${kill.victim_alliance_id}`" class="w-[19px] h-[19px] rounded-sm overflow-hidden bg-white/[0.04]" :aria-label="`Alliance: ${kill.victim_alliance_name || kill.victim_alliance_id}`">
                    <EveImage :src="`/images/alliances/${kill.victim_alliance_id}/logo?size=32`" :alt="kill.victim_alliance_name || ''" class="w-full h-full object-cover" loading="lazy" />
                </NuxtLink>
            </div>
            <div class="min-w-0">
                <NuxtLink v-if="kill.victim_character_id" :to="`/character/${kill.victim_character_id}`" class="block w-fit max-w-full text-xs text-gray-300 hover:text-blue-400 truncate">{{ kill.victim_character_name || 'Unknown' }}</NuxtLink>
                <NuxtLink v-else-if="kill.victim_corporation_id" :to="`/corporation/${kill.victim_corporation_id}`" class="block w-fit max-w-full text-xs text-gray-300 hover:text-blue-400 truncate">{{ kill.victim_corporation_name || 'Unknown' }}</NuxtLink>
                <div v-else class="text-xs text-gray-300 truncate">Unknown</div>
                <NuxtLink v-if="kill.victim_character_name && kill.victim_corporation_id" :to="`/corporation/${kill.victim_corporation_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ kill.victim_corporation_name }}</NuxtLink>
                <NuxtLink v-if="kill.victim_alliance_id" :to="`/alliance/${kill.victim_alliance_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ kill.victim_alliance_name }}</NuxtLink>
            </div>
        </div>

        <!-- Final Blow -->
        <div class="relative z-10 flex items-center gap-2 min-w-0 pointer-events-none [&_a]:pointer-events-auto">
            <NuxtLink v-if="kill.final_blow_character_id" :to="`/character/${kill.final_blow_character_id}`" class="flex-shrink-0 w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
                <EveImage :src="`/images/characters/${kill.final_blow_character_id}/portrait?size=64`" :alt="kill.final_blow_character_name || ''" class="w-full h-full object-cover" loading="lazy" />
            </NuxtLink>
            <NuxtLink v-else-if="kill.final_blow_ship_type_id" :to="`/item/${kill.final_blow_ship_type_id}`" class="flex-shrink-0 w-10 h-10 rounded overflow-hidden bg-white/[0.04]">
                <EveImage :src="`/images/types/${kill.final_blow_ship_type_id}/icon?size=64`" :alt="kill.final_blow_ship_name || 'NPC'" class="w-full h-full object-cover" loading="lazy" />
            </NuxtLink>
            <div v-else class="flex-shrink-0 w-10 h-10 rounded bg-white/[0.04] flex items-center justify-center">
                <Icon name="lucide:crosshair" class="text-sm text-gray-500" />
            </div>
            <div v-if="kill.final_blow_corporation_id || kill.final_blow_alliance_id" class="flex-shrink-0 flex flex-col gap-[2px]">
                <NuxtLink v-if="kill.final_blow_corporation_id" :to="`/corporation/${kill.final_blow_corporation_id}`" class="w-[19px] h-[19px] rounded-sm overflow-hidden bg-white/[0.04]" :aria-label="`Corporation: ${kill.final_blow_corporation_name || kill.final_blow_corporation_id}`">
                    <EveImage :src="`/images/corporations/${kill.final_blow_corporation_id}/logo?size=32`" :alt="kill.final_blow_corporation_name || ''" class="w-full h-full object-cover" loading="lazy" />
                </NuxtLink>
                <NuxtLink v-if="kill.final_blow_alliance_id" :to="`/alliance/${kill.final_blow_alliance_id}`" class="w-[19px] h-[19px] rounded-sm overflow-hidden bg-white/[0.04]" :aria-label="`Alliance: ${kill.final_blow_alliance_name || kill.final_blow_alliance_id}`">
                    <EveImage :src="`/images/alliances/${kill.final_blow_alliance_id}/logo?size=32`" :alt="kill.final_blow_alliance_name || ''" class="w-full h-full object-cover" loading="lazy" />
                </NuxtLink>
            </div>
            <div class="min-w-0">
                <NuxtLink v-if="kill.final_blow_character_id" :to="`/character/${kill.final_blow_character_id}`" class="block w-fit max-w-full text-xs text-gray-300 hover:text-blue-400 truncate">{{ kill.final_blow_character_name || 'Unknown' }}</NuxtLink>
                <NuxtLink v-else-if="kill.final_blow_corporation_id" :to="`/corporation/${kill.final_blow_corporation_id}`" class="block w-fit max-w-full text-xs text-gray-300 hover:text-blue-400 truncate">{{ kill.final_blow_corporation_name || 'Unknown' }}</NuxtLink>
                <NuxtLink v-else-if="kill.final_blow_ship_type_id" :to="`/item/${kill.final_blow_ship_type_id}`" class="block w-fit max-w-full text-xs text-npc/80 hover:text-blue-400 truncate">{{ kill.final_blow_ship_name || 'NPC' }}</NuxtLink>
                <div v-else class="text-xs text-npc/80 truncate">NPC</div>
                <NuxtLink v-if="kill.final_blow_character_name && kill.final_blow_corporation_id" :to="`/corporation/${kill.final_blow_corporation_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ kill.final_blow_corporation_name }}</NuxtLink>
                <NuxtLink v-if="kill.final_blow_alliance_id" :to="`/alliance/${kill.final_blow_alliance_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate">{{ kill.final_blow_alliance_name }}</NuxtLink>
            </div>
        </div>

        <!-- Location -->
        <div class="relative z-10 flex items-center gap-2 min-w-0 pointer-events-none [&_a]:pointer-events-auto">
            <NuxtLink :to="`/system/${kill.solar_system_id}`" class="flex-shrink-0" :aria-label="`System: ${kill.solar_system_name}`">
                <EveImage :src="`/images/systems/${kill.solar_system_id}?size=32`" alt="" class="w-6 h-6 rounded" loading="lazy" />
            </NuxtLink>
            <div class="min-w-0">
                <div class="text-fine truncate">
                    <span :class="secColor(kill.solar_system_security)" class="font-medium tabular-nums">{{ secLabel(kill.solar_system_security) }}</span>
                    <NuxtLink :to="`/system/${kill.solar_system_id}`" class="text-gray-300 hover:text-blue-400 ml-1" :class="pochvenClass(kill.region_id)">{{ kill.solar_system_name }}</NuxtLink>
                </div>
                <NuxtLink v-if="kill.region_id" :to="`/region/${kill.region_id}`" class="block w-fit max-w-full text-fine text-gray-400 hover:text-blue-400 truncate" :class="pochvenClass(kill.region_id)">{{ kill.region_name }}</NuxtLink>
                <div v-else class="text-fine text-gray-400 truncate">{{ kill.region_name }}</div>
            </div>
        </div>

    </div>
</template>
