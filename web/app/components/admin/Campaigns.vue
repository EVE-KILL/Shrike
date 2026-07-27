<script setup lang="ts">
const state = ref('all')
const search = ref('')
const page = ref(1)
const acting = ref<string | null>(null)
const confirmDelete = ref<string | null>(null)
const actionError = ref('')

const query = computed(() => ({
    state: state.value,
    q: search.value || undefined,
    page: page.value,
}))
const { data, pending, refresh } = useApiFetch<{
    campaigns: any[]
    page: number
    hasMore: boolean
}>('/api/admin/campaigns', { query })

watch([state, search], () => { page.value = 1 })

const act = async (campaignId: string, action: 'pause' | 'resume' | 'reprocess' | 'archive' | 'delete') => {
    if (action === 'delete' && confirmDelete.value !== campaignId) {
        confirmDelete.value = campaignId
        setTimeout(() => {
            if (confirmDelete.value === campaignId) confirmDelete.value = null
        }, 5000)
        return
    }
    acting.value = campaignId
    actionError.value = ''
    try {
        await apiFetch(`/api/admin/campaigns/${campaignId}/action`, {
            method: 'POST',
            body: { action },
        })
        confirmDelete.value = null
        await refresh()
    } catch (error: any) {
        actionError.value = error?.data?.message || error?.message || 'Campaign action failed'
    } finally {
        acting.value = null
    }
}

const statusLabel = (campaign: any) => {
    if (campaign.processing_paused) return 'Paused'
    if (Number(campaign.status) === 0) return 'Pending'
    if (Number(campaign.status) === 2) return 'Archived'
    return 'Active'
}

const statusClass = (campaign: any) => {
    if (campaign.processing_paused) return 'text-red-400 bg-red-500/10'
    if (Number(campaign.status) === 0) return 'text-yellow-400 bg-yellow-500/10'
    if (Number(campaign.status) === 2) return 'text-gray-500 bg-white/[0.04]'
    return 'text-green-400 bg-green-500/10'
}

const formatDuration = (milliseconds: number | null) => {
    if (!milliseconds) return '—'
    if (milliseconds < 1000) return `${milliseconds}ms`
    if (milliseconds < 60_000) return `${(milliseconds / 1000).toFixed(1)}s`
    return `${(milliseconds / 60_000).toFixed(1)}m`
}
</script>

<template>
    <div class="space-y-4">
        <div class="glass-panel p-4">
            <div class="flex flex-col md:flex-row gap-3">
                <div class="relative flex-1">
                    <Icon name="lucide:search" class="absolute left-3 top-2.5 text-sm text-gray-600" />
                    <input
                        v-model="search"
                        class="w-full pl-9 pr-3 py-2 rounded-md bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-blue-500/50"
                        placeholder="Campaign ID, name, or creator…"
                    >
                </div>
                <SelectMenu
                    v-model="state"
                    :options="[
                        { value: 'all', label: 'All campaigns' },
                        { value: 'paused', label: 'Paused' },
                        { value: 'failed', label: 'Failed' },
                        { value: 'pending', label: 'Pending' },
                        { value: 'active', label: 'Active' },
                        { value: 'archived', label: 'Archived' },
                    ]"
                />
            </div>
            <p v-if="actionError" class="mt-3 text-xs text-red-400">{{ actionError }}</p>
        </div>

        <div v-if="pending && !data" class="glass-panel p-8 text-center text-sm text-gray-500">
            Loading campaigns…
        </div>

        <div v-for="campaign in (data?.campaigns || [])" :key="campaign.campaign_id" class="glass-panel p-4">
            <div class="flex flex-col xl:flex-row xl:items-start gap-4">
                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 flex-wrap">
                        <NuxtLink :to="`/campaign/${campaign.campaign_id}`" class="text-sm font-medium text-white hover:text-blue-400 truncate">
                            {{ campaign.name }}
                        </NuxtLink>
                        <span class="px-2 py-0.5 rounded text-fine font-medium" :class="statusClass(campaign)">
                            {{ statusLabel(campaign) }}
                        </span>
                        <span class="text-fine text-gray-600">
                            {{ Number(campaign.visibility) === 0 ? 'Public' : Number(campaign.visibility) === 1 ? 'Private' : 'Killboard' }}
                        </span>
                    </div>
                    <div class="mt-1 text-fine text-gray-600">
                        <span class="font-mono">{{ campaign.campaign_id }}</span>
                        <span> · </span>
                        <NuxtLink :to="`/character/${campaign.created_by_character_id}`" class="hover:text-blue-400">
                            {{ campaign.creator_name || campaign.created_by_character_id }}
                        </NuxtLink>
                    </div>
                    <p v-if="campaign.processing_note" class="mt-2 text-xs text-red-300">{{ campaign.processing_note }}</p>
                    <p v-if="campaign.last_processing_error" class="mt-1 text-fine text-red-400/70 break-words">{{ campaign.last_processing_error }}</p>
                </div>

                <div class="grid grid-cols-3 gap-4 text-center xl:w-80">
                    <div>
                        <div class="text-xs text-gray-300 tabular-nums">{{ formatNumber(campaign.estimated_killmails ?? 0) }}</div>
                        <div class="text-fine text-gray-600">Estimated</div>
                    </div>
                    <div>
                        <div class="text-xs text-gray-300 tabular-nums">{{ formatNumber(campaign.last_processing_killmails ?? campaign.totals?.killCount ?? 0) }}</div>
                        <div class="text-fine text-gray-600">Processed</div>
                    </div>
                    <div>
                        <div class="text-xs text-gray-300 tabular-nums">{{ formatDuration(campaign.last_processing_duration_ms) }}</div>
                        <div class="text-fine text-gray-600">Runtime</div>
                    </div>
                </div>

                <div class="flex items-center gap-1.5 flex-wrap xl:justify-end">
                    <button
                        v-if="campaign.processing_paused"
                        class="px-2.5 py-1.5 rounded-md text-fine text-green-400 bg-green-500/10 hover:bg-green-500/20"
                        :disabled="acting === campaign.campaign_id"
                        @click="act(campaign.campaign_id, 'resume')"
                    >Resume</button>
                    <button
                        v-else
                        class="px-2.5 py-1.5 rounded-md text-fine text-yellow-400 bg-yellow-500/10 hover:bg-yellow-500/20"
                        :disabled="acting === campaign.campaign_id"
                        @click="act(campaign.campaign_id, 'pause')"
                    >Pause</button>
                    <button
                        class="px-2.5 py-1.5 rounded-md text-fine text-blue-400 bg-blue-500/10 hover:bg-blue-500/20"
                        :disabled="acting === campaign.campaign_id"
                        @click="act(campaign.campaign_id, 'reprocess')"
                    >Reprocess</button>
                    <button
                        class="px-2.5 py-1.5 rounded-md text-fine text-gray-400 bg-white/[0.04] hover:bg-white/[0.08]"
                        :disabled="acting === campaign.campaign_id"
                        @click="act(campaign.campaign_id, 'archive')"
                    >Archive</button>
                    <button
                        class="px-2.5 py-1.5 rounded-md text-fine bg-red-500/10 hover:bg-red-500/20"
                        :class="confirmDelete === campaign.campaign_id ? 'text-red-300 ring-1 ring-red-500/40' : 'text-red-400'"
                        :disabled="acting === campaign.campaign_id"
                        @click="act(campaign.campaign_id, 'delete')"
                    >{{ confirmDelete === campaign.campaign_id ? 'Confirm delete' : 'Delete' }}</button>
                </div>
            </div>
        </div>

        <div v-if="data && !data.campaigns.length" class="glass-panel p-8 text-center text-sm text-gray-500">
            No campaigns match these filters.
        </div>

        <div v-if="data && (page > 1 || data.hasMore)" class="flex items-center justify-center gap-2">
            <button class="glass-panel px-3 py-1.5 text-xs text-gray-400 disabled:opacity-30" :disabled="page <= 1" @click="page--">Previous</button>
            <span class="text-fine text-gray-600">Page {{ page }}</span>
            <button class="glass-panel px-3 py-1.5 text-xs text-gray-400 disabled:opacity-30" :disabled="!data.hasMore" @click="page++">Next</button>
        </div>
    </div>
</template>
