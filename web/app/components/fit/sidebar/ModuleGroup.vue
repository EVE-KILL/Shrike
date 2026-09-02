<script setup lang="ts">
/**
 * Recursive tree node for the HardwareListing. Each instance renders:
 *
 *   - An optional header (when `hideGroup` is false) displaying the
 *     market group's name and its CCP icon
 *   - Leaves for all `items` at this depth, sorted by meta-group
 *     (T1 → Storyline → Faction → Officer → Deadspace) then name
 *   - Recursive ModuleGroups for all nested `groups`
 *
 * Leaves wire up drag-start (with a custom setDragImage pointing at
 * the EVE image CDN), mouseenter/leave for preview overlays, and
 * double-click to commit. The drag protocol matches the React ref:
 * three custom MIME keys on e.dataTransfer.
 *
 * Self-imported to enable recursion — each market group can contain
 * more market groups any number of levels deep.
 */

import ModuleGroup from "./ModuleGroup.vue";
import TreeHeader from "./TreeHeader.vue";
import TreeLeaf from "./TreeLeaf.vue";
import TreeListing from "./TreeListing.vue";

// Slot types the sidebar knows how to drag. `Charge` and `DroneBay`
// are pseudo-slots outside our FitSlotType union — leave them as
// bare strings so the dataTransfer can carry them as-is.
export type SidebarSlotType =
    | "Low"
    | "Medium"
    | "High"
    | "Rig"
    | "SubSystem"
    | "DroneBay"
    | "Charge";

export interface ListingItem {
    name: string;
    meta: number;
    typeId: number;
    slotType: SidebarSlotType;
}

export interface ListingGroup {
    name: string;
    meta: number;
    iconID?: number;
    groups: Record<number, ListingGroup>;
    items: ListingItem[];
}

const props = withDefaults(
    defineProps<{
        level: number;
        group: ListingGroup;
        hideGroup?: boolean;
    }>(),
    { hideGroup: false },
);

const fitManager = useFitManager();

// Route "add" depending on slot type. Drone preview is a no-op
// because the engine doesn't have a transient drone state, and
// charges don't render a preview either — they'd need a per-
// compatible-module transient state the engine can't represent.
function addItem(typeId: number, slotType: SidebarSlotType, preview = false) {
    if (slotType === "Charge") {
        if (preview) return;
        // setChargeAuto walks every fitted module and loads the
        // charge into every group+size-compatible slot at once.
        // No-op if no compatible module is fitted.
        fitManager.setChargeAuto(typeId);
        return;
    }
    if (slotType === "DroneBay") {
        if (preview) return;
        fitManager.addDrone(typeId, 1, "Active");
        return;
    }
    fitManager.addModule(typeId, slotType, { preview });
}

// ------- Manual double-click counter -------
//
// We used to rely on the browser's native `dblclick` event, but the
// hardware tree recomputes whenever the fit changes (the hull-
// restriction filter feeds off currentStats) and the browser's
// click-pair state gets lost when Vue patches the DOM between two
// rapid clicks. Result: the user spam-clicks Mega Pulse II eight
// times and only ONE gets fitted because each consecutive click
// arrives against a freshly-patched element that forgets the
// pending pair.
//
// Fix: track a single "pending" click ourselves. A second click on
// the same row within DBLCLICK_WINDOW_MS fires addItem once and
// clears the pending slot; a click on a different row simply
// rebinds the slot (so switching items doesn't carry over a stale
// half-pair). The closure lives inside ModuleGroup's setup so it
// survives template re-renders — Vue doesn't tear down the
// component instance when props change.
const DBLCLICK_WINDOW_MS = 400;
let pendingClick: { typeId: number; time: number } | null = null;
function onLeafClick(item: ListingItem, event: MouseEvent) {
    if (event.shiftKey && item.slotType === "High") {
        pendingClick = null;
        fitManager.fillHighRack(item.typeId);
        return;
    }
    const now = Date.now();
    if (
        pendingClick
        && pendingClick.typeId === item.typeId
        && now - pendingClick.time <= DBLCLICK_WINDOW_MS
    ) {
        pendingClick = null;
        addItem(item.typeId, item.slotType);
    } else {
        pendingClick = { typeId: item.typeId, time: now };
    }
}

function removePreview() {
    fitManager.removePreview();
}

function onItemDragStart(e: DragEvent, item: ListingItem) {
    if (!e.dataTransfer) return;
    e.dataTransfer.effectAllowed = "copy";
    // Ghost image from the EVE type icon — gives the browser's drag
    // cursor a readable preview instead of a blurry clone of the leaf
    // row. Using `new Image()` here is deliberate: setDragImage needs
    // a fully-loaded Element, but the browser takes a screenshot of
    // the current bitmap state, so even a mid-load image is fine.
    const img = new Image();
    img.src = `https://images.evetech.net/types/${item.typeId}/icon?size=64`;
    e.dataTransfer.setDragImage(img, 32, 32);
    e.dataTransfer.setData("application/esf-type-id", String(item.typeId));
    e.dataTransfer.setData("application/esf-slot-type", item.slotType);
}

// Pre-sort the items by meta group (T1 before Faction before Officer
// before Deadspace) then by name. React does this inline in the
// render function — we compute it up front so each re-render doesn't
// re-sort.
const sortedItems = computed(() =>
    [...props.group.items].sort(
        (a, b) => a.meta - b.meta || a.name.localeCompare(b.name),
    ),
);

// Same for nested groups — sort by meta then name, return as [id, group] pairs.
const sortedGroupIds = computed(() => {
    const entries = Object.entries(props.group.groups);
    entries.sort(([, a], [, b]) => a.meta - b.meta || a.name.localeCompare(b.name));
    return entries;
});

// Icon URL for the market group header. The @eveshipfit/data package
// used to ship these as `icons/{iconID}.png` but our vendored upstream
// strips them (they're ~4 MB of raw CCP assets we don't host). Leave
// undefined for now — TreeHeader falls through and renders text only.
const headerIcon = computed(() => undefined as string | undefined);
</script>

<template>
    <!-- Pass-through node: no header, just render items + nested groups. -->
    <TreeListing v-if="props.hideGroup" :level="props.level">
        <TreeLeaf
            v-for="item in sortedItems"
            :key="item.typeId"
            :content="item.name"
            :image-url="`https://images.evetech.net/types/${item.typeId}/icon?size=32`"
            :draggable="true"
            @click="onLeafClick(item, $event)"
            @dragstart="onItemDragStart($event, item)"
            @enter="addItem(item.typeId, item.slotType, true)"
            @leave="removePreview()"
        />
        <ModuleGroup
            v-for="[id, group] in sortedGroupIds"
            :key="id"
            :level="props.level + 1"
            :group="group"
        />
    </TreeListing>

    <!-- Normal node: collapsible header + children. -->
    <TreeListing v-else :level="props.level" :has-header="true">
        <template #header>
            <TreeHeader :icon="headerIcon" :text="props.group.name" />
        </template>
        <TreeLeaf
            v-for="item in sortedItems"
            :key="item.typeId"
            :content="item.name"
            :image-url="`https://images.evetech.net/types/${item.typeId}/icon?size=32`"
            :draggable="true"
            @click="onLeafClick(item, $event)"
            @dragstart="onItemDragStart($event, item)"
            @enter="addItem(item.typeId, item.slotType, true)"
            @leave="removePreview()"
        />
        <ModuleGroup
            v-for="[id, group] in sortedGroupIds"
            :key="id"
            :level="props.level + 1"
            :group="group"
        />
    </TreeListing>
</template>
