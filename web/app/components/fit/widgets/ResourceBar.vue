<script setup lang="ts">
/**
 * Full-width resource readout. Sits between the (sidebar + ring) row
 * and the expandable bay drawers below.
 *
 * Five cells:
 *   CPU        — static readout  (hull.cpuOutput / cpuFree)
 *   Power      — static readout  (hull.powerOutput / powerFree)
 *   Drone Bay  — TOGGLE — click to expand the DroneBay widget below,
 *                and a drop target for sidebar drone drags even when
 *                collapsed
 *   Bandwidth  — static readout  (droneBandwidthLoad / droneBandwidth)
 *   Cargo      — TOGGLE — same expand-on-click + drop-target story as
 *                Drone Bay, but accepts any typed item
 *
 * Each cell has a used/total number and a fill bar that tints amber
 * at >90% and red at >100%. The toggle cells show a chevron on the
 * right that flips up when their drawer is open. The drop-target
 * affordance is a blue ring that lights up during a valid drag.
 *
 * State: `expanded` is a v-model prop managed by the parent page so
 * multiple widgets can coordinate (e.g. the expanded drawer renders
 * separately from this bar). Values are the BayKey literal union,
 * imported from ./bayKey.ts because <script setup> doesn't support
 * exporting types from inside.
 */

import type { BayKey } from "./bayKey";

const props = defineProps<{
    expanded: BayKey;
}>();

const emit = defineEmits<{
    (e: "update:expanded", value: BayKey): void;
}>();

const { currentFit, setFit } = useCurrentFit();
const capacity = useHullCapacity();
const {
    cpu,
    power,
    droneBayCapacity,
    droneBayUsed,
    droneBandwidth,
    droneBandwidthUsed,
    cargoCapacity,
    cargoUsed,
} = capacity;
const fitManager = useFitManager();

const droneDropActive = ref(false);
const cargoDropActive = ref(false);
// Transient rejection flash — toggles on when a drop is refused for
// capacity reasons, then auto-clears after ~1.2s. Used to paint the
// offending cell red so the user knows why nothing happened.
const droneDropRejected = ref(false);
const cargoDropRejected = ref(false);
let droneRejectTimer: ReturnType<typeof setTimeout> | null = null;
let cargoRejectTimer: ReturnType<typeof setTimeout> | null = null;

function flashDroneReject() {
    droneDropRejected.value = true;
    if (droneRejectTimer) clearTimeout(droneRejectTimer);
    droneRejectTimer = setTimeout(() => {
        droneDropRejected.value = false;
    }, 1200);
}

function flashCargoReject() {
    cargoDropRejected.value = true;
    if (cargoRejectTimer) clearTimeout(cargoRejectTimer);
    cargoRejectTimer = setTimeout(() => {
        cargoDropRejected.value = false;
    }, 1200);
}

function toggle(bay: Exclude<BayKey, null>) {
    emit("update:expanded", props.expanded === bay ? null : bay);
}

function parseNum(s: string | null | undefined): number | null {
    if (s === null || s === undefined || s === "") return null;
    const n = Number(s);
    return Number.isFinite(n) ? n : null;
}

/** Drone-bay drop handler — only accepts sidebar items flagged as DroneBay. */
function onDroneDragOver(e: DragEvent) {
    if (!e.dataTransfer) return;
    // We can't read the payload values during dragover (browsers only
    // expose .types), but the slot-type is on the MIME key list too.
    e.dataTransfer.dropEffect = "copy";
    e.preventDefault();
    droneDropActive.value = true;
}

function onDroneDragLeave() {
    droneDropActive.value = false;
}

function onDroneDrop(e: DragEvent) {
    e.preventDefault();
    droneDropActive.value = false;
    if (!e.dataTransfer) return;
    const slotType = e.dataTransfer.getData("application/esf-slot-type");
    const typeId = parseNum(e.dataTransfer.getData("application/esf-type-id"));
    if (slotType !== "DroneBay" || typeId === null) return;
    // Volume check up front so we can flash the cell red on refusal;
    // useFitManager.addDrone would silently no-op otherwise.
    if (!capacity.canAddDrone(typeId, 1)) {
        flashDroneReject();
        return;
    }
    fitManager.addDrone(typeId, 1, "Active");
}

/** Cargo drop handler — accepts any typed item from the sidebar. */
function onCargoDragOver(e: DragEvent) {
    if (!e.dataTransfer) return;
    e.dataTransfer.dropEffect = "copy";
    e.preventDefault();
    cargoDropActive.value = true;
}

function onCargoDragLeave() {
    cargoDropActive.value = false;
}

function onCargoDrop(e: DragEvent) {
    e.preventDefault();
    cargoDropActive.value = false;
    if (!e.dataTransfer) return;
    const typeId = parseNum(e.dataTransfer.getData("application/esf-type-id"));
    if (typeId === null) return;
    // Volume check up front — flash the cell red if the item doesn't
    // fit. Caller gets immediate visual feedback instead of a silent
    // no-op that looks like the drag missed the target.
    if (!capacity.canAddCargo(typeId, 1)) {
        flashCargoReject();
        return;
    }
    addCargoItem(typeId, 1);
}

/**
 * Inline cargo add — merges by type id so dropping the same type
 * twice increments rather than duplicating the row. Uses setFit so
 * the currentFitId is preserved (a saved fit stays saved after this
 * edit). Lives here instead of useFitManager because cargo doesn't
 * yet have a dedicated mutator surface; promote when we add a Cargo
 * tab to the sidebar.
 */
function addCargoItem(typeId: number, quantity: number) {
    const fit = currentFit.value;
    if (!fit) return;
    const idx = fit.cargo.findIndex((c) => c.typeId === typeId);
    const nextCargo = [...fit.cargo];
    if (idx >= 0) {
        nextCargo[idx] = { ...nextCargo[idx]!, quantity: nextCargo[idx]!.quantity + quantity };
    } else {
        nextCargo.push({ typeId, quantity });
    }
    setFit({ ...fit, cargo: nextCargo });
}

function pct(used: number, total: number): number {
    if (total <= 0) return 0;
    return Math.max(0, (used / total) * 100);
}

function fmt(n: number, fixed: number): string {
    if (!Number.isFinite(n)) return "0";
    return n.toLocaleString("en-US", {
        minimumFractionDigits: fixed,
        maximumFractionDigits: fixed,
    });
}

function tone(pctVal: number) {
    if (pctVal > 100) return { text: "text-red-400", bar: "bg-red-500/70" };
    if (pctVal > 90) return { text: "text-amber-300", bar: "bg-amber-400/70" };
    return { text: "text-gray-200", bar: "bg-blue-500/60" };
}
</script>

<template>
    <div
        v-if="currentFit"
        class="glass-panel grid grid-cols-5 gap-2 p-2"
    >
        <!-- CPU — static readout -->
        <div class="flex flex-col gap-1 px-3 py-1.5 rounded-md bg-white/[0.02] border border-white/[0.04]">
            <div class="text-fine font-bold uppercase tracking-[0.12em] text-gray-500">CPU</div>
            <div class="text-xs tabular-nums flex items-baseline gap-1">
                <span :class="tone(pct(cpu.used, cpu.total)).text">{{ fmt(cpu.used, 1) }}</span>
                <span class="text-gray-600">/</span>
                <span class="text-gray-400">{{ fmt(cpu.total, 1) }}</span>
                <span class="text-fine text-gray-600 ml-0.5">tf</span>
            </div>
            <div class="h-1 bg-black/40 rounded-full overflow-hidden">
                <div
                    class="h-full rounded-full transition-all"
                    :class="tone(pct(cpu.used, cpu.total)).bar"
                    :style="{ width: `${Math.min(100, pct(cpu.used, cpu.total))}%` }"
                ></div>
            </div>
        </div>

        <!-- Power Grid — static readout -->
        <div class="flex flex-col gap-1 px-3 py-1.5 rounded-md bg-white/[0.02] border border-white/[0.04]">
            <div class="text-fine font-bold uppercase tracking-[0.12em] text-gray-500">Power</div>
            <div class="text-xs tabular-nums flex items-baseline gap-1">
                <span :class="tone(pct(power.used, power.total)).text">{{ fmt(power.used, 1) }}</span>
                <span class="text-gray-600">/</span>
                <span class="text-gray-400">{{ fmt(power.total, 1) }}</span>
                <span class="text-fine text-gray-600 ml-0.5">MW</span>
            </div>
            <div class="h-1 bg-black/40 rounded-full overflow-hidden">
                <div
                    class="h-full rounded-full transition-all"
                    :class="tone(pct(power.used, power.total)).bar"
                    :style="{ width: `${Math.min(100, pct(power.used, power.total))}%` }"
                ></div>
            </div>
        </div>

        <!-- Drone Bay — toggle + drop target -->
        <button
            type="button"
            class="flex flex-col gap-1 px-3 py-1.5 rounded-md bg-white/[0.02] border transition-colors text-left"
            :class="[
                droneDropRejected
                    ? 'bg-red-500/[0.12] border-red-500/50 ring-1 ring-red-500/40'
                    : expanded === 'drones'
                        ? 'bg-blue-500/[0.08] border-blue-500/30'
                        : droneDropActive
                            ? 'bg-blue-500/[0.12] border-blue-400/60 ring-1 ring-blue-400/40'
                            : 'border-white/[0.04] hover:bg-white/[0.04] hover:border-white/[0.12]',
            ]"
            @click="toggle('drones')"
            @dragover="onDroneDragOver"
            @dragleave="onDroneDragLeave"
            @drop="onDroneDrop"
        >
            <div class="flex items-center justify-between">
                <span class="text-fine font-bold uppercase tracking-[0.12em] text-gray-500 flex items-center gap-1">
                    <Icon name="lucide:radar" class="w-3 h-3" />
                    Drone Bay
                </span>
                <Icon
                    :name="expanded === 'drones' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                    class="w-3 h-3 text-gray-600"
                />
            </div>
            <div class="text-xs tabular-nums flex items-baseline gap-1">
                <span :class="tone(pct(droneBayUsed, droneBayCapacity)).text">
                    {{ fmt(droneBayUsed, 1) }}
                </span>
                <span class="text-gray-600">/</span>
                <span class="text-gray-400">{{ fmt(droneBayCapacity, 0) }}</span>
                <span class="text-fine text-gray-600 ml-0.5">m³</span>
            </div>
            <div class="h-1 bg-black/40 rounded-full overflow-hidden">
                <div
                    class="h-full rounded-full transition-all"
                    :class="tone(pct(droneBayUsed, droneBayCapacity)).bar"
                    :style="{ width: `${Math.min(100, pct(droneBayUsed, droneBayCapacity))}%` }"
                ></div>
            </div>
        </button>

        <!-- Bandwidth — static readout -->
        <div class="flex flex-col gap-1 px-3 py-1.5 rounded-md bg-white/[0.02] border border-white/[0.04]">
            <div class="text-fine font-bold uppercase tracking-[0.12em] text-gray-500">Bandwidth</div>
            <div class="text-xs tabular-nums flex items-baseline gap-1">
                <span :class="tone(pct(droneBandwidthUsed, droneBandwidth)).text">
                    {{ fmt(droneBandwidthUsed, 0) }}
                </span>
                <span class="text-gray-600">/</span>
                <span class="text-gray-400">{{ fmt(droneBandwidth, 0) }}</span>
                <span class="text-fine text-gray-600 ml-0.5">Mbit</span>
            </div>
            <div class="h-1 bg-black/40 rounded-full overflow-hidden">
                <div
                    class="h-full rounded-full transition-all"
                    :class="tone(pct(droneBandwidthUsed, droneBandwidth)).bar"
                    :style="{ width: `${Math.min(100, pct(droneBandwidthUsed, droneBandwidth))}%` }"
                ></div>
            </div>
        </div>

        <!-- Cargo — toggle + drop target -->
        <button
            type="button"
            class="flex flex-col gap-1 px-3 py-1.5 rounded-md bg-white/[0.02] border transition-colors text-left"
            :class="[
                cargoDropRejected
                    ? 'bg-red-500/[0.12] border-red-500/50 ring-1 ring-red-500/40'
                    : expanded === 'cargo'
                        ? 'bg-blue-500/[0.08] border-blue-500/30'
                        : cargoDropActive
                            ? 'bg-blue-500/[0.12] border-blue-400/60 ring-1 ring-blue-400/40'
                            : 'border-white/[0.04] hover:bg-white/[0.04] hover:border-white/[0.12]',
            ]"
            @click="toggle('cargo')"
            @dragover="onCargoDragOver"
            @dragleave="onCargoDragLeave"
            @drop="onCargoDrop"
        >
            <div class="flex items-center justify-between">
                <span class="text-fine font-bold uppercase tracking-[0.12em] text-gray-500 flex items-center gap-1">
                    <Icon name="lucide:package" class="w-3 h-3" />
                    Cargo
                </span>
                <Icon
                    :name="expanded === 'cargo' ? 'lucide:chevron-up' : 'lucide:chevron-down'"
                    class="w-3 h-3 text-gray-600"
                />
            </div>
            <div class="text-xs tabular-nums flex items-baseline gap-1">
                <span :class="tone(pct(cargoUsed, cargoCapacity)).text">
                    {{ fmt(cargoUsed, 1) }}
                </span>
                <span class="text-gray-600">/</span>
                <span class="text-gray-400">{{ fmt(cargoCapacity, 0) }}</span>
                <span class="text-fine text-gray-600 ml-0.5">m³</span>
            </div>
            <div class="h-1 bg-black/40 rounded-full overflow-hidden">
                <div
                    class="h-full rounded-full transition-all"
                    :class="tone(pct(cargoUsed, cargoCapacity)).bar"
                    :style="{ width: `${Math.min(100, pct(cargoUsed, cargoCapacity))}%` }"
                ></div>
            </div>
        </button>
    </div>
</template>
