<script setup lang="ts">
import { campaignSideAccents } from '#shared/utils/entityAccent'

/**
 * Custom-domain homepage widget: compact list of campaigns featuring the
 * domain's entities. The /api/campaigns endpoint scopes itself by host in
 * domain mode, so no props are needed — works in any widget slot.
 */

interface WidgetCampaignSide {
    side_index: number
    name: string
    kills: number
    isk_destroyed: number
    palette?: { main_color?: string; secondary_color?: string; tertiary_color?: string } | null
}

interface WidgetCampaign {
    campaign_id: string
    name: string
    status: number
    end_time: string | null
    totals: { killCount: number; iskDestroyed: number } | null
    sides: WidgetCampaignSide[]
}

const { data, pending } = await useApiFetch<{ campaigns: WidgetCampaign[] }>('/api/campaigns', {
    default: () => ({ campaigns: [] }),
})

const items = computed(() => (data.value?.campaigns ?? []).slice(0, 5))

const SIDE_COLORS = ['text-red-400', 'text-blue-400', 'text-emerald-400', 'text-orange-400']
const accents = (c: WidgetCampaign) => campaignSideAccents(c.sides.map(s => s.palette))

</script>

<template>
    <div class="glass-panel p-3">
        <div class="flex items-center justify-between mb-2 px-1">
            <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Campaigns</div>
            <NuxtLink to="/campaigns" class="text-fine text-gray-600 hover:text-blue-400 transition-colors">All →</NuxtLink>
        </div>

        <div v-if="pending && items.length === 0" class="flex items-center justify-center py-6">
            <Icon name="lucide:loader-2" class="text-base text-gray-500 animate-spin" />
        </div>

        <div v-else-if="!items.length" class="py-5 text-center">
            <p class="text-fine text-gray-600">No campaigns feature this killboard's entities yet</p>
            <NuxtLink to="/campaigncreator" class="text-fine text-blue-400 hover:text-blue-300">Start one →</NuxtLink>
        </div>

        <div v-else class="space-y-0.5">
            <NuxtLink v-for="c in items" :key="c.campaign_id" :to="`/campaign/${c.campaign_id}`"
                class="block px-2 py-1.5 rounded-md hover:bg-blue-500/[0.06] transition-colors">
                <div class="flex items-center gap-1.5">
                    <span class="w-1.5 h-1.5 rounded-full flex-shrink-0"
                        :class="c.status === 0 ? 'bg-amber-400 animate-pulse' : c.status === 2 ? 'bg-gray-600' : 'bg-green-400'"
                        v-tooltip="c.status === 0 ? 'Computing' : c.status === 2 ? 'Archived' : 'Active'" />
                    <span class="text-xs text-gray-200 truncate flex-1">{{ c.name }}</span>
                    <span v-if="c.totals" class="text-fine text-gray-600 tabular-nums flex-shrink-0">{{ formatIsk(c.totals.iskDestroyed) }}</span>
                </div>
                <div class="flex items-center gap-1 mt-0.5 pl-3 text-fine min-w-0">
                    <template v-for="(s, i) in c.sides.slice(0, 3)" :key="s.side_index">
                        <span v-if="i > 0" class="text-gray-700 flex-shrink-0">vs</span>
                        <span class="truncate" :class="SIDE_COLORS[s.side_index] ?? 'text-gray-400'"
                            :style="accents(c)[s.side_index] ? { color: accents(c)[s.side_index]!.accent } : undefined">
                            {{ s.name }}
                        </span>
                        <span class="text-gray-600 tabular-nums flex-shrink-0">{{ formatNumber(s.kills) }}</span>
                    </template>
                </div>
            </NuxtLink>
        </div>
    </div>
</template>
