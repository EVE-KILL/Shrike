<script setup lang="ts">
const props = defineProps<{
    allianceId?: number
    corporationId?: number
    characterId?: number
}>()

interface CampaignListEntity {
    entity_type: number
    entity_id: number
}

interface CampaignListSide {
    side_index: number
    name: string
    kills: number
    losses: number
    isk_destroyed: number
    isk_lost: number
    palette?: { main_color?: string; secondary_color?: string; tertiary_color?: string } | null
    entities: CampaignListEntity[]
}

interface CampaignListItem {
    campaign_id: string
    name: string
    description: string | null
    status: number
    start_time: string
    end_time: string | null
    last_activity_at: string | null
    creator: { character_id: number; name: string | null }
    totals: { killCount: number; iskDestroyed: number } | null
    sides: CampaignListSide[]
}

const page = ref(1)

const fetchParams = computed(() => {
    const p: Record<string, any> = { page: page.value }
    if (props.allianceId) { p.entityType = 'alliance'; p.entityId = props.allianceId }
    else if (props.corporationId) { p.entityType = 'corporation'; p.entityId = props.corporationId }
    else if (props.characterId) { p.entityType = 'character'; p.entityId = props.characterId }
    return p
})

const { data, pending } = await useApiFetch<{ campaigns: CampaignListItem[]; hasMore: boolean; page: number }>('/api/campaigns', {
    params: fetchParams,
    watch: [page],
    default: () => ({ campaigns: [], hasMore: false, page: 1 }),
})

const items = computed(() => data.value?.campaigns ?? [])

</script>

<template>
    <div>
        <div v-if="pending && items.length === 0" class="flex items-center justify-center py-20">
            <Icon name="lucide:loader-2" class="text-2xl text-gray-500 animate-spin" />
        </div>

        <div v-else-if="!items.length" class="glass-panel p-12 text-center">
            <Icon name="lucide:flag-off" class="text-3xl text-gray-600 mb-3" />
            <p class="text-sm text-gray-500 mb-4">No campaigns feature this entity yet</p>
            <NuxtLink to="/campaigncreator" class="text-sm text-blue-400 hover:text-blue-300">Start one →</NuxtLink>
        </div>

        <template v-else>
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <CampaignCard v-for="c in items" :key="c.campaign_id" :campaign="c" />
            </div>

            <!-- Pagination -->
            <div v-if="page > 1 || data?.hasMore" class="flex items-center justify-center gap-2 mt-4">
                <button :disabled="page <= 1" @click="page--"
                    class="px-3 py-1.5 rounded-lg text-xs font-medium bg-white/[0.04] text-gray-400 border border-white/[0.08] disabled:opacity-40 hover:bg-blue-500/[0.08] transition-colors">
                    Previous
                </button>
                <span class="text-xs text-gray-500 tabular-nums">Page {{ page }}</span>
                <button :disabled="!data?.hasMore" @click="page++"
                    class="px-3 py-1.5 rounded-lg text-xs font-medium bg-white/[0.04] text-gray-400 border border-white/[0.08] disabled:opacity-40 hover:bg-blue-500/[0.08] transition-colors">
                    Next
                </button>
            </div>
        </template>
    </div>
</template>
