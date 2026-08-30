<script setup lang="ts">
interface SearchHit {
    id: string
    name: string
    ticker: string | null
    type: 'character' | 'corporation' | 'alliance' | 'faction' | 'system' | 'region' | 'constellation' | 'ship' | 'item'
}

interface SearchResponse {
    hits: SearchHit[]
    query: string
}

interface BattleEntity {
    id: number
    name: string
    type: 'alliance' | 'corporation' | 'faction'
    alliance_id?: number | null
    alliance_name?: string | null
}

interface SelectedSystem {
    id: number
    name: string
}

const { isAuthenticated, login } = useAuth()

useHead({ title: 'Battle Generator' })
useSeoMeta({
    description: 'Create custom battle reports by selecting systems, time ranges, and assigning alliances and corporations to sides.',
    ogTitle: 'Battle Generator — EVE-KILL',
    ogDescription: 'Generate custom EVE Online battle reports on EVE-KILL.',
})

useSchemaOrg([
    defineBreadcrumb({
        itemListElement: [
            { name: 'Home', item: '/' },
            { name: 'Battle Generator' },
        ],
    }),
])

// --- Step state ---
const currentStep = ref(1)

// Step 1: System + Time
const systems = ref<SelectedSystem[]>([])
const startTime = ref('')
const endTime = ref('')
const systemSearch = ref('')
const systemResults = ref<SearchHit[]>([])
const showSystemDropdown = ref(false)
const isSearchingSystems = ref(false)
let systemSearchTimeout: ReturnType<typeof setTimeout>

const systemLimitReached = computed(() => systems.value.length >= 5)

const handleSystemSearch = () => {
    const q = systemSearch.value.trim()
    if (!q || q.length < 2) {
        systemResults.value = []
        showSystemDropdown.value = false
        return
    }
    clearTimeout(systemSearchTimeout)
    systemSearchTimeout = setTimeout(async () => {
        isSearchingSystems.value = true
        try {
            const response = await apiFetch<SearchResponse>('/api/search', { params: { q, type: 'system', limit: 10 } })
            systemResults.value = (response.hits || []).filter(h => h.type === 'system')
            showSystemDropdown.value = systemResults.value.length > 0
        } catch {
            systemResults.value = []
        } finally {
            isSearchingSystems.value = false
        }
    }, 300)
}

const parseId = (raw: string): number => Number(String(raw).replace(/^[a-z]+_/i, ''))

const addSystem = (hit: SearchHit) => {
    if (systemLimitReached.value) return
    const numericId = parseId(hit.id)
    if (systems.value.some(s => s.id === numericId)) return
    systems.value.push({ id: numericId, name: hit.name })
    systemSearch.value = ''
    systemResults.value = []
    showSystemDropdown.value = false
}

const removeSystem = (id: number) => {
    systems.value = systems.value.filter(s => s.id !== id)
}

const step1Ready = computed(() =>
    systems.value.length > 0 && !!startTime.value && !!endTime.value
    && eveInputToDate(endTime.value) > eveInputToDate(startTime.value)
)

const timeSpanHours = computed(() => {
    if (!startTime.value || !endTime.value) return 0
    return (eveInputToDate(endTime.value).getTime() - eveInputToDate(startTime.value).getTime()) / 3_600_000
})

const timeError = computed(() => {
    if (startTime.value && endTime.value && eveInputToDate(endTime.value) <= eveInputToDate(startTime.value)) return 'End must be after start'
    if (timeSpanHours.value > 36) return 'Maximum 36 hour window'
    return null
})

// Step 2: Load & assign entities
const allEntities = ref<BattleEntity[]>([])
const sideA = ref<BattleEntity[]>([])
const sideB = ref<BattleEntity[]>([])
const loadingEntities = ref(false)
const entitiesLoaded = ref(false)
const entityFilter = ref('')

const undecided = computed(() => {
    const assignedIds = new Set([
        ...sideA.value.map(e => `${e.type}-${e.id}`),
        ...sideB.value.map(e => `${e.type}-${e.id}`),
    ])
    let list = allEntities.value.filter(e => !assignedIds.has(`${e.type}-${e.id}`))
    const q = entityFilter.value.toLowerCase().trim()
    if (q) {
        list = list.filter(e => e.name.toLowerCase().includes(q)
            || e.alliance_name?.toLowerCase().includes(q))
    }
    return list
})

const loadEntities = async () => {
    loadingEntities.value = true
    try {
        const result = await apiFetch<{ alliances: BattleEntity[]; corporations: BattleEntity[]; factions: BattleEntity[] }>('/api/battle/generator/entities', {
            method: 'POST',
            body: {
                systemIds: systems.value.map(s => s.id),
                startTime: eveInputToIso(startTime.value),
                endTime: eveInputToIso(endTime.value),
            },
        })

        // Alliances first, then standalone corps
        const entities: BattleEntity[] = [...result.factions]
        const allianceCorpIds = new Set<number>()

        for (const a of result.alliances) {
            entities.push(a)
        }

        // Track which corps belong to loaded alliances
        const loadedAllianceIds = new Set(result.alliances.map(a => a.id))
        for (const c of result.corporations) {
            if (c.alliance_id && loadedAllianceIds.has(c.alliance_id)) {
                allianceCorpIds.add(c.id)
            } else {
                entities.push(c)
            }
        }

        allEntities.value = entities
        // Store full corp list for alliance expansion
        allCorps.value = result.corporations

        // Carry any assignment already made through the reload instead of
        // clearing it. Going back to widen a window by ten minutes should not
        // cost you the sides you just sorted; entities that fell outside the
        // new window drop out on their own, and a first load has nothing to
        // keep, so the battle-edit seeding below still starts from empty.
        const stillListed = (side: BattleEntity[]) => side.filter(picked =>
            entities.some(e => e.type === picked.type && e.id === picked.id))
        sideA.value = stillListed(sideA.value)
        sideB.value = stillListed(sideB.value)

        entitiesLoaded.value = true
        currentStep.value = 2
    } catch (e: any) {
        console.error('Failed to load entities:', e)
    } finally {
        loadingEntities.value = false
    }
}

// Full corp list for showing alliance members
const allCorps = ref<BattleEntity[]>([])

// --- Prefill, for the battle page's "Edit sides" ---
//
// Battle sides are inferred from the killmails, and the inference is wrong
// often enough to be worth a human pass. "Edit sides" hands a report here with
// its system and window already filled in so the only work left is the part
// that needs judgement. With `battle=<id>` the result overwrites that battle's
// sides instead of creating a new one — unowned, so anyone signed in can fix a
// mis-sided battle.
//
// Sits below everything loadEntities touches: the watcher fires immediately,
// and anything declared later would still be in its temporal dead zone.
const route = useRoute()
const prefillSystemId = Number(route.query.system) || 0
const prefillStart = isoToEveInput(String(route.query.start ?? ''))
const prefillEnd = isoToEveInput(String(route.query.end ?? ''))
const hasPrefill = prefillSystemId > 0 && !!prefillStart && !!prefillEnd
const editingBattleId = Number(route.query.battle) || 0
const isEditing = computed(() => editingBattleId > 0)

if (hasPrefill) {
    systems.value = [{
        id: prefillSystemId,
        // Display only — every request keys off the id — so an edited name in
        // the URL misleads nobody but whoever typed it.
        name: String(route.query.systemName ?? '').slice(0, 60) || `System ${prefillSystemId}`,
    }]
    startTime.value = prefillStart
    endTime.value = prefillEnd
}

/**
 * Seed the two sides from the battle's current teams, so this is an edit
 * rather than a rebuild from nothing.
 *
 * Teams beyond the first two are left in Undecided on purpose: a battle row
 * can hold any number of teams but this wizard has exactly two, so a
 * multi-party battle cannot round-trip. Dropping those entities silently would
 * quietly delete sides; parking them where they have to be looked at does not.
 */
const droppedTeamCount = ref(0)

const prefillSidesFromBattle = async () => {
    if (!editingBattleId) return
    try {
        const existing = await apiFetch<any>(`/api/battle/${editingBattleId}`)
        const teams: any[] = Array.isArray(existing?.teams) ? existing.teams : []
        droppedTeamCount.value = Math.max(0, teams.length - 2)

        // Match the battle's members against the entity list the window
        // produced; anything no longer present there is simply skipped.
        const find = (type: 'alliance' | 'corporation', entityId: number) =>
            allEntities.value.find(e => e.type === type && e.id === entityId)

        const seed = (team: any): BattleEntity[] => {
            const picked: BattleEntity[] = []
            for (const group of team?.alliances ?? []) {
                if (group.alliance_id) {
                    const hit = find('alliance', group.alliance_id)
                    if (hit) picked.push(hit)
                    continue
                }
                for (const corp of group.corporations ?? []) {
                    const hit = find('corporation', corp.corporation_id)
                    if (hit) picked.push(hit)
                }
            }
            return picked
        }

        // Side names are not stored on battle_teams, so there is nothing to
        // restore — they stay at the defaults.
        sideA.value = seed(teams[0])
        sideB.value = seed(teams[1])
    } catch {
        // Fall back to a blank assignment — the window and system are still
        // filled in, so the user just does the sides from scratch.
    }
}

// Skip step 1 when asked. Auth resolves only after hydration, so this waits on
// it — firing on mount would silently do nothing for a signed-in user, since
// the entity endpoint needs the session.
const pendingAutoAdvance = ref(hasPrefill && route.query.step === 'sides')
watch(isAuthenticated, (authenticated) => {
    if (!authenticated || !pendingAutoAdvance.value) return
    pendingAutoAdvance.value = false
    void loadEntities().then(prefillSidesFromBattle)
}, { immediate: true })

const corpsForAlliance = (allianceId: number) =>
    allCorps.value.filter(c => c.alliance_id === allianceId)

// Move entities between sides
const moveToSide = (entity: BattleEntity, side: 'a' | 'b') => {
    // Remove from both sides first
    sideA.value = sideA.value.filter(e => !(e.type === entity.type && e.id === entity.id))
    sideB.value = sideB.value.filter(e => !(e.type === entity.type && e.id === entity.id))

    if (side === 'a') sideA.value.push(entity)
    else sideB.value.push(entity)
}

const moveToUndecided = (entity: BattleEntity) => {
    sideA.value = sideA.value.filter(e => !(e.type === entity.type && e.id === entity.id))
    sideB.value = sideB.value.filter(e => !(e.type === entity.type && e.id === entity.id))
}

const step2Ready = computed(() => sideA.value.length > 0 && sideB.value.length > 0)

// Step 3: Preview
const previewData = ref<any>(null)
const loadingPreview = ref(false)
const previewError = ref<string | null>(null)

const generatePreview = async () => {
    loadingPreview.value = true
    previewError.value = null
    try {
        const result = await apiFetch<any>('/api/battle/generator/preview', {
            method: 'POST',
            body: {
                systemIds: systems.value.map(s => s.id),
                startTime: eveInputToIso(startTime.value),
                endTime: eveInputToIso(endTime.value),
                sides: [
                    { name: sideAName.value, entities: sideA.value.map(e => ({ id: e.id, type: e.type })) },
                    { name: sideBName.value, entities: sideB.value.map(e => ({ id: e.id, type: e.type })) },
                ],
            },
        })
        previewData.value = result
        currentStep.value = 3
    } catch (e: any) {
        previewError.value = e.data?.message || 'Failed to generate preview'
    } finally {
        loadingPreview.value = false
    }
}

// Side names
const sideAName = ref('Team 1')
const sideBName = ref('Team 2')

// Save
const saving = ref(false)
const saveBattle = async () => {
    if (!previewData.value) return
    saving.value = true
    try {
        const result = await apiFetch<{ battle_id: number }>('/api/battle/generator/save', {
            method: 'POST',
            // battle_id present → rewrite that battle's sides in place rather
            // than creating another battle for the same fight.
            body: isEditing.value
                ? { ...previewData.value, battle_id: editingBattleId }
                : previewData.value,
        })
        await navigateTo(`/battle/${result.battle_id}`)
    } catch (e: any) {
        console.error('Failed to save battle:', e)
    } finally {
        saving.value = false
    }
}

// Formatting

const entityImage = (entity: BattleEntity): string => {
    if (entity.type === 'alliance') return `/images/alliances/${entity.id}/logo?size=64`
    if (entity.type === 'faction') return `/images/factions/${entity.id}/logo?size=64`
    return `/images/corporations/${entity.id}/logo?size=64`
}

// Expanded alliances in side panels
const expandedAlliances = ref<Set<number>>(new Set())
const toggleAllianceExpand = (id: number) => {
    if (expandedAlliances.value.has(id)) expandedAlliances.value.delete(id)
    else expandedAlliances.value.add(id)
}

// ── Wizard chrome ────────────────────────────────────────────────────────────
const WIZARD_STEPS = [
    { n: 1, label: 'Location & time', hint: 'Where and when' },
    { n: 2, label: 'Assign sides', hint: 'Who fought whom' },
    { n: 3, label: 'Preview & save', hint: 'Check, then publish' },
]

/** What still blocks "Load Entities", phrased as the thing to go do. */
const step1Checklist = computed(() => {
    const hasWindow = !!startTime.value && !!endTime.value
    return [
        {
            label: 'Pick up to 5 solar systems',
            ok: systems.value.length > 0,
            detail: systems.value.length
                ? `${systems.value.length} selected`
                : 'Search for the system the fight happened in',
        },
        {
            label: 'Set a start and end time',
            ok: hasWindow,
            detail: hasWindow ? null : 'Both ends of the window are required',
        },
        {
            label: 'Keep the window under 36 hours',
            ok: hasWindow && !timeError.value,
            detail: timeError.value ?? (timeSpanHours.value > 0 ? `${timeSpanHours.value.toFixed(1)}h window` : null),
        },
    ]
})

const assignedCount = computed(() => sideA.value.length + sideB.value.length)
</script>
<template>
    <div>
        <!-- Header -->
        <div class="glass-panel p-5 mb-4">
            <div class="flex items-start justify-between gap-4 flex-wrap">
                <div class="min-w-0">
                    <h1 class="flex items-center gap-2.5 text-xl font-bold text-white">
                        <span class="w-8 h-8 rounded-lg bg-blue-500/15 flex items-center justify-center flex-shrink-0">
                            <Icon name="lucide:swords" class="text-base text-blue-400" />
                        </span>
                        Battle Generator
                    </h1>
                    <p class="text-sm text-gray-500 mt-2 max-w-3xl">
                        Build a custom battle report: pick the systems and the window the fight happened in, sort the
                        participants into sides, then preview and save it.
                    </p>
                </div>
                <NuxtLink to="/battles"
                    class="inline-flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium text-gray-400 bg-white/[0.04] border border-white/[0.08] hover:bg-white/[0.07] hover:text-gray-200 transition-colors flex-shrink-0">
                    <Icon name="lucide:arrow-left" class="text-xs" />
                    All battles
                </NuxtLink>
            </div>
        </div>

        <!-- Login gate -->
        <LoginGate v-if="!isAuthenticated" message="You must be logged in to use the Battle Generator" />

        <template v-if="isAuthenticated">
        <WizardSteps :steps="WIZARD_STEPS" :current="currentStep" class="mb-4" @select="currentStep = $event" />

        <!-- ── Step 1: Location & Time ──────────────────────────────────── -->
        <div v-if="currentStep === 1" class="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_340px] gap-4 items-start">
            <div class="space-y-4 min-w-0">
                <!-- System search -->
                <div class="glass-panel p-4">
                    <div class="flex items-center justify-between gap-3 mb-3">
                        <div>
                            <div class="text-xs font-semibold text-gray-200">Solar systems</div>
                            <div class="text-fine text-gray-600 mt-0.5">Kills in any of these systems are considered.</div>
                        </div>
                        <span class="text-fine text-gray-600 tabular-nums flex-shrink-0">{{ systems.length }}/5</span>
                    </div>

                    <div class="relative">
                        <div class="glass-panel flex items-center gap-2 px-3 py-2.5 focus-within:border-blue-500/40 transition-colors">
                            <Icon name="lucide:search" class="text-base text-gray-500 flex-shrink-0" />
                            <input
                                v-model="systemSearch"
                                type="text"
                                :placeholder="systemLimitReached ? 'Maximum 5 systems reached' : 'Search for a solar system...'"
                                :disabled="systemLimitReached"
                                class="flex-1 min-w-0 bg-transparent text-white text-sm outline-none placeholder-gray-600 disabled:opacity-40"
                                @input="handleSystemSearch"
                            />
                            <Icon v-if="isSearchingSystems" name="lucide:loader-2" class="text-base text-gray-500 animate-spin" />
                        </div>

                        <!-- Dropdown -->
                        <div v-if="showSystemDropdown" class="absolute z-50 mt-1 w-full rounded-lg bg-black/90 backdrop-blur-xl border border-white/[0.08] shadow-2xl max-h-60 overflow-y-auto">
                            <button
                                v-for="hit in systemResults"
                                :key="hit.id"
                                @click="addSystem(hit)"
                                class="w-full flex items-center gap-2.5 px-3 py-2.5 border-b border-white/[0.04] last:border-b-0 hover:bg-blue-500/[0.06] transition-colors text-left cursor-pointer"
                            >
                                <img :src="`/images/systems/${parseId(hit.id)}?size=64`" class="w-7 h-7 rounded flex-shrink-0 bg-gray-900" loading="lazy" :alt="hit.name">
                                <div>
                                    <div class="text-sm text-gray-200">{{ hit.name }}</div>
                                    <div class="text-fine text-gray-500">Solar System</div>
                                </div>
                            </button>
                        </div>
                    </div>

                    <!-- Selected systems -->
                    <div v-if="systems.length" class="flex flex-wrap gap-1.5 mt-3">
                        <div v-for="sys in systems" :key="sys.id"
                            class="inline-flex items-center gap-1.5 px-2 py-1 rounded border border-amber-500/30 bg-amber-500/10 text-xs text-amber-400 font-medium">
                            <img :src="`/images/systems/${sys.id}?size=32`" class="w-4 h-4 rounded" loading="lazy" :alt="sys.name">
                            {{ sys.name }}
                            <button @click="removeSystem(sys.id)" class="ml-0.5 opacity-60 hover:opacity-100 cursor-pointer">
                                <Icon name="lucide:x" class="text-fine" />
                            </button>
                        </div>
                    </div>
                    <p v-else class="text-fine text-gray-600 mt-3">No systems picked yet — start typing above.</p>
                </div>

                <!-- Time range -->
                <div class="glass-panel p-4">
                    <div class="mb-3">
                        <div class="text-xs font-semibold text-gray-200">Time range</div>
                        <div class="text-fine text-gray-600 mt-0.5">EVE time (UTC). Up to a 36 hour window.</div>
                    </div>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                        <div>
                            <label class="text-fine text-gray-500 mb-1 block">Start</label>
                            <DateTimePicker v-model="startTime" placeholder="Pick a start" />
                        </div>
                        <div>
                            <label class="text-fine text-gray-500 mb-1 block">End</label>
                            <DateTimePicker v-model="endTime" placeholder="Pick an end" />
                        </div>
                    </div>
                    <div v-if="timeError" class="mt-2 flex items-center gap-1.5 text-xs text-red-400">
                        <Icon name="lucide:alert-circle" class="text-xs flex-shrink-0" />{{ timeError }}
                    </div>
                    <div v-else-if="timeSpanHours > 0" class="mt-2 text-fine text-gray-600 tabular-nums">{{ timeSpanHours.toFixed(1) }}h window</div>
                </div>
            </div>

            <!-- Sticky rail -->
            <aside class="xl:sticky xl:top-4 glass-panel p-4">
                <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-3">Before loading</div>
                <ul class="space-y-2">
                    <li v-for="item in step1Checklist" :key="item.label" class="flex items-start gap-2.5">
                        <Icon :name="item.ok ? 'lucide:check-circle-2' : 'lucide:circle'"
                            class="text-sm mt-px flex-shrink-0" :class="item.ok ? 'text-isk' : 'text-gray-600'" />
                        <div class="min-w-0">
                            <div class="text-xs" :class="item.ok ? 'text-gray-400' : 'text-gray-300'">{{ item.label }}</div>
                            <div v-if="item.detail" class="text-fine mt-0.5" :class="item.ok ? 'text-gray-600' : 'text-amber-400/80'">{{ item.detail }}</div>
                        </div>
                    </li>
                </ul>

                <button
                    @click="loadEntities"
                    :disabled="!step1Ready || loadingEntities || !!timeError"
                    class="w-full mt-4 py-2.5 rounded-lg text-sm font-medium transition-colors"
                    :class="step1Ready && !timeError && !loadingEntities
                        ? 'bg-blue-500 text-white hover:bg-blue-600 cursor-pointer'
                        : 'bg-white/[0.04] text-gray-600 cursor-not-allowed'"
                >
                    <Icon v-if="loadingEntities" name="lucide:loader-2" class="text-base animate-spin mr-2" />
                    {{ loadingEntities ? 'Loading entities…' : 'Load Entities' }}
                </button>
                <p class="text-fine text-gray-600 text-center mt-2">
                    Everyone who shot or died in that window gets pulled in for you to sort.
                </p>
            </aside>
        </div>

        <!-- ── Step 2: Assign Sides ─────────────────────────────────────── -->
        <div v-if="currentStep === 2" class="space-y-4">
            <div v-if="isEditing" class="rounded-lg border border-blue-500/20 bg-blue-500/[0.06] px-3 py-2.5 text-xs text-blue-300 flex items-start gap-2">
                <Icon name="lucide:pencil" class="text-sm flex-shrink-0 mt-px" />
                <div>
                    Editing the sides of
                    <NuxtLink :to="`/battle/${editingBattleId}`" class="font-medium underline hover:text-blue-200">battle #{{ editingBattleId }}</NuxtLink>.
                    Saving overwrites its sides — battles are correctable by anyone signed in.
                    <span v-if="droppedTeamCount > 0" class="block mt-1 text-amber-300">
                        This battle had {{ droppedTeamCount + 2 }} sides and the generator holds two.
                        The extra {{ droppedTeamCount === 1 ? 'side is' : 'sides are' }} waiting in Undecided —
                        place them before saving or they are dropped.
                    </span>
                </div>
            </div>

            <!-- Side names -->
            <div class="grid grid-cols-2 gap-3">
                <div class="relative rounded-lg bg-red-500/5 border border-red-500/20 p-3 pl-4 overflow-hidden">
                    <div class="absolute inset-y-0 left-0 w-[3px] bg-red-500" />
                    <label class="text-fine font-bold uppercase tracking-wider text-red-400/60 mb-1 block">Side 1 name</label>
                    <input v-model="sideAName" type="text" class="w-full px-2 py-1.5 rounded bg-white/[0.04] border border-white/[0.06] text-sm text-white focus:border-red-500/40 focus:outline-none" />
                </div>
                <div class="relative rounded-lg bg-blue-500/5 border border-blue-500/20 p-3 pl-4 overflow-hidden">
                    <div class="absolute inset-y-0 left-0 w-[3px] bg-blue-500" />
                    <label class="text-fine font-bold uppercase tracking-wider text-blue-400/60 mb-1 block">Side 2 name</label>
                    <input v-model="sideBName" type="text" class="w-full px-2 py-1.5 rounded bg-white/[0.04] border border-white/[0.06] text-sm text-white focus:border-blue-500/40 focus:outline-none" />
                </div>
            </div>

            <!-- Filter -->
            <div class="glass-panel flex items-center gap-2 px-3 py-2">
                <Icon name="lucide:filter" class="text-sm text-gray-500 flex-shrink-0" />
                <input
                    v-model="entityFilter"
                    type="text"
                    placeholder="Filter entities..."
                    class="flex-1 min-w-0 bg-transparent text-sm text-gray-300 outline-none placeholder-gray-600"
                />
                <span class="text-fine text-gray-600 tabular-nums flex-shrink-0">{{ undecided.length }} unassigned</span>
            </div>

            <!-- Three-column layout: Side A | Unassigned | Side B -->
            <div class="grid grid-cols-[48px_1fr_48px] md:grid-cols-3 gap-2 md:gap-3">
                <!-- Side A (left) -->
                <div class="relative rounded-lg bg-white/[0.04] border border-red-500/20 p-1.5 md:p-3 md:pl-4 min-h-[300px] max-h-[600px] overflow-y-auto">
                    <div class="hidden md:block absolute inset-y-0 left-0 w-[3px] bg-red-500" />
                    <div class="hidden md:block text-fine font-bold uppercase tracking-wider text-red-400 mb-3">{{ sideAName }} ({{ sideA.length }})</div>
                    <div class="md:hidden text-fine font-bold text-red-400 text-center mb-2">{{ sideA.length }}</div>
                    <div class="space-y-0.5">
                        <div v-for="entity in sideA" :key="`a-${entity.type}-${entity.id}`"
                            class="group flex items-center gap-2 px-1 md:px-2 py-1.5 rounded hover:bg-blue-500/[0.04] transition-colors">
                            <img :src="entityImage(entity)" class="w-7 h-7 md:w-5 md:h-5 rounded flex-shrink-0 bg-gray-900 cursor-pointer" loading="lazy" :alt="entity.name" v-tooltip="entity.name" @click="moveToUndecided(entity)">
                            <div class="hidden md:block flex-1 min-w-0">
                                <div class="text-xs text-gray-200 truncate">{{ entity.name }}</div>
                                <div class="text-fine text-gray-600 capitalize">{{ entity.type }}</div>
                            </div>
                            <button @click="moveToUndecided(entity)" class="hidden md:block opacity-0 group-hover:opacity-100 px-1 py-0.5 text-fine rounded bg-white/[0.06] text-gray-400 hover:bg-blue-500/[0.15] transition-opacity cursor-pointer" v-tooltip="'Remove'">
                                <Icon name="lucide:x" class="text-fine" />
                            </button>
                        </div>
                        <div v-if="!sideA.length" class="text-center py-8 text-xs text-gray-600">
                            <Icon name="lucide:arrow-left" class="text-base text-red-500/30 mb-1" /><br>
                            <span class="hidden md:inline">Use the arrows to add entities</span>
                        </div>
                    </div>
                </div>

                <!-- Unassigned (center) — arrows on opposite sides -->
                <div class="glass-panel p-2 md:p-3 min-h-[300px] max-h-[600px] overflow-y-auto">
                    <div class="text-fine font-bold uppercase tracking-wider text-gray-500 mb-3">
                        Unassigned ({{ undecided.length }})
                    </div>
                    <div class="space-y-0.5">
                        <div v-for="entity in undecided" :key="`u-${entity.type}-${entity.id}`"
                            class="group relative flex items-center gap-1.5 px-1 py-1.5 rounded hover:bg-blue-500/[0.06] transition-colors">
                            <!-- Desktop: left arrow -->
                            <button @click="moveToSide(entity, 'a')" class="hidden md:flex flex-shrink-0 w-6 h-6 items-center justify-center rounded bg-red-500/10 text-red-400/40 hover:bg-red-500/20 hover:text-red-400 transition-colors cursor-pointer" v-tooltip="'Move to Side 1'">
                                <Icon name="lucide:arrow-left" class="text-xs" />
                            </button>

                            <img :src="entityImage(entity)" class="w-8 h-8 md:w-5 md:h-5 rounded flex-shrink-0 bg-gray-900" loading="lazy" :alt="entity.name">
                            <div class="flex-1 min-w-0">
                                <div class="text-xs text-gray-300 truncate">{{ entity.name }}</div>
                                <div class="text-fine text-gray-600 capitalize truncate">
                                    {{ entity.type }}
                                    <template v-if="entity.type === 'corporation' && entity.alliance_name">
                                        · {{ entity.alliance_name }}
                                    </template>
                                    <template v-if="entity.type === 'alliance'">
                                        · {{ corpsForAlliance(entity.id).length }} corps
                                    </template>
                                </div>
                            </div>

                            <!-- Desktop: right arrow -->
                            <button @click="moveToSide(entity, 'b')" class="hidden md:flex flex-shrink-0 w-6 h-6 items-center justify-center rounded bg-blue-500/10 text-blue-400/40 hover:bg-blue-500/20 hover:text-blue-400 transition-colors cursor-pointer" v-tooltip="'Move to Side 2'">
                                <Icon name="lucide:arrow-right" class="text-xs" />
                            </button>

                            <!-- Mobile: overlay buttons -->
                            <div class="md:hidden flex gap-1 flex-shrink-0">
                                <button @click="moveToSide(entity, 'a')" class="w-8 h-8 flex items-center justify-center rounded bg-red-500/20 text-red-400 active:bg-red-500/40 transition-colors" v-tooltip="'Move to Side 1'">
                                    <Icon name="lucide:arrow-left" class="text-sm" />
                                </button>
                                <button @click="moveToSide(entity, 'b')" class="w-8 h-8 flex items-center justify-center rounded bg-blue-500/20 text-blue-400 active:bg-blue-500/40 transition-colors" v-tooltip="'Move to Side 2'">
                                    <Icon name="lucide:arrow-right" class="text-sm" />
                                </button>
                            </div>
                        </div>
                        <div v-if="!undecided.length && allEntities.length" class="text-center py-8 text-xs text-isk/60">
                            All entities assigned
                        </div>
                        <div v-else-if="!allEntities.length" class="text-center py-8 text-xs text-gray-600">
                            No entities found for this window
                        </div>
                    </div>
                </div>

                <!-- Side B (right) -->
                <div class="relative rounded-lg bg-white/[0.04] border border-blue-500/20 p-1.5 md:p-3 md:pl-4 min-h-[300px] max-h-[600px] overflow-y-auto">
                    <div class="hidden md:block absolute inset-y-0 left-0 w-[3px] bg-blue-500" />
                    <div class="hidden md:block text-fine font-bold uppercase tracking-wider text-blue-400 mb-3">{{ sideBName }} ({{ sideB.length }})</div>
                    <div class="md:hidden text-fine font-bold text-blue-400 text-center mb-2">{{ sideB.length }}</div>
                    <div class="space-y-0.5">
                        <div v-for="entity in sideB" :key="`b-${entity.type}-${entity.id}`"
                            class="group flex items-center gap-2 px-1 md:px-2 py-1.5 rounded hover:bg-blue-500/[0.04] transition-colors">
                            <img :src="entityImage(entity)" class="w-7 h-7 md:w-5 md:h-5 rounded flex-shrink-0 bg-gray-900 cursor-pointer" loading="lazy" :alt="entity.name" v-tooltip="entity.name" @click="moveToUndecided(entity)">
                            <div class="hidden md:block flex-1 min-w-0">
                                <div class="text-xs text-gray-200 truncate">{{ entity.name }}</div>
                                <div class="text-fine text-gray-600 capitalize">{{ entity.type }}</div>
                            </div>
                            <button @click="moveToUndecided(entity)" class="hidden md:block opacity-0 group-hover:opacity-100 px-1 py-0.5 text-fine rounded bg-white/[0.06] text-gray-400 hover:bg-blue-500/[0.15] transition-opacity cursor-pointer" v-tooltip="'Remove'">
                                <Icon name="lucide:x" class="text-fine" />
                            </button>
                        </div>
                        <div v-if="!sideB.length" class="text-center py-8 text-xs text-gray-600">
                            <Icon name="lucide:arrow-right" class="text-base text-blue-500/30 mb-1" /><br>
                            <span class="hidden md:inline">Use the arrows to add entities</span>
                        </div>
                    </div>
                </div>
            </div>

            <div v-if="previewError" class="flex items-start gap-2 text-sm text-red-400 bg-red-500/10 rounded-lg px-4 py-2">
                <Icon name="lucide:alert-circle" class="text-base mt-0.5 flex-shrink-0" />
                <span class="min-w-0">{{ previewError }}</span>
            </div>

            <!-- Sticky action bar -->
            <div class="sticky bottom-4 z-20 flex items-center gap-3 rounded-lg border border-white/[0.10] bg-black/80 backdrop-blur-xl p-3 shadow-2xl shadow-black/60">
                <button
                    @click="currentStep = 1"
                    class="px-4 py-2.5 rounded-lg text-sm font-medium bg-white/[0.04] text-gray-400 border border-white/[0.08] hover:bg-blue-500/[0.08] transition-colors cursor-pointer flex-shrink-0"
                >
                    Back
                </button>
                <div class="min-w-0 flex-1 text-fine">
                    <div v-if="step2Ready" class="text-gray-400">
                        <span class="text-red-400">{{ sideA.length }}</span> vs <span class="text-blue-400">{{ sideB.length }}</span>
                        · {{ assignedCount }} of {{ allEntities.length }} assigned
                    </div>
                    <div v-else class="text-amber-400/80">Both sides need at least one entity.</div>
                </div>
                <button
                    @click="generatePreview"
                    :disabled="!step2Ready || loadingPreview"
                    class="px-5 py-2.5 rounded-lg text-sm font-medium transition-colors flex-shrink-0"
                    :class="step2Ready && !loadingPreview
                        ? 'bg-blue-500 text-white hover:bg-blue-600 cursor-pointer'
                        : 'bg-white/[0.04] text-gray-600 cursor-not-allowed'"
                >
                    <Icon v-if="loadingPreview" name="lucide:loader-2" class="text-base animate-spin mr-2" />
                    {{ loadingPreview ? 'Generating…' : 'Generate Preview' }}
                </button>
            </div>
        </div>

        <!-- ── Step 3: Preview & Save ───────────────────────────────────── -->
        <div v-if="currentStep === 3 && previewData" class="space-y-4">
            <!-- Preview header -->
            <div class="glass-panel p-5">
                <div class="flex items-center gap-3 mb-4 pb-3 border-b border-white/[0.06]">
                    <img :src="`/images/systems/${previewData.solar_system_id}?size=64`" alt="" class="w-10 h-10 rounded-lg flex-shrink-0 bg-gray-900" loading="eager">
                    <div class="min-w-0">
                        <span class="text-lg font-bold text-white" :class="pochvenClass(previewData.region_id)">{{ previewData.solar_system_name }}</span>
                        <span class="ml-2 text-sm text-gray-500" :class="pochvenClass(previewData.region_id)">{{ previewData.region_name }}</span>
                    </div>
                    <span class="ml-auto px-2 py-0.5 text-fine rounded bg-purple-500/20 text-purple-400 font-medium flex-shrink-0">Preview</span>
                </div>

                <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
                    <div v-for="tile in [
                        { label: 'Ships Lost', icon: 'lucide:crosshair', value: formatNumber(previewData.kill_count), color: 'text-white' },
                        { label: 'ISK Destroyed', icon: 'lucide:coins', value: formatIsk(previewData.total_isk_destroyed), color: 'text-yellow-400' },
                        { label: 'Duration', icon: 'lucide:clock', value: `${previewData.duration_minutes}m`, color: 'text-white' },
                        { label: 'Characters', icon: 'lucide:user-round', value: formatNumber(previewData.characters_involved), color: 'text-blue-400' },
                        { label: 'Corporations', icon: 'lucide:building-2', value: formatNumber(previewData.corporations_involved), color: 'text-purple-400' },
                        { label: 'Alliances', icon: 'lucide:shield', value: formatNumber(previewData.alliances_involved), color: 'text-indigo-400' },
                        { label: 'Damage', icon: 'lucide:zap', value: formatNumber(previewData.total_damage), color: 'text-gray-300' },
                    ]" :key="tile.label" class="rounded-lg bg-white/[0.03] border border-white/[0.06] p-3 text-center">
                        <div class="text-base font-bold tabular-nums" :class="tile.color">{{ tile.value }}</div>
                        <div class="flex items-center justify-center gap-1.5 text-fine text-gray-500 uppercase tracking-wider mt-0.5">
                            <Icon :name="tile.icon" class="text-fine text-gray-600" />{{ tile.label }}
                        </div>
                    </div>
                </div>
            </div>

            <!-- Team summary cards -->
            <div v-if="previewData.teams?.length >= 2" class="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] gap-4 items-center">
                <div class="relative rounded-lg bg-white/[0.04] border border-red-500/20 p-4 pl-5 overflow-hidden">
                    <div class="absolute inset-y-0 left-0 w-[3px] bg-red-500" />
                    <div class="text-sm font-bold text-white mb-2">{{ previewData.teams[0].name || sideAName }}</div>
                    <div class="grid grid-cols-2 gap-2 text-xs">
                        <div><span class="text-green-400 tabular-nums font-medium">{{ formatNumber(previewData.teams[0].total_kills) }}</span><div class="text-gray-500">kills</div></div>
                        <div><span class="text-red-400 tabular-nums font-medium">{{ formatNumber(previewData.teams[0].total_losses) }}</span><div class="text-gray-500">losses</div></div>
                        <div><span class="text-green-400 tabular-nums font-medium">{{ formatIsk(previewData.teams[0].total_isk_destroyed) }}</span><div class="text-gray-500">destroyed</div></div>
                        <div><span class="text-red-400 tabular-nums font-medium">{{ formatIsk(previewData.teams[0].total_isk_lost) }}</span><div class="text-gray-500">lost</div></div>
                    </div>
                </div>

                <div class="flex flex-col items-center gap-1 px-4">
                    <div class="text-xl font-bold text-gray-600">VS</div>
                </div>

                <div class="relative rounded-lg bg-white/[0.04] border border-blue-500/20 p-4 pl-5 overflow-hidden">
                    <div class="absolute inset-y-0 left-0 w-[3px] bg-blue-500" />
                    <div class="text-sm font-bold text-white mb-2">{{ previewData.teams[1].name || sideBName }}</div>
                    <div class="grid grid-cols-2 gap-2 text-xs">
                        <div><span class="text-green-400 tabular-nums font-medium">{{ formatNumber(previewData.teams[1].total_kills) }}</span><div class="text-gray-500">kills</div></div>
                        <div><span class="text-red-400 tabular-nums font-medium">{{ formatNumber(previewData.teams[1].total_losses) }}</span><div class="text-gray-500">losses</div></div>
                        <div><span class="text-green-400 tabular-nums font-medium">{{ formatIsk(previewData.teams[1].total_isk_destroyed) }}</span><div class="text-gray-500">destroyed</div></div>
                        <div><span class="text-red-400 tabular-nums font-medium">{{ formatIsk(previewData.teams[1].total_isk_lost) }}</span><div class="text-gray-500">lost</div></div>
                    </div>
                </div>
            </div>

            <!-- Alliance breakdown -->
            <BattleSummary v-if="previewData.teams" :teams="previewData.teams" />

            <!-- Sticky action bar -->
            <div class="sticky bottom-4 z-20 flex items-center gap-3 rounded-lg border border-white/[0.10] bg-black/80 backdrop-blur-xl p-3 shadow-2xl shadow-black/60">
                <button
                    @click="currentStep = 2"
                    class="px-4 py-2.5 rounded-lg text-sm font-medium bg-white/[0.04] text-gray-400 border border-white/[0.08] hover:bg-blue-500/[0.08] transition-colors cursor-pointer flex-shrink-0"
                >
                    Back to sides
                </button>
                <div class="min-w-0 flex-1 text-fine text-gray-600">
                    Saving publishes this as a custom battle report at its own URL.
                </div>
                <button
                    v-if="isAuthenticated"
                    @click="saveBattle"
                    :disabled="saving"
                    class="px-5 py-2.5 rounded-lg text-sm font-medium bg-green-500 text-white hover:bg-green-600 transition-colors cursor-pointer flex-shrink-0 disabled:opacity-60"
                >
                    <Icon v-if="saving" name="lucide:loader-2" class="text-base animate-spin mr-2" />
                    {{ saving ? 'Saving…' : isEditing ? `Update Battle #${editingBattleId}` : 'Save Battle Report' }}
                </button>
                <button
                    v-else
                    @click="login({ redirect: '/battlegenerator' })"
                    class="px-5 py-2.5 rounded-lg text-sm font-medium bg-white/[0.06] text-gray-400 border border-white/[0.08] hover:bg-blue-500/[0.08] transition-colors cursor-pointer flex-shrink-0"
                >
                    <Icon name="lucide:log-in" class="text-base mr-2" />
                    Log in to save
                </button>
            </div>
        </div>
        </template>
    </div>
</template>
