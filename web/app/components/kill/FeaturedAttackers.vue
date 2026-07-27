<script setup lang="ts">
// Featured attackers — Final Blow + Top Damage cards, side by side when both
// exist. Shared by the desktop right column and the mobile Kill tab.
// `interactive` enables the desktop-only extras: hover scale/link styling,
// weapon links, the ship line under corp-only names, NPC ship links, and the
// larger solo-FB portrait (w-20 vs w-16).
interface FeaturedAttacker {
    character_id: number | null
    character_name: string | null
    corporation_id: number | null
    corporation_name: string | null
    ship_type_id: number | null
    ship_name: string | null
    weapon_type_id: number | null
    weapon_name: string | null
    damage_done: number
}

const props = defineProps<{
    finalBlow: FeaturedAttacker | null
    topDamage: FeaturedAttacker | null
    totalDamage: number
    interactive?: boolean
}>()

const isNpcAttacker = (a: any) => !a.character_id && !a.corporation_id
const attackerShipImage = (a: any, size: number) => {
    if (!a?.ship_type_id) return ''
    // NPC/environment "ships" don't have overlay renders — use icon
    return isNpcAttacker(a)
        ? `/images/types/${a.ship_type_id}/icon?size=${size}`
        : `/images/types/${a.ship_type_id}/overlayrender?size=${size}`
}

const damagePercent = (dmg: number) => {
    if (!props.totalDamage) return 0
    return ((dmg / props.totalDamage) * 100)
}
</script>

<template>
    <div class="grid gap-2 mb-3" :class="finalBlow && topDamage ? 'grid-cols-2' : 'grid-cols-1'">
        <!-- Final Blow -->
        <div v-if="finalBlow" class="relative rounded-xl border border-amber-500/20 overflow-hidden group">
            <div class="relative">
                <img v-if="finalBlow.ship_type_id"
                    :src="attackerShipImage(finalBlow, 512)"
                    :alt="`${finalBlow.ship_name || 'Ship'} — Final Blow`"
                    :class="interactive ? 'w-full h-auto object-contain' : 'w-full h-auto object-contain'"
                    loading="lazy">
                <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent"></div>
                <div class="absolute top-2 left-2 px-2 py-0.5 rounded bg-amber-500/20 backdrop-blur-sm">
                    <span class="text-fine font-bold uppercase tracking-[0.15em] text-amber-400">FB</span>
                </div>
                <!-- Portrait -->
                <NuxtLink v-if="finalBlow.character_id" :to="`/character/${finalBlow.character_id}`"
                    :class="[
                        interactive
                            ? 'absolute top-2 right-2 rounded-lg overflow-hidden border border-white/[0.1] hover:border-amber-500/40 transition-colors'
                            : 'absolute top-2 right-2 rounded-lg overflow-hidden border border-white/[0.1]',
                        topDamage ? 'w-10 h-10' : (interactive ? 'w-20 h-20' : 'w-16 h-16'),
                    ]">
                    <img :src="`/images/characters/${finalBlow.character_id}/portrait?size=128`" :alt="finalBlow.character_name || ''" class="w-full h-full object-cover">
                </NuxtLink>
                <div class="absolute inset-x-0 bottom-0 p-2.5 space-y-0.5">
                    <template v-if="finalBlow.character_id">
                        <NuxtLink :to="`/character/${finalBlow.character_id}`" :class="interactive ? 'text-sm text-white font-semibold drop-shadow-lg block truncate hover:text-amber-400 transition-colors' : 'text-sm text-white font-semibold drop-shadow-lg block truncate'">{{ finalBlow.character_name || 'Unknown' }}</NuxtLink>
                        <div class="text-fine drop-shadow truncate">
                            <NuxtLink v-if="finalBlow.ship_type_id" :to="`/item/${finalBlow.ship_type_id}`" :class="interactive ? 'text-gray-300/80 hover:text-blue-400 transition-colors' : 'text-gray-300/80'">{{ finalBlow.ship_name }}</NuxtLink>
                            <template v-if="interactive && finalBlow.weapon_name"> — <NuxtLink v-if="finalBlow.weapon_type_id" :to="`/item/${finalBlow.weapon_type_id}`" class="text-gray-400 hover:text-blue-400 transition-colors">{{ finalBlow.weapon_name }}</NuxtLink></template>
                        </div>
                    </template>
                    <template v-else-if="finalBlow.corporation_id">
                        <NuxtLink :to="`/corporation/${finalBlow.corporation_id}`" :class="interactive ? 'text-sm text-white font-semibold drop-shadow-lg block truncate hover:text-amber-400 transition-colors' : 'text-sm text-white font-semibold drop-shadow-lg block truncate'">{{ finalBlow.corporation_name || 'Unknown' }}</NuxtLink>
                        <div v-if="interactive" class="text-fine drop-shadow truncate">
                            <NuxtLink v-if="finalBlow.ship_type_id" :to="`/item/${finalBlow.ship_type_id}`" class="text-gray-300/80 hover:text-blue-400 transition-colors">{{ finalBlow.ship_name }}</NuxtLink>
                        </div>
                    </template>
                    <template v-else>
                        <template v-if="interactive">
                            <NuxtLink v-if="finalBlow.ship_type_id" :to="`/item/${finalBlow.ship_type_id}`" class="text-sm text-amber-300 font-semibold drop-shadow-lg block truncate hover:text-blue-400 transition-colors">{{ finalBlow.ship_name || 'NPC' }}</NuxtLink>
                            <span v-else class="text-sm text-amber-300 font-semibold drop-shadow-lg block truncate">NPC</span>
                        </template>
                        <span v-else class="text-sm text-amber-300 font-semibold drop-shadow-lg block truncate">{{ finalBlow.ship_name || 'NPC' }}</span>
                    </template>
                    <div class="flex items-center gap-1 text-fine">
                        <Icon name="lucide:zap" class="text-red-400 text-fine" />
                        <span class="text-red-400 font-medium tabular-nums drop-shadow">{{ formatNumber(finalBlow.damage_done) }}</span>
                        <span class="text-gray-400 tabular-nums drop-shadow">({{ damagePercent(finalBlow.damage_done).toFixed(1) }}%)</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Top Damage -->
        <div v-if="topDamage" class="relative rounded-xl border border-red-500/20 overflow-hidden group">
            <div class="relative">
                <img v-if="topDamage.ship_type_id"
                    :src="attackerShipImage(topDamage, 512)"
                    :alt="`${topDamage.ship_name || 'Ship'} — Top Damage`"
                    :class="interactive ? 'w-full h-auto object-contain' : 'w-full h-auto object-contain'"
                    loading="lazy">
                <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent"></div>
                <div class="absolute top-2 left-2 px-2 py-0.5 rounded bg-red-500/20 backdrop-blur-sm">
                    <span class="text-fine font-bold uppercase tracking-[0.15em] text-red-400">TD</span>
                </div>
                <!-- Portrait -->
                <NuxtLink v-if="topDamage.character_id" :to="`/character/${topDamage.character_id}`"
                    :class="interactive
                        ? 'absolute top-2 right-2 w-10 h-10 rounded-lg overflow-hidden border border-white/[0.1] hover:border-red-500/40 transition-colors'
                        : 'absolute top-2 right-2 w-10 h-10 rounded-lg overflow-hidden border border-white/[0.1]'">
                    <img :src="`/images/characters/${topDamage.character_id}/portrait?size=128`" :alt="topDamage.character_name || ''" class="w-full h-full object-cover">
                </NuxtLink>
                <div class="absolute inset-x-0 bottom-0 p-2.5 space-y-0.5">
                    <template v-if="topDamage.character_id">
                        <NuxtLink :to="`/character/${topDamage.character_id}`" :class="interactive ? 'text-sm text-white font-semibold drop-shadow-lg block truncate hover:text-red-400 transition-colors' : 'text-sm text-white font-semibold drop-shadow-lg block truncate'">{{ topDamage.character_name || 'Unknown' }}</NuxtLink>
                        <div class="text-fine drop-shadow truncate">
                            <NuxtLink v-if="topDamage.ship_type_id" :to="`/item/${topDamage.ship_type_id}`" :class="interactive ? 'text-gray-300/80 hover:text-blue-400 transition-colors' : 'text-gray-300/80'">{{ topDamage.ship_name }}</NuxtLink>
                            <template v-if="interactive && topDamage.weapon_name"> — <NuxtLink v-if="topDamage.weapon_type_id" :to="`/item/${topDamage.weapon_type_id}`" class="text-gray-400 hover:text-blue-400 transition-colors">{{ topDamage.weapon_name }}</NuxtLink></template>
                        </div>
                    </template>
                    <template v-else-if="topDamage.corporation_id">
                        <NuxtLink :to="`/corporation/${topDamage.corporation_id}`" :class="interactive ? 'text-sm text-white font-semibold drop-shadow-lg block truncate hover:text-red-400 transition-colors' : 'text-sm text-white font-semibold drop-shadow-lg block truncate'">{{ topDamage.corporation_name || 'Unknown' }}</NuxtLink>
                        <div v-if="interactive" class="text-fine drop-shadow truncate">
                            <NuxtLink v-if="topDamage.ship_type_id" :to="`/item/${topDamage.ship_type_id}`" class="text-gray-300/80 hover:text-blue-400 transition-colors">{{ topDamage.ship_name }}</NuxtLink>
                        </div>
                    </template>
                    <template v-else>
                        <template v-if="interactive">
                            <NuxtLink v-if="topDamage.ship_type_id" :to="`/item/${topDamage.ship_type_id}`" class="text-sm text-red-300 font-semibold drop-shadow-lg block truncate hover:text-blue-400 transition-colors">{{ topDamage.ship_name || 'NPC' }}</NuxtLink>
                            <span v-else class="text-sm text-red-300 font-semibold drop-shadow-lg block truncate">NPC</span>
                        </template>
                        <span v-else class="text-sm text-red-300 font-semibold drop-shadow-lg block truncate">{{ topDamage.ship_name || 'NPC' }}</span>
                    </template>
                    <div class="flex items-center gap-1 text-fine">
                        <Icon name="lucide:zap" class="text-red-400 text-fine" />
                        <span class="text-red-400 font-medium tabular-nums drop-shadow">{{ formatNumber(topDamage.damage_done) }}</span>
                        <span class="text-gray-400 tabular-nums drop-shadow">({{ damagePercent(topDamage.damage_done).toFixed(1) }}%)</span>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>
