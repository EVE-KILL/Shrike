<script setup lang="ts">
interface CharEntry {
    characterId: number
    name: string
    kills: number
}

interface CorpData {
    name: string
    ticker: string
    characters: CharEntry[]
}

interface AllianceData {
    name: string
    ticker: string
    corporations: Record<number, CorpData>
}

interface ScanResult {
    alliances: Record<number, AllianceData>
    corporations: Record<number, CorpData>
    unresolved: string[]
    totalCharacters: number
    totalDangerous: number
}

const props = defineProps<{ result: ScanResult }>()

const sortedAlliances = computed(() => {
    return Object.entries(props.result.alliances)
        .map(([id, ally]) => {
            const corps = Object.entries(ally.corporations).map(([corpId, corp]) => ({
                id: Number(corpId),
                ...corp,
                characters: [...corp.characters].sort((a, b) => b.kills - a.kills),
            }))
            const charCount = corps.reduce((s, c) => s + c.characters.length, 0)
            return {
                id: Number(id),
                name: ally.name,
                ticker: ally.ticker,
                corporations: corps.sort((a, b) => b.characters.length - a.characters.length),
                charCount,
            }
        })
        .sort((a, b) => b.charCount - a.charCount)
})

const sortedCorps = computed(() => {
    return Object.entries(props.result.corporations)
        .map(([id, corp]) => ({
            id: Number(id),
            ...corp,
            characters: [...corp.characters].sort((a, b) => b.kills - a.kills),
        }))
        .sort((a, b) => b.characters.length - a.characters.length)
})

const threats = computed(() => {
    const all: (CharEntry & { corpName?: string, allianceName?: string })[] = []

    for (const ally of Object.values(props.result.alliances)) {
        for (const corp of Object.values(ally.corporations)) {
            for (const ch of corp.characters) {
                if (ch.kills > 5) all.push({ ...ch, corpName: corp.name, allianceName: ally.name })
            }
        }
    }
    for (const corp of Object.values(props.result.corporations)) {
        for (const ch of corp.characters) {
            if (ch.kills > 5) all.push({ ...ch, corpName: corp.name })
        }
    }

    return all.sort((a, b) => b.kills - a.kills)
})

const totalAllianceCount = computed(() => Object.keys(props.result.alliances).length)
const totalCorpCount = computed(() => {
    let count = Object.keys(props.result.corporations).length
    for (const ally of Object.values(props.result.alliances)) {
        count += Object.keys(ally.corporations).length
    }
    return count
})

const collapsed = ref<Record<string, boolean>>({})
function toggle(key: string) {
    collapsed.value[key] = collapsed.value[key] === undefined ? false : !collapsed.value[key]
}
function isOpen(key: string) {
    return collapsed.value[key] === false
}
</script>

<template>
    <div class="space-y-4">
        <!-- Threats -->
        <div v-if="threats.length > 0" class="rounded-lg bg-white/[0.04] border border-red-500/20 overflow-hidden">
            <div class="px-3 py-2 border-b border-white/[0.08] flex items-center gap-2">
                <Icon name="lucide:shield-alert" class="text-red-400 text-sm" />
                <span class="text-fine font-bold uppercase tracking-[0.15em] text-red-400/80">Potential Threats</span>
                <span class="text-fine text-red-400/60 ml-auto tabular-nums">{{ threats.length }}</span>
            </div>
            <div class="divide-y divide-white/[0.04] max-h-80 overflow-y-auto">
                <div v-for="t in threats" :key="t.characterId" class="px-3 py-1.5 flex items-center justify-between hover:bg-blue-500/[0.04] transition-colors">
                    <div class="flex items-center gap-2">
                        <img
                            :src="`/images/characters/${t.characterId}/portrait?size=64`"
                            :alt="t.name"
                            class="w-6 h-6 rounded"
                            loading="lazy"
                        />
                        <div>
                            <NuxtLink :to="`/character/${t.characterId}`" class="text-xs text-gray-300 hover:text-blue-400 hover:underline">{{ t.name }}</NuxtLink>
                            <div class="text-fine text-gray-600">
                                {{ t.corpName }}<span v-if="t.allianceName"> · {{ t.allianceName }}</span>
                            </div>
                        </div>
                    </div>
                    <span class="text-xs text-red-400/80 tabular-nums">{{ t.kills }}/wk</span>
                </div>
            </div>
        </div>

        <!-- Summary stats -->
        <div class="glass-panel p-3">
            <div class="px-1 pb-2 mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80 border-b border-white/[0.08]">
                Breakdown
            </div>
            <div class="grid grid-cols-3 gap-2">
                <div class="rounded bg-white/[0.04] p-2.5 text-center">
                    <div class="text-base font-semibold text-white tabular-nums">{{ totalAllianceCount }}</div>
                    <div class="text-fine text-gray-600 uppercase tracking-wide">Alliances</div>
                </div>
                <div class="rounded bg-white/[0.04] p-2.5 text-center">
                    <div class="text-base font-semibold text-white tabular-nums">{{ totalCorpCount }}</div>
                    <div class="text-fine text-gray-600 uppercase tracking-wide">Corporations</div>
                </div>
                <div class="rounded bg-white/[0.04] p-2.5 text-center">
                    <div class="text-base font-semibold text-white tabular-nums">{{ result.totalCharacters }}</div>
                    <div class="text-fine text-gray-600 uppercase tracking-wide">Characters</div>
                </div>
            </div>
        </div>

        <!-- Alliances -->
        <div v-for="ally in sortedAlliances" :key="ally.id" class="glass-panel overflow-hidden">
            <button
                class="w-full px-3 py-2.5 flex items-center justify-between hover:bg-blue-500/[0.04] transition-colors"
                @click="toggle(`ally-${ally.id}`)"
            >
                <div class="flex items-center gap-2">
                    <img
                        :src="`/images/alliances/${ally.id}/logo?size=64`"
                        :alt="ally.name"
                        class="w-6 h-6 rounded"
                        loading="lazy"
                    />
                    <Icon :name="isOpen(`ally-${ally.id}`) ? 'lucide:chevron-down' : 'lucide:chevron-right'" class="text-gray-600 text-xs" />
                    <NuxtLink :to="`/alliance/${ally.id}`" class="text-sm text-gray-300 hover:text-blue-400 hover:underline" @click.stop>{{ ally.name }}</NuxtLink>
                    <span class="text-xs text-gray-600">[{{ ally.ticker }}]</span>
                </div>
                <div class="flex items-center gap-3">
                    <span class="text-fine text-gray-600">{{ ally.corporations.length }} corps</span>
                    <span class="text-xs text-gray-500 tabular-nums">{{ ally.charCount }}</span>
                </div>
            </button>

            <div v-if="isOpen(`ally-${ally.id}`)" class="border-t border-white/[0.06]">
                <div v-for="corp in ally.corporations" :key="corp.id" class="border-b border-white/[0.04] last:border-b-0">
                    <button
                        class="w-full px-3 py-2 pl-8 flex items-center justify-between hover:bg-blue-500/[0.04] transition-colors"
                        @click="toggle(`corp-${ally.id}-${corp.id}`)"
                    >
                        <div class="flex items-center gap-2">
                            <img
                                :src="`/images/corporations/${corp.id}/logo?size=64`"
                                :alt="corp.name"
                                class="w-5 h-5 rounded"
                                loading="lazy"
                            />
                            <Icon :name="isOpen(`corp-${ally.id}-${corp.id}`) ? 'lucide:chevron-down' : 'lucide:chevron-right'" class="text-gray-600 text-xs" />
                            <NuxtLink :to="`/corporation/${corp.id}`" class="text-xs text-gray-400 hover:text-blue-400 hover:underline" @click.stop>{{ corp.name }}</NuxtLink>
                            <span class="text-fine text-gray-700">[{{ corp.ticker }}]</span>
                        </div>
                        <span class="text-xs text-gray-600 tabular-nums">{{ corp.characters.length }}</span>
                    </button>

                    <div v-if="isOpen(`corp-${ally.id}-${corp.id}`)">
                        <div
                            v-for="ch in corp.characters"
                            :key="ch.characterId"
                            class="px-3 py-1 pl-14 flex items-center justify-between text-xs hover:bg-blue-500/[0.04] transition-colors"
                        >
                            <NuxtLink :to="`/character/${ch.characterId}`" class="text-gray-500 hover:text-blue-400 hover:underline">{{ ch.name }}</NuxtLink>
                            <span v-if="ch.kills > 0" class="text-xs tabular-nums" :class="ch.kills > 5 ? 'text-red-400/80' : 'text-gray-700'">{{ ch.kills }}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Non-alliance corps -->
        <div v-if="sortedCorps.length > 0" class="rounded-lg bg-white/[0.04] border-0 overflow-hidden">
            <div class="px-3 py-2 border-b border-white/[0.08] flex items-center gap-2">
                <Icon name="lucide:users" class="text-yellow-500/60 text-sm" />
                <span class="text-fine font-bold uppercase tracking-[0.15em] text-yellow-500/60">Without Alliance</span>
                <span class="text-fine text-yellow-500/40 ml-auto tabular-nums">
                    {{ sortedCorps.reduce((s, c) => s + c.characters.length, 0) }}
                </span>
            </div>

            <div v-for="corp in sortedCorps" :key="corp.id" class="border-b border-white/[0.04] last:border-b-0">
                <button
                    class="w-full px-3 py-2 flex items-center justify-between hover:bg-blue-500/[0.04] transition-colors"
                    @click="toggle(`corp-na-${corp.id}`)"
                >
                    <div class="flex items-center gap-2">
                        <img
                            :src="`/images/corporations/${corp.id}/logo?size=64`"
                            :alt="corp.name"
                            class="w-5 h-5 rounded"
                            loading="lazy"
                        />
                        <Icon :name="isOpen(`corp-na-${corp.id}`) ? 'lucide:chevron-down' : 'lucide:chevron-right'" class="text-gray-600 text-xs" />
                        <NuxtLink :to="`/corporation/${corp.id}`" class="text-xs text-gray-400 hover:text-blue-400 hover:underline" @click.stop>{{ corp.name }}</NuxtLink>
                        <span class="text-fine text-gray-700">[{{ corp.ticker }}]</span>
                    </div>
                    <span class="text-xs text-gray-600 tabular-nums">{{ corp.characters.length }}</span>
                </button>

                <div v-if="isOpen(`corp-na-${corp.id}`)">
                    <div
                        v-for="ch in corp.characters"
                        :key="ch.characterId"
                        class="px-3 py-1 pl-10 flex items-center justify-between text-xs hover:bg-blue-500/[0.04] transition-colors"
                    >
                        <NuxtLink :to="`/character/${ch.characterId}`" class="text-gray-500 hover:text-blue-400 hover:underline">{{ ch.name }}</NuxtLink>
                        <span v-if="ch.kills > 0" class="text-xs tabular-nums" :class="ch.kills > 5 ? 'text-red-400/80' : 'text-gray-700'">{{ ch.kills }}</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Unresolved names -->
        <div v-if="result.unresolved.length > 0" class="glass-panel p-3">
            <p class="text-xs text-gray-600">
                {{ result.unresolved.length }} name{{ result.unresolved.length > 1 ? 's' : '' }} not found:
                <span class="text-gray-500">{{ result.unresolved.slice(0, 20).join(', ') }}<span v-if="result.unresolved.length > 20">...</span></span>
            </p>
        </div>
    </div>
</template>
