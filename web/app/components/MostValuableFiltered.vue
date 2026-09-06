<script setup lang="ts">
const props = defineProps<{
    killType: string
    apiEndpoint?: string
}>()

interface MvKill {
    killmail_id: number
    ship_type_id: number
    ship_name: string
    total_value: number
    victim_character_id: number | null
    victim_character_name: string | null
    victim_corporation_name: string | null
    victim_alliance_name: string | null
}

const endpoint = computed(() => props.apiEndpoint || '/api/kills/most-valuable')
const { data, pending } = await useApiFetch<{ entries: MvKill[] }>(endpoint, {
    params: { type: props.killType, limit: 8, days: 7 },
    default: () => ({ entries: [] }),
})

const items = computed(() => data.value?.entries || [])

</script>

<template>
    <div class="glass-panel p-3 mb-6">
    <div v-if="pending" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-8 gap-2">
        <div v-for="i in 8" :key="i" class="aspect-square rounded-lg bg-white/[0.02] animate-pulse"></div>
    </div>

    <div v-else-if="items.length === 0"></div>

    <div v-else class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-8 gap-2">
        <NuxtLink
            v-for="kill in items"
            :key="kill.killmail_id"
            :to="`/kill/${kill.killmail_id}`"
            class="group relative aspect-square rounded-lg overflow-hidden bg-black/40 border border-white/[0.04] hover:border-blue-500/30 transition-all"
        >
            <EveImage
                :src="`/images/types/${kill.ship_type_id}/overlayrender?size=256`"
                :alt="kill.ship_name"
                sizes="(min-width: 1360px) 144px, (min-width: 1024px) calc((100vw - 198px) / 8), (min-width: 768px) calc((100vw - 166px) / 4), (min-width: 640px) calc((100vw - 58px) / 3), calc((100vw - 50px) / 2)"
                class="absolute inset-0 w-full h-full object-contain p-2 transition-transform duration-300 group-hover:scale-110"
                loading="lazy"
            />
            <div class="absolute inset-x-0 bottom-0 h-2/3 bg-gradient-to-t from-black/90 via-black/50 to-transparent pointer-events-none"></div>
            <div class="absolute top-1.5 right-1.5 px-1.5 py-0.5 rounded bg-black/60 backdrop-blur-sm text-fine font-semibold text-green-400">
                {{ formatIsk(kill.total_value) }}
            </div>
            <div class="absolute inset-x-0 bottom-0 p-2.5 flex flex-col items-start">
                <div class="text-xs font-semibold text-white leading-tight truncate w-full drop-shadow-lg">{{ kill.ship_name }}</div>
                <div v-if="kill.victim_character_name" class="text-fine text-gray-300/90 truncate w-full mt-0.5 drop-shadow">{{ kill.victim_character_name }}</div>
                <div v-if="kill.victim_corporation_name" class="text-fine text-gray-400/80 truncate w-full drop-shadow">{{ kill.victim_corporation_name }}</div>
            </div>
        </NuxtLink>
    </div>
    </div>
</template>
