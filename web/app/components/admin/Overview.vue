<script setup lang="ts">
// overview + esiData are fetched by the parent admin page (shared between the
// Overview and ESI sections) and passed down as props.
const props = defineProps<{
    overview?: any
    esiData?: any
}>()

/**
 * Queues with work waiting in them.
 *
 * The overview used to be three stat cards and a lone 7-day number, which meant
 * the only way to discover that eleven comments were sitting unreviewed was to
 * open the Moderation tab and look. These read from the same payload that feeds
 * the sidebar badges, so the landing page and the nav can't disagree.
 */
const attention = computed(() => {
    const a = props.overview?.attention
    if (!a) return []
    return [
        { key: 'moderation', count: a.moderation ?? 0, label: 'awaiting moderation', icon: 'lucide:shield-check', to: '/admin/moderation' },
        { key: 'assets', count: a.domainAssets ?? 0, label: 'domain assets to review', icon: 'lucide:image', to: '/admin/domains' },
        { key: 'paused', count: a.campaignsPaused ?? 0, label: 'campaigns paused', icon: 'lucide:pause', to: '/admin/campaigns' },
        { key: 'failed', count: a.campaignsFailed ?? 0, label: 'campaigns erroring', icon: 'lucide:triangle-alert', to: '/admin/campaigns' },
    ].filter(item => item.count > 0)
})

const esiUnhealthy = computed(() => (props.overview?.esi?.errorRate ?? 0) > 5)
</script>

<template>
    <!-- ═══════════════ OVERVIEW ═══════════════ -->
    <div class="space-y-4">
        <!-- Needs attention -->
        <div v-if="attention.length" class="glass-panel p-4 border-yellow-500/20">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-3">Needs Attention</h2>
            <div class="flex flex-wrap gap-2">
                <NuxtLink
                    v-for="item in attention" :key="item.key"
                    :to="item.to"
                    class="flex items-center gap-2 px-3 py-2 rounded-lg bg-yellow-500/[0.08] border border-yellow-500/20 text-sm text-yellow-400 hover:bg-yellow-500/[0.15] transition-colors"
                >
                    <Icon :name="item.icon" class="text-base flex-shrink-0" />
                    <span class="font-bold tabular-nums">{{ fmt(item.count) }}</span>
                    <span class="text-yellow-400/70">{{ item.label }}</span>
                </NuxtLink>
            </div>
        </div>

        <!-- Stat cards. Four of them, in a four-column grid — this used to hold
             three, leaving a permanently empty column on desktop. -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Users</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ fmt(overview?.users?.total) }}</div>
                <div class="text-xs text-gray-500 mt-1">+{{ fmt(overview?.users?.recent7d) }} this week</div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Killmails</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ fmt(overview?.killmails?.total) }}</div>
                <div class="text-xs text-gray-500 mt-1">{{ fmt(overview?.killmails?.last24h) }} today</div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Killmails (7d)</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ fmt(overview?.killmails?.last7d) }}</div>
                <div class="text-xs text-gray-500 mt-1">
                    ~{{ fmt(Math.round((overview?.killmails?.last7d ?? 0) / 7)) }}/day
                </div>
            </div>
            <div class="glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">ESI 24h</div>
                <div class="text-2xl font-bold text-white tabular-nums">{{ fmt(overview?.esi?.requests24h) }}</div>
                <div class="text-xs mt-1" :class="esiUnhealthy ? 'text-red-400' : 'text-gray-500'">
                    {{ overview?.esi?.errorRate }}% error rate
                </div>
            </div>
        </div>

        <!-- ESI volume chart -->
        <EsiVolumeChart :hours="esiData?.volumeByHour ?? []" title="ESI Request Volume (24h)" />
    </div>
</template>
