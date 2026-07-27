<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'

/**
 * Generic multi-pick typeahead over /api/search, advanced-search style: one
 * input, no type toggle — the query runs across every allowed type at once
 * and each result row is labeled with what it is. Emits `select` for every
 * picked hit and leaves list state (chips, caps, dedupe) to the parent.
 * Used by the campaign creator's battlefield/side/grant pickers — adopt it
 * anywhere an entity, system or region needs resolving by name.
 */

type PickableType = 'alliance' | 'corporation' | 'character' | 'system' | 'region' | 'constellation'

interface SearchHit {
    id: string
    name: string
    ticker: string | null
    type: PickableType
}

const props = defineProps<{
    /** Which result types to search across and accept. */
    types: PickableType[]
    placeholder?: string
    disabled?: boolean
    /** Rows for which this returns true render greyed-out and unpickable. */
    isPicked?: (type: PickableType, id: number) => boolean
}>()

const emit = defineEmits<{
    (e: 'select', picked: { type: PickableType; id: number; name: string; ticker: string | null }): void
}>()

const query = ref('')
const debouncedQuery = refDebounced(query, 300)
const results = ref<SearchHit[]>([])
const searching = ref(false)
const open = ref(false)
const containerRef = ref<HTMLElement | null>(null)
onClickOutside(containerRef, () => { open.value = false })

// /api/search ids are formatted as `${type}_${entity_id}` — strip the prefix.
const parseId = (raw: string): number => Number(String(raw).replace(/^[a-z]+_/i, ''))

watch(debouncedQuery, async (raw) => {
    const q = raw.trim()
    if (q.length < 2) { results.value = []; open.value = false; return }
    searching.value = true
    try {
        const res = await apiFetch<{ hits: SearchHit[] }>('/api/search', {
            params: { q, type: props.types.join(','), limit: 10 },
        })
        results.value = (res.hits || []).filter(h => props.types.includes(h.type))
        open.value = results.value.length > 0
    } catch {
        results.value = []
    } finally {
        searching.value = false
    }
})

const pick = (hit: SearchHit) => {
    emit('select', { type: hit.type, id: parseId(hit.id), name: hit.name, ticker: hit.ticker ?? null })
    query.value = ''
    results.value = []
    open.value = false
}

const rowImage = (hit: SearchHit): string | null => {
    const id = parseId(hit.id)
    if (hit.type === 'character') return `/images/characters/${id}/portrait?size=64`
    if (hit.type === 'corporation') return `/images/corporations/${id}/logo?size=64`
    if (hit.type === 'alliance') return `/images/alliances/${id}/logo?size=64`
    return null
}

// Same icons the /search type filters use, so a hit reads the same wherever
// it is surfaced.
const rowIcon = (hit: SearchHit): string => {
    if (hit.type === 'region') return 'lucide:map'
    if (hit.type === 'constellation') return 'lucide:compass'
    return 'lucide:map-pin'
}

const defaultPlaceholder = computed(() => {
    const names = props.types.map(t => t === 'corporation' ? 'corp' : t)
    const list = names.length > 1 ? `${names.slice(0, -1).join(', ')} or ${names.at(-1)}` : names[0]
    return `Search ${list}...`
})
</script>

<template>
    <div ref="containerRef" class="relative">
        <div class="glass-panel flex items-center gap-2 px-2.5 py-1.5 focus-within:border-blue-500/40 transition-colors">
            <Icon name="lucide:search" class="text-xs text-gray-500 flex-shrink-0" />
            <input v-model="query" type="text"
                :placeholder="placeholder ?? defaultPlaceholder"
                :disabled="disabled"
                class="flex-1 bg-transparent text-white text-xs outline-none placeholder-gray-600 disabled:opacity-40 min-w-0" />
            <Icon v-if="searching" name="lucide:loader-2" class="text-xs text-gray-500 animate-spin" />
        </div>
        <!-- Same row anatomy as the advanced-search dropdown: image (or icon
             tile for locations), then name over a type + ticker subline. -->
        <div v-if="open" class="absolute z-50 mt-1 w-full rounded-lg bg-black/90 backdrop-blur-xl border border-white/[0.08] shadow-2xl max-h-80 overflow-y-auto">
            <button v-for="hit in results" :key="hit.id" @click="pick(hit)"
                :disabled="isPicked?.(hit.type, parseId(hit.id))"
                class="w-full flex items-center gap-2.5 px-3 py-2.5 border-b border-white/[0.04] last:border-b-0 hover:bg-blue-500/[0.06] transition-colors text-left disabled:opacity-40 cursor-pointer">
                <EveImage v-if="rowImage(hit)" :src="rowImage(hit)!" :size="32" :alt="hit.name" class="w-7 h-7 rounded flex-shrink-0" />
                <div v-else class="w-7 h-7 rounded bg-white/[0.06] flex items-center justify-center flex-shrink-0">
                    <Icon :name="rowIcon(hit)" class="text-xs text-gray-500" />
                </div>
                <div class="min-w-0">
                    <div class="text-sm text-gray-200 truncate">{{ hit.name }}</div>
                    <div class="text-fine text-gray-500 capitalize">{{ hit.type }}
                        <span v-if="hit.ticker" class="text-gray-600 font-mono ml-1">[{{ hit.ticker }}]</span>
                    </div>
                </div>
            </button>
        </div>
    </div>
</template>
