<script setup lang="ts">
/**
 * Detail modal for one ESI request-log row.
 *
 * The admin copy grew three sections the settings copy never got — the entity
 * the request was about, who triggered it, and links to the killmails it
 * discovered — and the two also used different label styling for the same
 * fields. Both now render from here; `showEntity` gates the admin-only parts,
 * which are meaningless on your own log where every row is you.
 */
import type { EsiLogRow } from '~/utils/esiLog'

const props = defineProps<{
    row: EsiLogRow | null
    /** Show entity / requester / discovered-killmail sections. Admin only. */
    showEntity?: boolean
}>()

const open = defineModel<boolean>({ required: true })

const newItemIds = computed(() => Array.isArray(props.row?.new_item_ids) ? props.row!.new_item_ids! : [])
</script>

<template>
    <Modal v-model="open">
        <template #header>
            <div class="flex items-center gap-3">
                <span class="px-2 py-0.5 rounded text-xs font-mono tabular-nums" :class="esiStatusBadgeClass(row)">
                    {{ row?.status_code ?? '—' }}
                </span>
                <span class="text-sm font-medium text-white">{{ row?.method }} Request</span>
            </div>
        </template>

        <template v-if="row" #default>
            <div class="space-y-4">
                <div>
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-1">Endpoint</div>
                    <div class="text-xs font-mono text-gray-300 break-all bg-white/[0.04] rounded-lg px-3 py-2">{{ row.endpoint }}</div>
                </div>

                <div v-if="showEntity && row.endpoint_entity_name" class="flex items-center gap-3">
                    <AdminEsiEntityAvatar :entity-type="row.endpoint_type" :entity-id="row.endpoint_entity_id" :size="64" class="w-10 h-10 rounded-full" />
                    <div>
                        <div class="text-sm text-white font-medium">{{ row.endpoint_entity_name }}</div>
                        <div class="text-xs text-gray-500">{{ row.endpoint_type }} #{{ row.endpoint_entity_id }}</div>
                    </div>
                </div>

                <div v-if="row.error_message">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-red-400/80 mb-1">Error</div>
                    <div class="text-xs text-red-400/70 bg-red-500/[0.06] rounded-lg px-3 py-2 break-all">{{ row.error_message }}</div>
                </div>

                <div class="grid grid-cols-2 gap-x-6 gap-y-3">
                    <div>
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">Time</div>
                        <div class="text-xs text-gray-300 tabular-nums">{{ fmtDate(row.created_at) }}</div>
                    </div>
                    <div>
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">Duration</div>
                        <div class="text-xs tabular-nums" :class="(row.request_duration_ms ?? 0) > 2000 ? 'text-yellow-400' : 'text-gray-300'">
                            {{ row.request_duration_ms ?? '—' }} ms
                        </div>
                    </div>
                    <div>
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">Items Returned</div>
                        <div class="text-xs text-gray-300 tabular-nums">{{ row.items_returned ?? 0 }}</div>
                    </div>
                    <div>
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">New Items</div>
                        <div class="text-xs tabular-nums" :class="(row.new_items ?? 0) > 0 ? 'text-blue-400' : 'text-gray-300'">{{ row.new_items ?? 0 }}</div>
                    </div>
                    <div>
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">Source</div>
                        <div class="text-xs text-gray-300">{{ row.source }}</div>
                    </div>
                    <div v-if="showEntity">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">Requested By</div>
                        <div v-if="row.character_name" class="flex items-center gap-1.5 text-xs text-gray-300">
                            <img :src="`/images/characters/${row.character_id}/portrait?size=32`" class="w-4 h-4 rounded-full" loading="lazy">
                            {{ row.character_name }}
                        </div>
                        <div v-else class="text-xs text-gray-600">—</div>
                    </div>
                    <div>
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-600 mb-0.5">Method</div>
                        <div class="text-xs text-gray-300">{{ row.method }}</div>
                    </div>
                </div>

                <div v-if="showEntity && (row.new_items ?? 0) > 0">
                    <div class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 mb-1.5 flex items-center gap-1.5">
                        <Icon name="lucide:sparkles" class="text-xs" />
                        New Killmails
                        <span class="text-gray-600 font-normal normal-case tracking-normal">({{ row.new_items }})</span>
                    </div>
                    <div v-if="newItemIds.length" class="flex flex-wrap gap-1.5 bg-blue-500/[0.04] rounded-lg px-3 py-2">
                        <NuxtLink
                            v-for="kmId in newItemIds" :key="kmId"
                            :to="`/kill/${kmId}`"
                            target="_blank"
                            class="text-xs font-mono tabular-nums px-1.5 py-0.5 rounded bg-blue-500/15 text-blue-400 hover:bg-blue-500/25 transition-colors"
                        >{{ kmId }}</NuxtLink>
                    </div>
                    <div v-else class="text-xs text-gray-500 italic bg-white/[0.02] rounded-lg px-3 py-2">
                        Item IDs not recorded for this request (logged before tracking was added).
                    </div>
                </div>
            </div>
        </template>
    </Modal>
</template>
