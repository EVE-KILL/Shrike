<script setup lang="ts">
import type { WidgetConfig } from '~/composables/useDomainConfig'
import type { EntityTracker } from '~/composables/useTrackerNotifications'

const props = defineProps<{
    widget: WidgetConfig
    trackers: EntityTracker[]
    statsEndpoint: string
    killlistEndpoint: string
    streamTopics: string[] | null
}>()

const topBoxMap: Record<string, { title: string; dataType: string }> = {
    topCharacters: { title: 'Active Pilots', dataType: 'characters' },
    topCorporations: { title: 'Active Corporations', dataType: 'corporations' },
    topAlliances: { title: 'Active Alliances', dataType: 'alliances' },
    topShips: { title: 'Most Used Ships', dataType: 'ships' },
    topSystems: { title: 'Top Systems', dataType: 'systems' },
    topRegions: { title: 'Top Regions', dataType: 'regions' },
}

const topBox = computed(() => topBoxMap[props.widget.type])
const killlistType = computed(() => props.widget.killlistType || 'latest')

function trackerIcon(type: string): string {
    if (type === 'character') return 'lucide:user'
    if (type === 'corporation') return 'lucide:building-2'
    if (type === 'alliance') return 'lucide:flag'
    if (type === 'region') return 'lucide:map'
    if (type === 'constellation') return 'lucide:orbit'
    return 'lucide:map-pin'
}
</script>

<template>
    <MostValuable
        v-if="widget.type === 'mostValuable'"
        :api-endpoint="statsEndpoint"
    />

    <KillList
        v-else-if="widget.type === 'killList'"
        :key="`${killlistType}-${streamTopics?.join(',') || 'offline'}`"
        :killlist-type="killlistType"
        :entity-endpoint="killlistEndpoint"
        :extra-params="{ type: killlistType }"
        :stream-topics="streamTopics"
    />

    <TopBox
        v-else-if="topBox"
        :title="topBox.title"
        :data-type="topBox.dataType"
        :limit="10"
        :days="7"
        :api-endpoint="statsEndpoint"
    />

    <div v-else-if="widget.type === 'entityInfo'" class="glass-panel mb-4 p-2">
        <div class="mb-1 border-b border-white/[0.08] px-1 pb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-300">
            Tracked Scope
        </div>
        <div class="space-y-px">
            <NuxtLink
                v-for="tracker in trackers"
                :key="tracker.id"
                :to="`/${tracker.target_type}/${tracker.target_id}`"
                class="flex items-center gap-2 rounded-md px-2 py-1.5 text-xs transition-colors hover:bg-blue-500/[0.06] hover:text-blue-300"
                :class="tracker.enabled ? 'text-gray-300' : 'text-gray-600'"
            >
                <Icon :name="trackerIcon(tracker.target_type)" class="flex-shrink-0 text-gray-500" />
                <span class="min-w-0 flex-1 truncate">{{ tracker.target_name }}</span>
                <span v-if="!tracker.enabled" class="text-fine uppercase text-gray-700">Paused</span>
            </NuxtLink>
        </div>
        <NuxtLink to="/trackers" class="mt-2 flex items-center gap-1 px-2 py-1 text-fine text-blue-400 hover:text-blue-300">
            <Icon name="lucide:settings-2" /> Manage trackers
        </NuxtLink>
    </div>

    <div v-else-if="widget.type === 'textBlock'" class="glass-panel mb-4 p-4">
        <p class="whitespace-pre-wrap text-sm leading-relaxed text-gray-400">{{ widget.content }}</p>
    </div>
</template>
