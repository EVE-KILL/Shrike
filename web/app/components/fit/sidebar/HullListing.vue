<script setup lang="ts">
/**
 * Left sidebar — ship browser. Restyled to match the EK site
 * design: card container, lucide search icon, flat filter pills,
 * EK-style collapsible tree. Functional port from
 * @eveshipfit/react's HullListing preserved underneath.
 *
 * Walks the SDE for all published ships (categoryID 6), buckets
 * by group name → race → hull. Click the "+" action on a hull
 * row to create a new fit on it.
 *
 * Local / character / corp / alliance fit filters are visually
 * present but disabled — we don't have the LocalFits / ESI
 * character providers yet. The "current hull" toggle + search
 * are wired up.
 */

import ShipFitIcon, { type ShipFitIconName } from "../ShipFitIcon.vue";
import TreeHeader from "./TreeHeader.vue";
import TreeHeaderAction from "./TreeHeaderAction.vue";
import TreeListing from "./TreeListing.vue";

const { sde } = useEveData();
const { currentFit } = useCurrentFit();
const fitManager = useFitManager();

const factionIdToRace: Record<number, string> = {
    500001: "Caldari",
    500002: "Minmatar",
    500003: "Amarr",
    500004: "Gallente",
    1: "Non-Empire",
};

const RACE_ORDER: Array<{ id: number; name: string }> = [
    { id: 500003, name: "Amarr" },
    { id: 500001, name: "Caldari" },
    { id: 500004, name: "Gallente" },
    { id: 500002, name: "Minmatar" },
    { id: 1, name: "Non-Empire" },
];

const search = ref("");
const searchInput = ref<HTMLInputElement | null>(null);

defineExpose({
    focusSearch: () => searchInput.value?.focus(),
});

const filter = reactive({
    localFits: false,
    characterFits: false,
    currentHull: false,
});

interface HullEntry {
    typeId: number;
    name: string;
}
interface RaceEntry {
    raceId: number;
    raceName: string;
    hulls: HullEntry[];
}
interface GroupEntry {
    groupName: string;
    races: RaceEntry[];
}

const hullGroups = computed<GroupEntry[]>(() => {
    const sdeData = sde.value;
    if (!sdeData) return [];

    const grouped: Record<string, Record<string, HullEntry[]>> = {};
    const searchLower = search.value.toLowerCase();
    const currentShipId = currentFit.value?.shipTypeId;

    for (const [typeId, hull] of sdeData.types) {
        if (hull.categoryID !== 6) continue;
        if (hull.marketGroupID === undefined || hull.marketGroupID === null) continue;
        if (!hull.published) continue;
        if (filter.currentHull && currentShipId !== typeId) continue;
        if (searchLower && !hull.name.toLowerCase().includes(searchLower)) continue;

        const groupName = sdeData.groups.get(hull.groupID)?.name ?? "Unknown Group";
        const raceName = factionIdToRace[hull.factionID ?? 0] ?? "Non-Empire";

        (grouped[groupName] ??= {})[raceName] ??= [];
        grouped[groupName][raceName].push({ typeId, name: hull.name });
    }

    const result: GroupEntry[] = [];
    for (const groupName of Object.keys(grouped).sort()) {
        const races: RaceEntry[] = [];
        for (const race of RACE_ORDER) {
            const hulls = grouped[groupName][race.name];
            if (!hulls || hulls.length === 0) continue;
            hulls.sort((a, b) => a.name.localeCompare(b.name));
            races.push({ raceId: race.id, raceName: race.name, hulls });
        }
        if (races.length > 0) {
            result.push({ groupName, races });
        }
    }
    return result;
});

function onSimulate(typeId: number) {
    fitManager.createNewFit(typeId);
}

/**
 * Double-clicking a hull row is the "soft apply" path — it only
 * loads the hull if the editor is currently empty (no fit at all,
 * or an empty placeholder with shipTypeId 0). For a populated fit
 * the user has to use the explicit `+` action button, which makes
 * sense: swapping the hull nukes the whole fit and shouldn't
 * happen from an easy-to-trigger gesture.
 */
function onHullDblclick(typeId: number) {
    const fit = currentFit.value;
    if (fit && fit.shipTypeId > 0) return;
    fitManager.createNewFit(typeId);
}

interface FilterPill {
    icon: ShipFitIconName;
    title: string;
    enabled: boolean;
    selected: boolean;
    onClick: () => void;
}

const filterPills = computed<FilterPill[]>(() => [
    {
        icon: "fitting-local",
        title: "Filter: Browser-stored fittings (not wired up yet)",
        enabled: false,
        selected: filter.localFits,
        onClick: () => (filter.localFits = !filter.localFits),
    },
    {
        icon: "fitting-character",
        title: "Filter: In-game personal fittings (not wired up yet)",
        enabled: false,
        selected: filter.characterFits,
        onClick: () => (filter.characterFits = !filter.characterFits),
    },
    {
        icon: "fitting-corporation",
        title: "CCP didn't implement this ESI endpoint",
        enabled: false,
        selected: false,
        onClick: () => {},
    },
    {
        icon: "fitting-alliance",
        title: "CCP didn't implement this ESI endpoint",
        enabled: false,
        selected: false,
        onClick: () => {},
    },
    {
        icon: "fitting-hull",
        title: "Filter: Current hull",
        enabled: true,
        selected: filter.currentHull,
        onClick: () => (filter.currentHull = !filter.currentHull),
    },
]);
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
                placeholder="Search hulls…"
                class="flex-1 bg-transparent text-sm text-gray-200 placeholder-gray-600 outline-none"
            />
        </div>

        <!-- Filter pills -->
        <div class="flex gap-1 px-2 py-1.5 border-b border-white/[0.04]">
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

        <!-- Tree -->
        <div class="flex-1 overflow-y-auto py-1">
            <TreeListing
                v-for="group in hullGroups"
                :key="group.groupName"
                :level="1"
                :has-header="true"
            >
                <template #header>
                    <TreeHeader :text="group.groupName" />
                </template>
                <TreeListing
                    v-for="race in group.races"
                    :key="race.raceName"
                    :level="2"
                    :has-header="true"
                >
                    <template #header>
                        <TreeHeader
                            :icon="`https://images.evetech.net/corporations/${race.raceId}/logo?size=32`"
                            :text="`${race.raceName} · ${race.hulls.length}`"
                        />
                    </template>
                    <!-- Hull row — direct clickable div instead of a
                         TreeListing wrapper so we don't have an empty
                         "saved fits" subtree to expand. Saved fits are
                         handled by /fits on the main site, not inside
                         the builder sidebar. Double-click on the row
                         soft-applies (only if the editor is empty);
                         the explicit + button force-replaces the
                         current fit. -->
                    <div
                        v-for="hull in race.hulls"
                        :key="hull.typeId"
                        class="flex items-center gap-1.5 pl-5 pr-2 h-7 text-xs text-gray-400 cursor-pointer hover:bg-white/[0.04] transition-colors select-none"
                        @dblclick="onHullDblclick(hull.typeId)"
                    >
                        <img
                            :src="`https://images.evetech.net/types/${hull.typeId}/icon?size=32`"
                            width="24"
                            height="24"
                            alt=""
                            class="flex-shrink-0"
                        />
                        <span class="flex-1 whitespace-nowrap overflow-hidden text-ellipsis">{{ hull.name }}</span>
                        <TreeHeaderAction
                            icon="lucide:plus"
                            title="Create new fit"
                            @click="onSimulate(hull.typeId)"
                        />
                    </div>
                </TreeListing>
            </TreeListing>
        </div>
    </div>
</template>
