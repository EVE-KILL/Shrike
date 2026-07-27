<script setup lang="ts">
/**
 * ESI request log — table on desktop, stacked cards on mobile.
 *
 * One list, two audiences: the admin monitor shows every user's requests and so
 * needs an Entity column, while the settings tab shows only your own and drops
 * it. That is the sole difference between what used to be two ~90-line copies.
 */
import type { EsiLogRow } from '~/utils/esiLog'

defineProps<{
    rows: EsiLogRow[]
    /** Show which character/corporation each request was about. Admin only. */
    showEntity?: boolean
}>()

const emit = defineEmits<{ select: [row: EsiLogRow] }>()
</script>

<template>
    <div>
        <!-- Desktop: table -->
        <div class="hidden md:block overflow-x-auto">
            <table v-if="rows.length" class="w-full text-sm">
                <thead>
                    <tr class="border-b border-white/[0.06]">
                        <th class="text-left px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Time</th>
                        <th class="text-left px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Type</th>
                        <th class="text-left px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Action</th>
                        <th v-if="showEntity" class="text-left px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Entity</th>
                        <th class="text-right px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">New</th>
                        <th class="text-center px-4 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600 w-[50px]">Status</th>
                    </tr>
                </thead>
                <tbody>
                    <tr
                        v-for="row in rows" :key="row.id"
                        class="border-b border-white/[0.04] hover:bg-blue-500/[0.04] cursor-pointer transition-colors"
                        :class="esiRowTintClass(row)"
                        @click="emit('select', row)"
                    >
                        <td class="px-4 py-2 text-gray-500 text-xs tabular-nums whitespace-nowrap">{{ fmtDate(row.created_at) }}</td>
                        <td class="px-4 py-2">
                            <span class="text-xs px-1.5 py-0.5 rounded" :class="esiTypeChipClass(row.endpoint_type)">
                                {{ row.endpoint_type }}
                            </span>
                        </td>
                        <td class="px-4 py-2 text-gray-300 text-xs">{{ row.endpoint_action || '—' }}</td>
                        <td v-if="showEntity" class="px-4 py-2">
                            <div v-if="row.endpoint_entity_name && row.endpoint_type && row.endpoint_entity_id != null" class="flex items-center gap-1.5 text-xs text-gray-300">
                                <AdminEsiEntityAvatar :entity-type="row.endpoint_type" :entity-id="row.endpoint_entity_id" :size="32" class="w-4 h-4 rounded-full" />
                                <span class="truncate max-w-[180px]">{{ row.endpoint_entity_name }}</span>
                            </div>
                            <span v-else class="text-gray-700 text-xs">{{ row.endpoint_entity_id || '—' }}</span>
                        </td>
                        <td class="px-4 py-2 text-right tabular-nums text-xs" :class="(row.new_items ?? 0) > 0 ? 'text-blue-400' : 'text-gray-700'">
                            {{ row.new_items ?? 0 }}
                        </td>
                        <td class="px-4 py-2 text-center">
                            <span class="inline-block w-2 h-2 rounded-full" :class="esiStatusDotClass(row)"></span>
                        </td>
                    </tr>
                </tbody>
            </table>
            <div v-else class="px-5 py-8 text-center text-gray-600 text-sm">No ESI requests found.</div>
        </div>

        <!-- Mobile: card rows -->
        <div class="md:hidden">
            <div v-if="rows.length">
                <div
                    v-for="row in rows" :key="`m-${row.id}`"
                    class="px-3 py-2.5 border-b border-white/[0.04] active:bg-white/[0.03] transition-colors"
                    :class="esiRowTintClass(row)"
                    @click="emit('select', row)"
                >
                    <div class="flex items-center justify-between gap-2 mb-1">
                        <div class="flex items-center gap-2 min-w-0">
                            <span class="inline-block w-2 h-2 rounded-full flex-shrink-0" :class="esiStatusDotClass(row)"></span>
                            <span class="text-xs text-gray-300 truncate">{{ row.endpoint_action || '—' }}</span>
                            <span class="text-fine px-1.5 py-0.5 rounded flex-shrink-0" :class="esiTypeChipClass(row.endpoint_type)">
                                {{ row.endpoint_type }}
                            </span>
                        </div>
                        <span v-if="(row.new_items ?? 0) > 0" class="text-fine text-blue-400 tabular-nums flex-shrink-0">+{{ row.new_items }}</span>
                    </div>
                    <div class="flex items-center gap-2" :class="showEntity ? 'justify-between' : 'justify-end'">
                        <template v-if="showEntity">
                            <div v-if="row.endpoint_entity_name && row.endpoint_type && row.endpoint_entity_id != null" class="flex items-center gap-1.5 min-w-0">
                                <AdminEsiEntityAvatar :entity-type="row.endpoint_type" :entity-id="row.endpoint_entity_id" :size="32" class="w-4 h-4 rounded-full flex-shrink-0" />
                                <span class="text-fine text-gray-400 truncate">{{ row.endpoint_entity_name }}</span>
                            </div>
                            <span v-else class="text-fine text-gray-700">{{ row.endpoint_entity_id || '—' }}</span>
                        </template>
                        <span class="text-fine text-gray-600 tabular-nums flex-shrink-0">{{ fmtDate(row.created_at) }}</span>
                    </div>
                </div>
            </div>
            <div v-else class="px-5 py-8 text-center text-gray-600 text-sm">No ESI requests found.</div>
        </div>
    </div>
</template>
