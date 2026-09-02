<script setup lang="ts">
import { killmailFitToUiFit } from "~/composables/fit/killmailToFit";
import type { Fit, FitSlotType } from "~/composables/fit/types";
import { fitToEngine } from "~/composables/fit/toEngine";
import { calculateFit } from "@evekill/dogma";
import { calculateFitCost } from "~/composables/useFitCost";

interface FittingModule {
    slot_group: number;
    ordinal: number;
    type_id: number;
    name: string | null;
    charge_type_id: number | null;
}

interface FittingDrone {
    type_id: number;
    name: string | null;
    quantity: number;
}

interface FittingFamily {
    family_hash: string;
    total_uses: number;
    variant_count: number;
    last_used: string;
    fit_cost: number;
    modules: FittingModule[];
    drones: FittingDrone[];
    context?: FitFamilyContext;
}

interface FittingsResponse {
    window_days: number;
    hull_cost: number | null;
    families: FittingFamily[];
}

const { currentFit } = useCurrentFit();
const { sde } = useEveData();
const loader = useFitLoader();
const fitManager = useFitManager();
const capacity = useHullCapacity();
const { stats: currentStats } = useFitStatistics();
const data = ref<FittingsResponse | null>(null);
const pending = ref(false);
const selectedFamily = ref<string | null>(null);
const customFit = ref<Fit | null>(null);
const loadingFit = ref(false);
const observedSnapshot = ref<Fit | null>(null);
const customStats = shallowRef<any>(null);
const customPrice = ref(0);
const { total: currentPrice } = useFitCost();
const showComparison = ref(false);
let requestGeneration = 0;

function cloneFit(fit: Fit): Fit {
    return structuredClone(toRaw(fit));
}

onMounted(() => {
    if (currentFit.value) customFit.value = cloneFit(currentFit.value);
});

watch(currentFit, (fit) => {
    if (!fit || loadingFit.value || selectedFamily.value) return;
    customFit.value = cloneFit(fit);
}, { deep: true });

watch(customFit, async (fit) => {
    const [stats, price] = fit
        ? await Promise.all([calculateFit(fitToEngine(fit)), calculateFitCost(fit)])
        : [null, 0];
    customStats.value = stats ? markRaw(stats) : null;
    customPrice.value = price;
}, { deep: true });

const shipTypeId = computed(() => currentFit.value?.shipTypeId ?? 0);
const shipName = computed(() =>
    sde.value?.types.get(shipTypeId.value)?.name ?? "Current hull",
);
const shipGroupName = computed(() => {
    const type = sde.value?.types.get(shipTypeId.value);
    return type ? sde.value?.groups.get(type.groupID)?.name ?? null : null;
});

const suggestionSlots: Array<{ group: number; slot: FitSlotType; label: string }> = [
    { group: 1, slot: "High", label: "High slots" },
    { group: 2, slot: "Medium", label: "Mid slots" },
    { group: 3, slot: "Low", label: "Low slots" },
    { group: 4, slot: "Rig", label: "Rigs" },
    { group: 5, slot: "SubSystem", label: "Subsystems" },
];

function typeAttribute(typeId: number, name: string): number {
    const sdeData = sde.value;
    if (!sdeData) return 0;
    const attributeId = sdeData.attributeNameToId.get(name);
    if (attributeId === undefined) return 0;
    const attributes = sdeData.typeDogma.get(typeId)?.dogmaAttributes as Array<{ attributeID: number; value: number }> | undefined;
    return attributes?.find(attribute => attribute.attributeID === attributeId)?.value ?? 0;
}

const fitSuggestions = computed(() => {
    if (!currentFit.value || !data.value) return [];
    const fitted = new Set(currentFit.value.modules.map(module => module.typeId));
    const hasFittedModules = fitted.size > 0;
    const cpuFree = Math.max(0, capacity.cpu.value.total - capacity.cpu.value.used);
    const powerFree = Math.max(0, capacity.power.value.total - capacity.power.value.used);

    return suggestionSlots.flatMap(({ group, slot, label }) => {
        if (capacity.slotsUsed.value[slot] >= capacity.slotCounts.value[slot]) return [];

        const scores = new Map<number, {
            typeId: number;
            name: string;
            score: number;
            families: number;
            cpu: number;
            power: number;
            uses: number;
        }>();
        for (const family of data.value!.families) {
            const overlap = family.modules.filter(module => fitted.has(module.type_id)).length;
            if (hasFittedModules && overlap === 0) continue;
            const relevance = Math.max(1, overlap);
            const candidates = new Map(
                family.modules
                    .filter(module => module.slot_group === group)
                    .map(module => [module.type_id, module]),
            );
            for (const module of candidates.values()) {
                const cpu = typeAttribute(module.type_id, "cpu");
                const power = typeAttribute(module.type_id, "power");
                if (cpu > cpuFree || power > powerFree) continue;
                if (slot === "High" && !capacity.hasHardpointFor(module.type_id)) continue;
                const existing = scores.get(module.type_id) ?? {
                    typeId: module.type_id,
                    name: module.name ?? sde.value?.types.get(module.type_id)?.name ?? `Type ${module.type_id}`,
                    score: 0,
                    families: 0,
                    cpu,
                    power,
                    uses: 0,
                };
                existing.score += family.total_uses * relevance;
                existing.families++;
                existing.uses += family.total_uses;
                scores.set(module.type_id, existing);
            }
        }
        const suggestions = [...scores.values()].sort((a, b) => b.score - a.score).slice(0, 2);
        const observedTotal = data.value!.families.reduce((sum, family) => sum + family.total_uses, 0);
        return suggestions.length ? [{
            slot,
            label,
            suggestions: suggestions.map(suggestion => ({
                ...suggestion,
                frequency: observedTotal > 0 ? Math.round(suggestion.uses / observedTotal * 100) : 0,
            })),
        }] : [];
    });
});

watch(shipTypeId, async (id) => {
    const generation = ++requestGeneration;
    selectedFamily.value = null;
    data.value = null;
    if (!id) return;
    pending.value = true;
    try {
        const response = await apiFetch<FittingsResponse>(`/api/item/${id}/fittings`);
        if (generation === requestGeneration) data.value = response;
    } finally {
        if (generation === requestGeneration) pending.value = false;
    }
}, { immediate: true });

function loadFamily(family: FittingFamily, index: number) {
    loadingFit.value = true;
    selectedFamily.value = family.family_hash;
    showComparison.value = true;
    const next = killmailFitToUiFit({
        shipTypeId: shipTypeId.value,
        modules: family.modules,
        drones: family.drones,
        name: `${shipName.value} Observed Fit #${index + 1}`,
        description: `${family.total_uses} observed losses in the last ${data.value?.window_days ?? 90} days.`,
    });
    observedSnapshot.value = cloneFit(next);
    loader.loadFit(next);
    nextTick(() => { loadingFit.value = false; });
}

function restoreCustomFit() {
    if (!customFit.value) return;
    loadingFit.value = true;
    selectedFamily.value = null;
    observedSnapshot.value = null;
    showComparison.value = false;
    loader.loadFit(cloneFit(customFit.value));
    nextTick(() => { loadingFit.value = false; });
}

function keepAsCustomFit() {
    if (!currentFit.value) return;
    customFit.value = cloneFit(currentFit.value);
    selectedFamily.value = null;
    observedSnapshot.value = null;
    showComparison.value = false;
}

function resetObservedFit() {
    if (!observedSnapshot.value) return;
    loadingFit.value = true;
    loader.loadFit(cloneFit(observedSnapshot.value));
    nextTick(() => { loadingFit.value = false; });
}

const observedModified = computed(() => Boolean(
    selectedFamily.value && observedSnapshot.value && currentFit.value
    && JSON.stringify(toRaw(observedSnapshot.value)) !== JSON.stringify(toRaw(currentFit.value)),
));

const editorMode = computed(() => selectedFamily.value
    ? observedModified.value ? "Modified observed fit" : "Previewing observed fit"
    : "Editing custom fit",
);

function hullStat(stats: any, name: string): number {
    const id = sde.value?.attributeNameToId.get(name);
    if (id === undefined || !stats) return 0;
    const attr = stats.hull.attributes.get(id);
    return Number(attr?.value ?? attr?.base_value ?? 0);
}

const comparisonMetrics = computed(() => {
    if (!customStats.value || !currentStats.value || !selectedFamily.value) return [];
    return [
        ["DPS", "damagePerSecondWithReload", ""],
        ["EHP", "ehp", ""],
        ["Speed", "maxVelocity", "m/s"],
        ["CPU free", "cpuFree", "tf"],
        ["PG free", "powerFree", "MW"],
    ].map(([label, attr, unit]) => {
        const custom = hullStat(customStats.value, attr!);
        const observed = hullStat(currentStats.value, attr!);
        return { label, custom, observed, delta: observed - custom, unit };
    });
});

const moduleDifference = computed(() => {
    if (!customFit.value || !currentFit.value || !selectedFamily.value) return { added: 0, removed: 0, addedNames: [], removedNames: [] };
    const counts = (fit: Fit) => {
        const map = new Map<number, number>();
        for (const module of fit.modules) map.set(module.typeId, (map.get(module.typeId) ?? 0) + 1);
        return map;
    };
    const custom = counts(customFit.value);
    const observed = counts(currentFit.value);
    let added = 0;
    let removed = 0;
    const addedNames: string[] = [];
    const removedNames: string[] = [];
    for (const [typeId, count] of observed) {
        const difference = Math.max(0, count - (custom.get(typeId) ?? 0));
        added += difference;
        if (difference) addedNames.push(`${difference}× ${sde.value?.types.get(typeId)?.name ?? typeId}`);
    }
    for (const [typeId, count] of custom) {
        const difference = Math.max(0, count - (observed.get(typeId) ?? 0));
        removed += difference;
        if (difference) removedNames.push(`${difference}× ${sde.value?.types.get(typeId)?.name ?? typeId}`);
    }
    return { added, removed, addedNames, removedNames };
});

const observedAgo = (iso: string): string => {
    const days = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000));
    if (days === 0) return "today";
    if (days === 1) return "1d ago";
    return `${days}d ago`;
};

const suggestionExplanation = computed(() => currentFit.value?.modules.length
    ? "Ranked by observed fits sharing your current modules."
    : "Ranked by the most common observed fits for this hull.");

const familySearchQuery = computed(() => fitFamilyAdvancedSearchQuery(data.value?.window_days ?? 90));

function familySearchLink(family: FittingFamily, view: "kills" | "fits") {
    return {
        path: "/advancedsearch",
        query: {
            q: familySearchQuery.value,
            dh: family.family_hash,
            dm: "family",
            ...(view === "fits" ? { view: "fits", dedup: "exact" } : {}),
        },
    };
}
</script>

<template>
    <aside class="glass-panel flex h-full min-h-0 flex-col overflow-hidden">
        <div class="border-b border-white/[0.06] px-3 py-3">
            <div class="flex items-center gap-2">
                <Icon name="lucide:layers-3" class="h-4 w-4 text-purple-400" />
                <h2 class="text-xs font-bold uppercase tracking-[0.14em] text-purple-300">
                    Observed Fits
                </h2>
            </div>
            <div class="mt-1 flex items-center justify-between gap-2">
                <p class="min-w-0 truncate text-fine text-gray-600">{{ shipName }} · {{ editorMode }}</p>
                <button
                    v-if="selectedFamily"
                    type="button"
                    class="shrink-0 rounded px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider text-purple-300 hover:bg-purple-400/10"
                    @click="showComparison = !showComparison"
                >
                    {{ showComparison ? "Hide diff" : "Compare" }}
                </button>
            </div>
        </div>

        <div v-if="pending" class="flex flex-1 items-center justify-center">
            <Icon name="lucide:loader-2" class="h-5 w-5 animate-spin text-gray-600" />
        </div>
        <div v-else-if="!data?.families.length" class="flex flex-1 items-center justify-center px-5 text-center text-xs text-gray-600">
            No observed fitting families for this hull.
        </div>
        <div v-else class="min-h-0 flex-1 overflow-y-auto p-2">
            <button
                v-if="customFit"
                type="button"
                class="group mb-2 w-full rounded-lg border px-2.5 py-2.5 text-left transition-colors"
                :class="selectedFamily === null
                    ? 'border-blue-400/35 bg-blue-400/[0.09]'
                    : 'border-blue-400/15 bg-blue-400/[0.035] hover:border-blue-400/30 hover:bg-blue-400/[0.07]'"
                @click="restoreCustomFit"
            >
                <div class="flex items-center gap-2">
                    <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded bg-blue-400/10 text-blue-300">
                        <Icon name="lucide:pencil-ruler" class="h-3.5 w-3.5" />
                    </span>
                    <div class="min-w-0">
                        <div class="text-xs font-semibold text-blue-200">Custom Fit</div>
                        <div class="truncate text-fine text-gray-600">Your working fit · always preserved</div>
                    </div>
                </div>
            </button>
            <div v-if="customFit" class="mb-2 border-t border-white/[0.06]" />
            <div v-if="showComparison && selectedFamily" class="mb-2 rounded-lg border border-purple-400/15 bg-purple-400/[0.035] p-2">
                <div class="mb-1.5 flex items-center justify-between text-[9px] font-bold uppercase tracking-[0.1em] text-purple-300/80">
                    <span>Observed vs custom</span>
                    <span class="text-gray-600">+{{ moduleDifference.added }} / −{{ moduleDifference.removed }} modules</span>
                </div>
                <div v-for="metric in comparisonMetrics" :key="metric.label" class="grid grid-cols-[1fr_auto_auto] gap-2 py-0.5 text-fine tabular-nums">
                    <span class="text-gray-500">{{ metric.label }}</span>
                    <span class="text-gray-600">{{ metric.custom.toFixed(1) }}</span>
                    <span :class="metric.delta > 0 ? 'text-emerald-400' : metric.delta < 0 ? 'text-red-400' : 'text-gray-600'">
                        {{ metric.delta > 0 ? '+' : '' }}{{ metric.delta.toFixed(1) }} {{ metric.unit }}
                    </span>
                </div>
                <div class="grid grid-cols-[1fr_auto_auto] gap-2 py-0.5 text-fine tabular-nums">
                    <span class="text-gray-500">Price</span>
                    <span class="text-gray-600">{{ formatIsk(customPrice) }}</span>
                    <span :class="currentPrice - customPrice > 0 ? 'text-red-400' : currentPrice - customPrice < 0 ? 'text-emerald-400' : 'text-gray-600'">
                        {{ currentPrice - customPrice > 0 ? '+' : '' }}{{ formatIsk(currentPrice - customPrice) }}
                    </span>
                </div>
                <div v-if="moduleDifference.addedNames.length || moduleDifference.removedNames.length" class="mt-1.5 space-y-1 border-t border-white/[0.05] pt-1.5 text-[9px]">
                    <div v-if="moduleDifference.addedNames.length" class="truncate text-emerald-400/80" :title="moduleDifference.addedNames.join(', ')">+ {{ moduleDifference.addedNames.slice(0, 3).join(', ') }}</div>
                    <div v-if="moduleDifference.removedNames.length" class="truncate text-red-400/80" :title="moduleDifference.removedNames.join(', ')">− {{ moduleDifference.removedNames.slice(0, 3).join(', ') }}</div>
                </div>
                <div v-if="observedModified" class="mt-1.5 rounded bg-amber-400/[0.06] px-2 py-1.5 text-[9px] text-amber-300/80">
                    <div>This observed fit has been modified.</div>
                    <div class="mt-1 flex gap-1">
                        <button type="button" class="rounded bg-amber-400/10 px-1.5 py-0.5 hover:bg-amber-400/20" @click="keepAsCustomFit">Keep as custom</button>
                        <button type="button" class="rounded px-1.5 py-0.5 text-gray-500 hover:bg-white/5" @click="resetObservedFit">Reset preview</button>
                    </div>
                </div>
            </div>
            <div v-if="fitSuggestions.length" class="mb-2 rounded-lg border border-cyan-400/10 bg-cyan-400/[0.025] p-2">
                <div class="mb-1.5 flex items-center gap-1.5 text-fine font-bold uppercase tracking-[0.12em] text-cyan-300/80">
                    <Icon name="lucide:sparkles" class="h-3 w-3" />
                    Suggestions
                </div>
                <div v-for="group in fitSuggestions" :key="group.slot" class="mb-1.5 last:mb-0">
                    <div class="px-1 text-[9px] font-bold uppercase tracking-[0.1em] text-gray-600">{{ group.label }}</div>
                    <button
                        v-for="suggestion in group.suggestions"
                        :key="suggestion.typeId"
                        type="button"
                        class="group/suggestion flex w-full items-center gap-2 rounded px-1.5 py-1.5 text-left hover:bg-cyan-400/[0.07]"
                        :title="`${suggestion.name} · seen in ${suggestion.frequency}% of observed fits · ${suggestion.cpu.toFixed(0)} tf · ${suggestion.power.toFixed(0)} MW`"
                        @click="fitManager.addModule(suggestion.typeId, group.slot)"
                        @mouseenter="fitManager.addModule(suggestion.typeId, group.slot, { preview: true })"
                        @mouseleave="fitManager.removePreview()"
                    >
                        <img :src="`https://images.evetech.net/types/${suggestion.typeId}/icon?size=32`" class="h-7 w-7 shrink-0 rounded" alt="" />
                        <span class="min-w-0 flex-1">
                            <span class="block truncate text-xs font-medium text-gray-300 group-hover/suggestion:text-cyan-100">{{ suggestion.name }}</span>
                            <span class="mt-0.5 block truncate text-[9px] tabular-nums text-gray-600">
                                {{ suggestion.frequency }}% observed · {{ suggestion.cpu.toFixed(0) }} tf · {{ suggestion.power.toFixed(0) }} MW
                            </span>
                        </span>
                        <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded bg-cyan-400/[0.05] group-hover/suggestion:bg-cyan-400/10">
                            <Icon name="lucide:plus" class="h-3.5 w-3.5 text-cyan-400/80" />
                        </span>
                    </button>
                </div>
                <div class="mt-1 px-1 text-[9px] leading-3 text-gray-700">{{ suggestionExplanation }} Only options fitting open slots, hardpoints, CPU and powergrid are shown. Hover to preview.</div>
            </div>
            <div
                v-for="(family, index) in data.families"
                :key="family.family_hash"
                class="group relative mb-1.5 w-full rounded-lg border px-2.5 py-2 text-left transition-colors last:mb-0"
                :class="selectedFamily === family.family_hash
                    ? 'border-purple-400/35 bg-purple-400/[0.09]'
                    : 'border-white/[0.06] bg-white/[0.02] hover:border-purple-400/20 hover:bg-purple-400/[0.045]'"
            >
                <button
                    type="button"
                    class="absolute inset-0 z-0 rounded-lg"
                    :aria-label="`Preview ${classifyFitFamily(family.modules, family.drones, { hullGroupName: shipGroupName })} fitting family`"
                    @click="loadFamily(family, index)"
                />
                <div class="pointer-events-none relative z-10 flex items-center gap-2">
                    <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded bg-black/25 text-fine font-bold tabular-nums text-gray-600">
                        {{ String(index + 1).padStart(2, "0") }}
                    </span>
                    <div class="min-w-0 flex-1">
                        <div class="flex w-full items-baseline justify-between gap-2 text-left">
                            <span class="truncate text-xs font-semibold text-gray-200 group-hover:text-purple-100">
                                {{ classifyFitFamily(family.modules, family.drones, { hullGroupName: shipGroupName }) }}
                            </span>
                            <span class="shrink-0 text-fine text-gray-600">
                                {{ observedAgo(family.last_used) }}
                            </span>
                        </div>
                        <div class="mt-0.5 flex items-center justify-between gap-2 text-fine text-gray-600">
                            <span class="flex min-w-0 items-center gap-1">
                                <NuxtLink :to="familySearchLink(family, 'kills')" target="_blank" rel="noopener" class="pointer-events-auto rounded hover:text-blue-300 hover:underline">
                                    {{ family.total_uses.toLocaleString("en-US") }} losses
                                </NuxtLink>
                                <span>·</span>
                                <NuxtLink :to="familySearchLink(family, 'fits')" target="_blank" rel="noopener" class="pointer-events-auto rounded hover:text-blue-300 hover:underline">
                                    {{ family.variant_count }} {{ family.variant_count === 1 ? "variant" : "variants" }}
                                </NuxtLink>
                            </span>
                            <span class="shrink-0 tabular-nums text-yellow-400/75">
                                {{ formatIsk(family.fit_cost + (data?.hull_cost ?? 0)) }}
                            </span>
                        </div>
                        <div
                            v-if="fitFamilyContextParts(family.context).length"
                            class="mt-1 block w-full truncate text-left text-[9px] text-gray-600 transition-colors hover:text-purple-300/80"
                            :title="fitFamilyContextParts(family.context).join(' · ')"
                        >
                            {{ fitFamilyContextParts(family.context).join(" · ") }}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </aside>
</template>
