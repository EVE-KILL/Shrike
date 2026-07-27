<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import type { EntityKind } from '~/composables/apiDocs'

interface SearchHit {
    id: string
    name: string
    ticker?: string | null
    type: EntityKind
    entity_id?: number
}

const props = defineProps<{
    modelValue: string | number | null | undefined
    entityType: EntityKind
    baseUrl: string
    placeholder?: string
}>()

const emit = defineEmits<{
    (e: 'update:modelValue', v: number | null): void
}>()

const containerRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const isOpen = ref(false)
const query = ref('')
const debouncedQuery = refDebounced(query, 250)
const results = ref<SearchHit[]>([])
const loading = ref(false)
const selectedName = ref<string | null>(null)
const manualMode = ref(false)

onClickOutside(containerRef, () => { isOpen.value = false })

watch(debouncedQuery, async (q) => {
    if (!q || q.length < 2) { results.value = []; return }
    loading.value = true
    try {
        const data = await apiFetch<{ hits: any[] }>(`${props.baseUrl}/search`, {
            query: { q, type: props.entityType, limit: 10 },
        })
        results.value = (data.hits ?? []).map(h => ({
            id: h.id,
            name: h.name,
            ticker: h.ticker,
            type: h.type,
            // /search ids are formatted as `${type}_${entity_id}` — strip the prefix
            entity_id: Number(String(h.id).split('_').pop()),
        }))
    } catch {
        results.value = []
    } finally {
        loading.value = false
    }
})

const select = (hit: SearchHit) => {
    if (!hit.entity_id) return
    selectedName.value = hit.ticker ? `${hit.name} [${hit.ticker}]` : hit.name
    emit('update:modelValue', hit.entity_id)
    isOpen.value = false
    query.value = ''
}

const clear = () => {
    selectedName.value = null
    emit('update:modelValue', null)
    query.value = ''
    manualMode.value = false
}

const open = () => {
    isOpen.value = true
    nextTick(() => inputRef.value?.focus())
}

const switchToManual = () => {
    manualMode.value = true
    isOpen.value = false
}

const onManualInput = (e: Event) => {
    const v = (e.target as HTMLInputElement).value
    selectedName.value = null
    emit('update:modelValue', v ? Number(v) : null)
}

const imageUrl = (hit: SearchHit) => {
    switch (hit.type) {
        case 'character': return `/images/characters/${hit.entity_id}/portrait?size=32`
        case 'corporation': return `/images/corporations/${hit.entity_id}/logo?size=32`
        case 'alliance': return `/images/alliances/${hit.entity_id}/logo?size=32`
        case 'ship':
        case 'item': return `/images/types/${hit.entity_id}/icon?size=32`
        case 'faction': return `/images/corporations/${hit.entity_id}/logo?size=32`
        default: return ''
    }
}

const placeholderLabel = computed(() => props.placeholder ?? `Search ${props.entityType}…`)
</script>

<template>
    <div ref="containerRef" class="relative inline-flex w-full">
        <!-- Manual numeric input -->
        <div v-if="manualMode" class="flex items-center gap-1.5 w-full">
            <input
                type="number"
                :value="modelValue ?? ''"
                placeholder="Numeric ID"
                class="flex-1 bg-white/[0.04] border border-white/[0.10] rounded-md px-2.5 py-1.5 text-xs text-white placeholder-gray-600 outline-none focus:border-blue-500/40"
                @input="onManualInput"
            >
            <button
                class="text-[10px] text-gray-500 hover:text-blue-400 transition-colors px-2 py-1.5 rounded-md hover:bg-white/[0.04]"
                @click="manualMode = false"
            >
                search
            </button>
        </div>

        <!-- Selected state -->
        <div
            v-else-if="modelValue != null && selectedName"
            class="flex items-center gap-1.5 w-full px-2.5 py-1.5 rounded-md text-xs bg-blue-500/[0.08] text-blue-300 border border-blue-500/[0.20] hover:border-blue-500/[0.40] transition-colors cursor-pointer"
            role="button"
            tabindex="0"
            @click="open"
            @keydown.enter="open"
        >
            <Icon name="lucide:check-circle-2" class="text-[12px] flex-shrink-0" />
            <span class="truncate flex-1 text-left">{{ selectedName }}</span>
            <span class="text-[10px] text-blue-400/60">{{ modelValue }}</span>
            <span
                class="text-blue-400/60 hover:text-blue-300 flex-shrink-0 cursor-pointer"
                role="button"
                tabindex="0"
                aria-label="Clear selection"
                @click.stop="clear"
                @keydown.enter.stop="clear"
            >
                <Icon name="lucide:x" class="text-[12px]" />
            </span>
        </div>

        <!-- Empty trigger -->
        <button
            v-else-if="!isOpen"
            class="flex items-center gap-1.5 w-full px-2.5 py-1.5 rounded-md text-xs bg-white/[0.04] text-gray-500 border border-white/[0.08] hover:border-white/[0.16] hover:text-blue-400 transition-colors text-left"
            @click="open"
        >
            <Icon name="lucide:search" class="text-[12px] opacity-60 flex-shrink-0" />
            <span class="truncate flex-1">{{ placeholderLabel }}</span>
            <span
                class="text-[10px] text-gray-600 hover:text-blue-400 flex-shrink-0 px-1"
                @click.stop="switchToManual"
            >
                #
            </span>
        </button>

        <!-- Active search input -->
        <div v-else class="flex items-center gap-1.5 w-full bg-white/[0.06] border border-blue-500/[0.30] rounded-md px-2.5 py-1.5">
            <Icon name="lucide:search" class="text-blue-400 text-[12px] flex-shrink-0" />
            <input
                ref="inputRef"
                v-model="query"
                type="text"
                :placeholder="placeholderLabel"
                class="bg-transparent text-xs text-white placeholder-gray-600 outline-none w-full"
                @keydown.escape="isOpen = false"
            >
            <Icon v-if="loading" name="lucide:loader-2" class="text-[12px] text-blue-400 animate-spin flex-shrink-0" />
        </div>

        <!-- Results dropdown -->
        <Transition
            enter-active-class="transition duration-150 ease-out"
            enter-from-class="opacity-0 -translate-y-1"
            enter-to-class="opacity-100 translate-y-0"
            leave-active-class="transition duration-100 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
        >
            <div
                v-if="isOpen && (results.length > 0 || (debouncedQuery.length >= 2 && !loading))"
                class="absolute top-full left-0 right-0 mt-1.5 z-50 rounded-lg border border-white/[0.10] bg-[#0c0c0c]/95 backdrop-blur-2xl shadow-2xl shadow-black/60 overflow-y-auto max-h-64 p-1"
            >
                <button
                    v-for="hit in results"
                    :key="hit.id"
                    class="flex items-center gap-2 w-full px-2 py-1.5 rounded text-xs text-gray-400 hover:text-blue-300 hover:bg-blue-500/[0.08] transition-colors text-left cursor-pointer"
                    @click="select(hit)"
                >
                    <img v-if="imageUrl(hit)" :src="imageUrl(hit)" class="w-5 h-5 rounded-sm flex-shrink-0" loading="lazy">
                    <div class="flex-1 min-w-0">
                        <div class="truncate text-white/90">{{ hit.name }}</div>
                        <div v-if="hit.ticker" class="text-[10px] text-gray-600 truncate">[{{ hit.ticker }}]</div>
                    </div>
                    <span class="text-[10px] text-gray-600 font-mono flex-shrink-0">{{ hit.entity_id }}</span>
                </button>
                <div v-if="results.length === 0 && !loading" class="px-3 py-2.5 text-xs text-gray-600 text-center">
                    No results
                </div>
            </div>
        </Transition>
    </div>
</template>
