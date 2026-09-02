<script setup lang="ts">
/**
 * Left sidebar — module/charge browser. Restyled to match the
 * EK site design: card container, lucide search icon, flat
 * filter pills, EK-style segmented tab strip for Modules/
 * Charges. Functional port from @eveshipfit/react's
 * HardwareListing preserved underneath.
 *
 * Deferred features (flagged in code):
 *   - Charges tab shows all charges in a single tree (no
 *     per-module filter by fitted weapon)
 *   - Meta group synthetic parents (Faction/Officer/Deadspace)
 */

import ShipFitIcon, { type ShipFitIconName } from "../ShipFitIcon.vue";
import ModuleGroup, {
    type ListingGroup,
    type ListingItem,
    type SidebarSlotType,
} from "./ModuleGroup.vue";
import { fuzzyFitMatch, fuzzyFitScore } from "~/utils/fuzzyFitSearch";

const { sde } = useEveData();
const { currentFit } = useCurrentFit();
const capacity = useHullCapacity();
// `currentStats` is the committed (non-preview) snapshot. Using it
// instead of `stats` means hover-preview on the sidebar doesn't
// rebuild the whole hardware tree — the restriction filter only
// recomputes when the user actually commits a fit change.
const { currentStats } = useFitStatistics();

const search = ref("");
const searchInput = ref<HTMLInputElement | null>(null);
const selectedResult = ref(0);
const fillPromptItem = ref<ListingItem | null>(null);

defineExpose({
    focusSearch: () => searchInput.value?.focus(),
    openCommandSearch: (mode: "modules" | "charges" = "modules") => {
        selection.value = mode;
        search.value = "";
        selectedResult.value = 0;
        nextTick(() => searchInput.value?.focus());
    },
});

const filter = reactive({
    lowslot: false,
    medslot: false,
    hislot: false,
    rigSubsystem: false,
    drone: false,
    // Default-on — removes a dozen irrelevant module groups (capital
    // remote reps on a frigate, etc.) and is what the user sensibly
    // expects as a starting point. They can still toggle it off via
    // the filter pill to browse the full catalog.
    hullRestriction: true,
});
const selection = ref<"modules" | "charges">("modules");

// Effect name → ID lookup for the 5 slot-type effects. Walks
// sde.dogmaEffects once when the SDE loads, cached via computed
// identity on sde.value.
const slotEffectIds = computed(() => {
    const sdeData = sde.value;
    if (!sdeData) return null;
    const byName: Record<string, number> = {};
    for (const [id, eff] of sdeData.dogmaEffects) {
        if (eff?.name) byName[eff.name] = id;
    }
    return {
        loPower: byName.loPower,
        medPower: byName.medPower,
        hiPower: byName.hiPower,
        rigSlot: byName.rigSlot,
        subSystem: byName.subSystem,
    };
});

function slotTypeFor(
    typeId: number,
    categoryID: number,
): SidebarSlotType | undefined {
    if (categoryID === 18) return "DroneBay";
    if (categoryID === 8) return "Charge";

    const sdeData = sde.value;
    const effectIds = slotEffectIds.value;
    if (!sdeData || !effectIds) return undefined;

    const typeDogma = sdeData.typeDogma.get(typeId);
    if (!typeDogma?.dogmaEffects) return undefined;

    for (const effect of typeDogma.dogmaEffects as Array<{ effectID: number }>) {
        if (effect.effectID === effectIds.loPower) return "Low";
        if (effect.effectID === effectIds.medPower) return "Medium";
        if (effect.effectID === effectIds.hiPower) return "High";
        if (effect.effectID === effectIds.rigSlot) return "Rig";
        if (effect.effectID === effectIds.subSystem) return "SubSystem";
    }
    return undefined;
}

// ========== Hull restriction filter ==========
//
// When `filter.hullRestriction` is on, we only show modules that the
// current hull can actually fit. The logic mirrors the React reference
// piece-for-piece:
//
//   - Rigs must match the hull's rigSize
//   - Drones require droneBandwidth > 0 on the hull
//   - Modules must have at least one slot of their kind on the hull
//   - Capital modules (volume ≥ 3500) require the hull to be isCapitalSize
//   - If the module declares fitsToShipType / canFitShipType1..11 or
//     canFitShipGroup01..20, at least one must match the hull's type/group
//
// All of these data points live in the SDE we already loaded — zero
// network work, zero server API.

/** Static attribute-ID cache — resolves once when the SDE loads. */
const hullRestrictionAttrs = computed(() => {
    const sdeData = sde.value;
    if (!sdeData) return null;
    const attr = (name: string) => sdeData.attributeNameToId.get(name);

    // Set of every "canFitShipType" / "fitsToShipType" attribute id.
    // Modules with a fit restriction carry one or more of these, with
    // the target type id as the value.
    const canFitTypeIds = new Set<number>();
    const canFitTypeNames = [
        "fitsToShipType",
        "canFitShipType1",
        "canFitShipType2",
        "canFitShipType3",
        "canFitShipType4",
        "canFitShipType5",
        "canFitShipType6",
        "canFitShipType7",
        "canFitShipType8",
        "canFitShipType9",
        "canFitShipType10",
        "canFitShipType11",
    ];
    for (const n of canFitTypeNames) {
        const id = attr(n);
        if (id !== undefined) canFitTypeIds.add(id);
    }

    // Set of every "canFitShipGroup" attribute id. Same idea but the
    // value is a group id instead of a type id.
    const canFitGroupIds = new Set<number>();
    for (let i = 1; i <= 20; i++) {
        const name = `canFitShipGroup${String(i).padStart(2, "0")}`;
        const id = attr(name);
        if (id !== undefined) canFitGroupIds.add(id);
    }

    return {
        rigSize: attr("rigSize"),
        droneBandwidth: attr("droneBandwidth"),
        isCapitalSize: attr("isCapitalSize"),
        hiSlots: attr("hiSlots"),
        hiSlotModifier: attr("hiSlotModifier"),
        medSlots: attr("medSlots"),
        medSlotModifier: attr("medSlotModifier"),
        lowSlots: attr("lowSlots"),
        lowSlotModifier: attr("lowSlotModifier"),
        maxSubSystems: attr("maxSubSystems"),
        canFitTypeIds,
        canFitGroupIds,
    };
});

/** Reactive snapshot of the current hull's relevant values. */
const hullRestrictionState = computed(() => {
    const sdeData = sde.value;
    const s = currentStats.value;
    const attrs = hullRestrictionAttrs.value;
    const shipTypeId = currentFit.value?.shipTypeId;

    if (!sdeData || !s || !attrs || shipTypeId === undefined) return null;

    const hullType = sdeData.types.get(shipTypeId);
    if (!hullType) return null;

    // Pull a final computed attribute off the hull, falling back to
    // base_value if no effect modified it.
    const hullAttr = (id: number | undefined): number => {
        if (id === undefined) return 0;
        const a = s.hull.attributes.get(id);
        if (!a) return 0;
        return a.value ?? a.base_value ?? 0;
    };

    // isCapitalSize isn't in the computed hull attributes on purpose —
    // it's a static type-dogma attribute. Dig it out of raw typeDogma.
    const typeDogma = sdeData.typeDogma.get(shipTypeId);
    let isCapitalSize = 0;
    if (typeDogma?.dogmaAttributes && attrs.isCapitalSize !== undefined) {
        const attr = (typeDogma.dogmaAttributes as Array<{ attributeID: number; value: number }>).find(
            (a) => a.attributeID === attrs.isCapitalSize,
        );
        if (attr) isCapitalSize = attr.value;
    }

    return {
        shipTypeId,
        groupId: hullType.groupID as number,
        rigSize: hullAttr(attrs.rigSize),
        droneBandwidth: hullAttr(attrs.droneBandwidth),
        isCapitalSize,
        slots: {
            High: Math.max(
                Math.floor(hullAttr(attrs.hiSlots) + hullAttr(attrs.hiSlotModifier)),
                capacity.slotCounts.value.High,
            ),
            Medium: Math.max(
                Math.floor(hullAttr(attrs.medSlots) + hullAttr(attrs.medSlotModifier)),
                capacity.slotCounts.value.Medium,
            ),
            Low: Math.max(
                Math.floor(hullAttr(attrs.lowSlots) + hullAttr(attrs.lowSlotModifier)),
                capacity.slotCounts.value.Low,
            ),
            SubSystem: Math.min(Math.floor(hullAttr(attrs.maxSubSystems)), 4),
        },
    };
});

/**
 * True if the given module can physically be fit onto the current hull.
 * Called per-module inside the trees() computed when the hull-restriction
 * filter is enabled.
 */
function moduleFitsHull(
    typeId: number,
    slotType: SidebarSlotType,
    module: { volume?: number },
): boolean {
    const sdeData = sde.value;
    const attrs = hullRestrictionAttrs.value;
    const state = hullRestrictionState.value;
    if (!sdeData || !attrs || !state) return true; // no hull = pass through

    // Gather the module's raw dogma attributes once.
    const typeDogma = sdeData.typeDogma.get(typeId);
    const moduleAttrs = (typeDogma?.dogmaAttributes ?? []) as Array<{
        attributeID: number;
        value: number;
    }>;
    const findAttr = (id: number | undefined): number | undefined => {
        if (id === undefined) return undefined;
        const a = moduleAttrs.find((x) => x.attributeID === id);
        return a?.value;
    };

    // Slot-type-specific checks.
    if (slotType === "Rig") {
        const moduleRigSize = findAttr(attrs.rigSize);
        if (moduleRigSize !== state.rigSize) return false;
    } else if (slotType === "DroneBay") {
        if (state.droneBandwidth === 0) return false;
    } else {
        if (slotType === "Low" && state.slots.Low === 0) return false;
        if (slotType === "Medium" && state.slots.Medium === 0) return false;
        if (slotType === "High" && state.slots.High === 0) return false;
        if (slotType === "SubSystem" && state.slots.SubSystem === 0) return false;

        // Capital-volume modules require a capital hull.
        const volume = module.volume ?? 0;
        if (volume >= 3500 && state.isCapitalSize !== 1) return false;
    }

    // canFitShipType / canFitShipGroup whitelist check. The module may
    // declare zero, one, or several. If it declares any, at least one
    // must match the hull's type id or group id respectively.
    let anyFitTypeDeclared = false;
    let anyFitGroupDeclared = false;
    let fitTypeMatch = false;
    let fitGroupMatch = false;

    for (const a of moduleAttrs) {
        if (attrs.canFitTypeIds.has(a.attributeID)) {
            anyFitTypeDeclared = true;
            if (a.value === state.shipTypeId) fitTypeMatch = true;
        } else if (attrs.canFitGroupIds.has(a.attributeID)) {
            anyFitGroupDeclared = true;
            if (a.value === state.groupId) fitGroupMatch = true;
        }
    }

    if ((anyFitTypeDeclared || anyFitGroupDeclared) && !fitTypeMatch && !fitGroupMatch) {
        return false;
    }

    return true;
}

function marketGroupChain(leafId: number): number[] {
    const sdeData = sde.value;
    if (!sdeData) return [];
    const chain: number[] = [];
    let current: number | undefined = leafId;
    while (current !== undefined && current !== null) {
        chain.push(current);
        const mg = sdeData.marketGroups.get(current);
        current = mg?.parentGroupID;
    }
    // Drop the root — every module shares "Ship Equipment" or
    // "Ammunition & Charges" and rendering it as a level just
    // wastes depth.
    chain.pop();
    return chain;
}

interface BuiltTrees {
    modules: ListingGroup;
    charges: ListingGroup;
}

const trees = computed<BuiltTrees>(() => {
    const sdeData = sde.value;
    const emptyModules: ListingGroup = {
        name: "Modules",
        meta: 0,
        groups: {},
        items: [],
    };
    const emptyCharges: ListingGroup = {
        name: "Charges",
        meta: 0,
        groups: {},
        items: [],
    };
    if (!sdeData) return { modules: emptyModules, charges: emptyCharges };

    const searchLower = search.value.toLowerCase();
    const slotFiltersActive =
        filter.lowslot || filter.medslot || filter.hislot || filter.rigSubsystem || filter.drone;

    for (const [typeId, module] of sdeData.types) {
        const cat: number = module.categoryID;
        if (cat !== 7 && cat !== 8 && cat !== 18 && cat !== 32 && cat !== 66) continue;
        if (module.marketGroupID === undefined || module.marketGroupID === null) continue;
        if (!module.published) continue;

        const slotType = slotTypeFor(typeId, cat);
        if (slotType === undefined) continue;

        if (cat !== 8 && slotFiltersActive) {
            if (slotType === "Low" && !filter.lowslot) continue;
            if (slotType === "Medium" && !filter.medslot) continue;
            if (slotType === "High" && !filter.hislot) continue;
            if ((slotType === "Rig" || slotType === "SubSystem") && !filter.rigSubsystem)
                continue;
            if (cat === 18 && !filter.drone) continue;
        }

        // Hull-restriction pass — only for modules, not charges. A
        // charge's "fittability" depends on what weapon you've loaded,
        // which we handle via the drag-drop protocol instead.
        if (filter.hullRestriction && cat !== 8) {
            if (!moduleFitsHull(typeId, slotType, module)) continue;
        }

        if (searchLower && !fuzzyFitMatch(searchLower, String(module.name))) continue;

        const chain = marketGroupChain(module.marketGroupID);
        if (cat === 18) chain.unshift(157);
        if (cat === 66) chain.unshift(477);

        const targetTree = cat === 8 ? emptyCharges : emptyModules;
        let current = targetTree;
        for (let i = chain.length - 1; i >= 0; i--) {
            const groupId = chain[i];
            const mg = sdeData.marketGroups.get(groupId!);
            if (!current.groups[groupId!]) {
                current.groups[groupId!] = {
                    name: mg?.name ?? "Unknown group",
                    meta: 1,
                    iconID: mg?.iconID,
                    groups: {},
                    items: [],
                };
            }
            current = current.groups[groupId!]!;
        }

        const item: ListingItem = {
            name: module.name,
            meta: module.metaGroupID ?? 0,
            typeId,
            slotType,
        };
        current.items.push(item);
    }

    return { modules: emptyModules, charges: emptyCharges };
});

interface FilterPill {
    icon: ShipFitIconName;
    title: string;
    selected: boolean;
    onClick: () => void;
    enabled: boolean;
}

const filterPills = computed<FilterPill[]>(() => [
    {
        icon: "fitting-lowslot",
        title: "Filter: Low Slot",
        selected: filter.lowslot,
        enabled: true,
        onClick: () => (filter.lowslot = !filter.lowslot),
    },
    {
        icon: "fitting-medslot",
        title: "Filter: Mid Slot",
        selected: filter.medslot,
        enabled: true,
        onClick: () => (filter.medslot = !filter.medslot),
    },
    {
        icon: "fitting-hislot",
        title: "Filter: High Slot",
        selected: filter.hislot,
        enabled: true,
        onClick: () => (filter.hislot = !filter.hislot),
    },
    {
        icon: "fitting-rig-subsystem",
        title: "Filter: Rig & Subsystem Slots",
        selected: filter.rigSubsystem,
        enabled: true,
        onClick: () => (filter.rigSubsystem = !filter.rigSubsystem),
    },
    {
        icon: "fitting-drones",
        title: "Filter: Drones",
        selected: filter.drone,
        enabled: true,
        onClick: () => (filter.drone = !filter.drone),
    },
    {
        icon: "fitting-hull-restriction",
        title: "Filter: Only show modules that fit the current hull",
        selected: filter.hullRestriction,
        enabled: true,
        onClick: () => (filter.hullRestriction = !filter.hullRestriction),
    },
]);

// ========== Flat search results (bypass tree when searching) ==============

const isSearching = computed(() => search.value.trim().length > 0);

const flatResults = computed<ListingItem[]>(() => {
    if (!isSearching.value) return [];
    const sdeData = sde.value;
    if (!sdeData) return [];

    const searchLower = search.value.toLowerCase();
    const slotFiltersActive =
        filter.lowslot || filter.medslot || filter.hislot || filter.rigSubsystem || filter.drone;
    const isChargesTab = selection.value === "charges";
    const results: ListingItem[] = [];

    for (const [typeId, module] of sdeData.types) {
        const cat: number = module.categoryID;
        if (cat !== 7 && cat !== 8 && cat !== 18 && cat !== 32 && cat !== 66) continue;
        if (module.marketGroupID === undefined || module.marketGroupID === null) continue;
        if (!module.published) continue;

        // Respect the Modules / Charges tab
        if (isChargesTab && cat !== 8) continue;
        if (!isChargesTab && cat === 8) continue;

        const slotType = slotTypeFor(typeId, cat);
        if (slotType === undefined) continue;

        if (cat !== 8 && slotFiltersActive) {
            if (slotType === "Low" && !filter.lowslot) continue;
            if (slotType === "Medium" && !filter.medslot) continue;
            if (slotType === "High" && !filter.hislot) continue;
            if ((slotType === "Rig" || slotType === "SubSystem") && !filter.rigSubsystem) continue;
            if (cat === 18 && !filter.drone) continue;
        }

        if (filter.hullRestriction && cat !== 8) {
            if (!moduleFitsHull(typeId, slotType, module)) continue;
        }

        if (!fuzzyFitMatch(searchLower, String(module.name))) continue;

        results.push({
            name: module.name,
            meta: module.metaGroupID ?? 0,
            typeId,
            slotType,
        });

    }

    results.sort((a, b) =>
        fuzzyFitScore(search.value, a.name) - fuzzyFitScore(search.value, b.name)
        || a.meta - b.meta
        || a.name.localeCompare(b.name),
    );
    return results.slice(0, 15);
});

// Interaction handlers for flat search results (same logic as ModuleGroup)
const fitManager = useFitManager();

const DBLCLICK_WINDOW_MS = 400;
let pendingSearchClick: { typeId: number; time: number } | null = null;

function onSearchResultClick(item: ListingItem, event: MouseEvent) {
    if (event.shiftKey && item.slotType === "High") {
        pendingSearchClick = null;
        fillPromptItem.value = null;
        fitManager.fillHighRack(item.typeId);
        return;
    }
    const now = Date.now();
    if (
        pendingSearchClick
        && pendingSearchClick.typeId === item.typeId
        && now - pendingSearchClick.time <= DBLCLICK_WINDOW_MS
    ) {
        pendingSearchClick = null;
        if (item.slotType === "Charge") {
            fitManager.setChargeAuto(item.typeId);
        } else if (item.slotType === "DroneBay") {
            fitManager.addDrone(item.typeId);
        } else {
            fitManager.addModule(item.typeId, item.slotType as any);
        }
    } else {
        pendingSearchClick = { typeId: item.typeId, time: now };
    }
}

function addSearchResult(item: ListingItem) {
    if (item.slotType === "Charge") fitManager.setChargeAuto(item.typeId);
    else if (item.slotType === "DroneBay") fitManager.addDrone(item.typeId);
    else {
        fitManager.addModule(item.typeId, item.slotType as any);
        if (item.slotType === "High") fillPromptItem.value = item;
    }
}

function fillRemainingHighSlots() {
    const item = fillPromptItem.value;
    if (!item) return;
    fitManager.fillHighRack(item.typeId);
    fillPromptItem.value = null;
}

watch(flatResults, () => { selectedResult.value = 0; });

function onSearchKeydown(event: KeyboardEvent) {
    if (event.key === "ArrowDown") {
        event.preventDefault();
        selectedResult.value = Math.min(selectedResult.value + 1, flatResults.value.length - 1);
    } else if (event.key === "ArrowUp") {
        event.preventDefault();
        selectedResult.value = Math.max(selectedResult.value - 1, 0);
    } else if (event.key === "Enter") {
        event.preventDefault();
        const item = flatResults.value[selectedResult.value];
        if (item && event.shiftKey && item.slotType === "High") {
            fillPromptItem.value = null;
            fitManager.fillHighRack(item.typeId);
        } else if (item) addSearchResult(item);
    } else if (event.key === "Escape" && fillPromptItem.value) {
        event.stopPropagation();
        fillPromptItem.value = null;
    }
}

function onSearchResultDragStart(e: DragEvent, item: ListingItem) {
    if (!e.dataTransfer) return;
    e.dataTransfer.effectAllowed = "copy";
    const img = new Image();
    img.src = `https://images.evetech.net/types/${item.typeId}/icon?size=64`;
    e.dataTransfer.setDragImage(img, 32, 32);
    e.dataTransfer.setData("application/esf-type-id", String(item.typeId));
    e.dataTransfer.setData("application/esf-slot-type", item.slotType);
}
</script>

<template>
    <div class="glass-panel flex flex-col h-full overflow-hidden">
        <!-- Search -->
        <div class="flex items-center gap-2 px-3 py-2 border-b border-white/[0.08]">
            <Icon name="lucide:search" class="w-4 h-4 text-gray-600 flex-shrink-0" />
            <input
                ref="searchInput"
                v-model="search"
                type="text"
                placeholder="Search modules & charges…"
                class="min-w-0 flex-1 bg-transparent text-sm text-gray-200 placeholder-gray-600 outline-none"
                @keydown="onSearchKeydown"
            />
            <button
                v-if="search"
                type="button"
                class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded text-gray-600 transition-colors hover:bg-white/[0.06] hover:text-gray-300"
                aria-label="Clear hardware search"
                @mousedown.prevent
                @click="search = ''; selectedResult = 0; fillPromptItem = null; searchInput?.focus()"
            >
                <Icon name="lucide:x" class="h-3.5 w-3.5" />
            </button>
        </div>

        <div v-if="fillPromptItem" class="border-b border-blue-400/15 bg-blue-400/[0.06] px-3 py-2">
            <div class="text-xs font-medium text-blue-200">Fill available high slots?</div>
            <div class="mt-0.5 truncate text-fine text-gray-500">{{ fillPromptItem.name }}</div>
            <div class="mt-2 flex gap-2">
                <button type="button" class="rounded bg-blue-500/20 px-2 py-1 text-fine font-semibold text-blue-200 hover:bg-blue-500/30" @click="fillRemainingHighSlots">Fill rack</button>
                <button type="button" class="rounded px-2 py-1 text-fine text-gray-500 hover:bg-white/[0.05] hover:text-gray-300" @click="fillPromptItem = null">Just one</button>
            </div>
        </div>

        <!-- Slot filters (shown only on Modules tab) -->
        <div
            v-if="selection === 'modules'"
            class="flex gap-1 px-2 py-1.5 border-b border-white/[0.04]"
        >
            <button
                v-for="(pill, i) in filterPills"
                :key="i"
                type="button"
                v-tooltip="pill.title"
                :disabled="!pill.enabled"
                :class="[
                    'flex items-center justify-center w-8 h-8 rounded transition-colors',
                    pill.selected
                        ? 'bg-blue-500/[0.15] ring-1 ring-blue-400/30'
                        : pill.enabled
                            ? 'hover:bg-white/[0.04]'
                            : 'opacity-30 cursor-not-allowed',
                ]"
                @click="pill.enabled && pill.onClick()"
            >
                <ShipFitIcon :name="pill.icon" :size="20" />
            </button>
        </div>

        <!-- Modules / Charges tab strip -->
        <div class="flex gap-1 p-1 border-b border-white/[0.04] bg-white/[0.02]">
            <button
                type="button"
                :class="[
                    'flex-1 px-3 py-1.5 text-[11px] font-bold uppercase tracking-[0.12em] rounded transition-colors',
                    selection === 'modules'
                        ? 'bg-blue-500/[0.15] text-blue-400'
                        : 'text-gray-500 hover:text-gray-300',
                ]"
                @click="selection = 'modules'"
            >
                Modules
            </button>
            <button
                type="button"
                :class="[
                    'flex-1 px-3 py-1.5 text-[11px] font-bold uppercase tracking-[0.12em] rounded transition-colors',
                    selection === 'charges'
                        ? 'bg-blue-500/[0.15] text-blue-400'
                        : 'text-gray-500 hover:text-gray-300',
                ]"
                @click="selection = 'charges'"
            >
                Charges
            </button>
        </div>

        <!-- Flat search results (when searching) or tree browser (when not) -->
        <div class="flex-1 min-h-0 overflow-y-auto py-1">
            <template v-if="isSearching">
                <div
                    v-for="(item, index) in flatResults"
                    :key="item.typeId"
                    class="flex items-center gap-1.5 px-2 py-1 text-xs cursor-grab active:cursor-grabbing select-none"
                    :class="index === selectedResult ? 'bg-blue-400/10 text-blue-200' : 'text-gray-300 hover:bg-white/[0.04]'"
                    draggable="true"
                    @mouseenter="selectedResult = index"
                    @click="onSearchResultClick(item, $event)"
                    @dragstart="onSearchResultDragStart($event, item)"
                >
                    <img
                        :src="`https://images.evetech.net/types/${item.typeId}/icon?size=32`"
                        class="w-5 h-5 rounded-sm flex-none"
                        loading="lazy"
                    />
                    <span class="flex-1 whitespace-nowrap overflow-hidden text-ellipsis">{{ item.name }}</span>
                </div>
                <div
                    v-if="flatResults.length === 0"
                    class="px-3 py-4 text-center text-xs text-gray-600"
                >
                    No matches
                </div>
            </template>
            <template v-else>
                <ModuleGroup
                    v-if="selection === 'modules'"
                    :level="0"
                    :group="trees.modules"
                    :hide-group="true"
                />
                <ModuleGroup
                    v-else
                    :level="0"
                    :group="trees.charges"
                    :hide-group="true"
                />
            </template>
        </div>
    </div>
</template>
