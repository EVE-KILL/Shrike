<script setup lang="ts">
/**
 * Table/list view of the current fit — alternative to the ring view.
 * Shows modules grouped by slot type with per-module stats: DPS,
 * volley, CPU, powergrid, optimal range, and loaded charge.
 *
 * Mirrors the layout popularised by Pyfa's module list.
 */

import FitContextMenu from "./FitContextMenu.vue";
import type { FitSlotType } from "~/composables/fit/types";

defineProps<{
    constrained?: boolean;
}>();

const tableRef = ref<HTMLTableElement | null>(null);
const columnWidths = ref<Array<number | null>>([null, 55, 58, 46, 46, 60, 120]);
const selectedRow = ref<{ slotType: string; slotIndex: number } | null>(null);
const dropTarget = ref<{ slotType: string; slotIndex: number } | null>(null);
const COLUMN_PREFS_KEY = "ek-fit-table-columns-v1";
const columnMinimums = [140, 42, 48, 40, 40, 54, 80];

function startColumnResize(event: PointerEvent, index: number) {
    if (!tableRef.value || index >= columnWidths.value.length - 1) return;
    event.preventDefault();

    const headers = [...tableRef.value.querySelectorAll("thead th")];
    const current = headers[index]?.getBoundingClientRect().width ?? 0;
    const next = headers[index + 1]?.getBoundingClientRect().width ?? 0;
    const startX = event.clientX;

    // Freeze the live defaults only once the user begins dragging. Until
    // then the table retains its existing responsive widths.
    columnWidths.value = headers.map((header) => header.getBoundingClientRect().width);
    const previousCursor = document.body.style.cursor;
    const previousSelection = document.body.style.userSelect;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";

    const onMove = (move: PointerEvent) => {
        const delta = move.clientX - startX;
        const bounded = Math.max(
            columnMinimums[index]! - current,
            Math.min(delta, next - columnMinimums[index + 1]!),
        );
        const widths = [...columnWidths.value];
        widths[index] = current + bounded;
        widths[index + 1] = next - bounded;
        columnWidths.value = widths;
    };
    const onUp = () => {
        document.removeEventListener("pointermove", onMove);
        document.removeEventListener("pointerup", onUp);
        document.body.style.cursor = previousCursor;
        document.body.style.userSelect = previousSelection;
    };
    document.addEventListener("pointermove", onMove);
    document.addEventListener("pointerup", onUp, { once: true });
}

const { sde } = useEveData();
const { stats } = useFitStatistics();
const { currentFit } = useCurrentFit();
const capacity = useHullCapacity();
// Note: capacity is also used by droneActivateAll for maxActivatableDrones

// Context menu state
const ctxMenu = reactive({
    visible: false,
    x: 0,
    y: 0,
    typeId: 0,
    slotType: "High" as FitSlotType,
    slotIndex: 0,
    currentState: "Active",
    hasCharge: false,
    chargeTypeId: null as number | null,
    canFillRack: false,
    empty: false,
});

function openContextMenu(e: MouseEvent, row: ModuleRow, slotType: string) {
    e.preventDefault();
    droneCtx.visible = false;
    ctxMenu.visible = true;
    ctxMenu.x = e.clientX;
    ctxMenu.y = e.clientY;
    ctxMenu.typeId = row.typeId;
    ctxMenu.slotType = slotType as FitSlotType;
    ctxMenu.slotIndex = row.slotIndex;
    ctxMenu.currentState = row.state;
    ctxMenu.hasCharge = row.chargeName !== null;
    ctxMenu.chargeTypeId = row.chargeTypeId;
    ctxMenu.empty = row.typeId === 0;
    ctxMenu.canFillRack = slotType === "High"
        && (capacity.needsLauncherHardpoint(row.typeId) || capacity.needsTurretHardpoint(row.typeId))
        && capacity.slotsUsed.value.High < capacity.slotCounts.value.High
        && capacity.hasHardpointFor(row.typeId);
}

function closeContextMenu() {
    ctxMenu.visible = false;
}

// Drone context menu
const droneCtx = reactive({
    visible: false,
    x: 0,
    y: 0,
    typeId: 0,
    name: "",
    active: 0,
    passive: 0,
});

function openDroneContextMenu(e: MouseEvent, drone: { typeId: number; name: string; active: number; passive: number }) {
    e.preventDefault();
    ctxMenu.visible = false;
    droneCtx.visible = true;
    droneCtx.x = e.clientX;
    droneCtx.y = e.clientY;
    droneCtx.typeId = drone.typeId;
    droneCtx.name = drone.name;
    droneCtx.active = drone.active;
    droneCtx.passive = drone.passive;
}

function closeDroneContextMenu() {
    droneCtx.visible = false;
}

const droneMenuEl = ref<HTMLElement | null>(null);

function onGlobalMousedown(e: MouseEvent) {
    if (droneCtx.visible && droneMenuEl.value && !droneMenuEl.value.contains(e.target as Node)) {
        droneCtx.visible = false;
    }
}

onMounted(() => {
    document.addEventListener("mousedown", onGlobalMousedown);
    try {
        const saved = JSON.parse(localStorage.getItem(COLUMN_PREFS_KEY) ?? "null");
        if (Array.isArray(saved) && saved.length === columnWidths.value.length) columnWidths.value = saved;
    } catch { /* Ignore stale preferences. */ }
});
onUnmounted(() => document.removeEventListener("mousedown", onGlobalMousedown));
watch(columnWidths, widths => {
    if (!import.meta.client) return;
    try { localStorage.setItem(COLUMN_PREFS_KEY, JSON.stringify(widths)); } catch { /* Storage unavailable. */ }
}, { deep: true });

const fitManager = useFitManager();

function draggedNumber(event: DragEvent, key: string): number | undefined {
    const raw = event.dataTransfer?.getData(key) ?? "";
    const value = Number.parseInt(raw, 10);
    return Number.isInteger(value) ? value : undefined;
}

function hasFittingPayload(event: DragEvent): boolean {
    return Array.from(event.dataTransfer?.types ?? []).includes("application/esf-type-id");
}

function canDropOnRow(event: DragEvent, slotType: string): boolean {
    const draggedSlotType = event.dataTransfer?.getData("application/esf-slot-type");
    return draggedSlotType === "Charge" || draggedSlotType === slotType;
}

function onRowDragOver(event: DragEvent, row: ModuleRow, slotType: string) {
    // Firefox does not expose custom drag data until `drop`, but it does expose
    // the MIME types. Accept the drag here and do the compatibility check once
    // the payload becomes readable in onRowDrop.
    if (!event.dataTransfer || !hasFittingPayload(event)) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = "copy";
    dropTarget.value = { slotType, slotIndex: row.slotIndex };
}

function onRowDragLeave(event: DragEvent) {
    const row = event.currentTarget as HTMLElement;
    if (event.relatedTarget instanceof Node && row.contains(event.relatedTarget)) return;
    dropTarget.value = null;
}

function onRowDrop(event: DragEvent, row: ModuleRow, slotType: string) {
    dropTarget.value = null;
    if (!event.dataTransfer || !canDropOnRow(event, slotType)) return;
    event.preventDefault();

    const typeId = draggedNumber(event, "application/esf-type-id");
    const sourceIndex = draggedNumber(event, "application/esf-slot-index");
    const sourceSlotType = event.dataTransfer.getData("application/esf-slot-type");
    if (typeId === undefined) return;

    if (sourceSlotType === "Charge") {
        fitManager.setCharge(slotType as FitSlotType, row.slotIndex, typeId);
    } else if (sourceIndex !== undefined) {
        fitManager.swapModule(
            sourceSlotType as FitSlotType,
            sourceIndex,
            slotType as FitSlotType,
            row.slotIndex,
        );
    } else {
        fitManager.addModule(typeId, slotType as FitSlotType, { index: row.slotIndex });
    }
}

function removeRowItem(row: ModuleRow, slotType: string) {
    if (row.typeId === 0) return;
    if (row.chargeTypeId !== null) {
        fitManager.removeCharge(slotType as FitSlotType, row.slotIndex);
    } else {
        fitManager.removeModule(slotType as FitSlotType, row.slotIndex);
    }
}

function selectTableRow(event: MouseEvent, row: ModuleRow, slotType: string) {
    selectedRow.value = { slotType, slotIndex: row.slotIndex };
    (event.currentTarget as HTMLElement)?.focus();
}

function onRowKeydown(event: KeyboardEvent, row: ModuleRow, slotType: string) {
    if (event.key === "Delete" || event.key === "Backspace") {
        event.preventDefault();
        removeRowItem(row, slotType);
        return;
    }
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    event.preventDefault();
    const rows = [...document.querySelectorAll<HTMLElement>("[data-fit-table-row]")];
    const current = rows.indexOf(event.currentTarget as HTMLElement);
    const next = rows[current + (event.key === "ArrowDown" ? 1 : -1)];
    next?.focus();
    next?.click();
}

function fillHighRack() {
    if (!ctxMenu.canFillRack) return;
    fitManager.fillHighRack(ctxMenu.typeId, ctxMenu.chargeTypeId ?? undefined);
    closeContextMenu();
}

function droneActivateAll() {
    const { typeId, passive } = droneCtx;
    closeDroneContextMenu();

    // Try to activate normally first — bandwidth and count caps
    // in fitManager.activateDrones handle the limits.
    const canActivate = capacity.maxActivatableDrones(typeId);

    if (canActivate >= passive) {
        // Enough room — just activate.
        fitManager.activateDrones(typeId, passive);
        return;
    }

    // Not enough room. Free up slots by deactivating OTHER drone
    // types, one at a time, until we have enough room or run out
    // of others to deactivate. This handles the "swap" case (5
    // Warriors active, user wants to activate Hammerheads) and the
    // "partial swap" case (2 Sentries active, user wants 5 EC-300s
    // but only 3 slots free — deactivate nothing, activate 3).
    const fit = currentFit.value;
    if (fit) {
        let needed = passive - canActivate;
        for (const d of fit.drones) {
            if (needed <= 0) break;
            if (d.typeId === typeId || d.states.Active === 0) continue;
            const deactivateCount = Math.min(d.states.Active, needed);
            fitManager.activateDrones(d.typeId, -deactivateCount);
            needed -= deactivateCount;
        }
    }

    // Now activate with the freed-up room.
    fitManager.activateDrones(typeId, passive);
}

function droneDeactivateAll() {
    const { typeId, active } = droneCtx;
    closeDroneContextMenu();
    fitManager.activateDrones(typeId, -active);
}

function droneRemoveOne() {
    const { typeId } = droneCtx;
    closeDroneContextMenu();
    fitManager.removeDrone(typeId, 1);
}

function droneRemoveAll() {
    const { typeId, active, passive } = droneCtx;
    closeDroneContextMenu();
    fitManager.removeDrone(typeId, active + passive);
}

interface ModuleRow {
    typeId: number;
    name: string;
    slotIndex: number;
    state: string;
    dps: number | null;
    volley: number | null;
    cpu: number;
    power: number;
    optimal: string | null;
    capUse: number | null;
    chargeName: string | null;
    chargeTypeId: number | null;
}

interface SlotGroup {
    label: string;
    slotType: string;
    rows: ModuleRow[];
}

function attrVal(item: any, name: string): number | null {
    const sdeData = sde.value;
    if (!sdeData || !item) return null;
    const id = sdeData.attributeNameToId.get(name);
    if (id === undefined) return null;
    const a = item.attributes.get(id);
    if (!a) return null;
    return a.value ?? a.base_value ?? null;
}

function fmtNum(v: number | null, decimals = 1): string {
    if (v === null || v === 0) return "";
    return v.toLocaleString("en", { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
}

function fmtRange(item: any): string | null {
    const sdeData = sde.value;
    if (!sdeData) return null;

    // Turrets: maxRange + falloff
    const maxRange = attrVal(item, "maxRange");
    if (maxRange && maxRange > 0) {
        const falloff = attrVal(item, "falloff") ?? 0;
        const km = (v: number) => (v / 1000).toFixed(1);
        if (falloff > 0) return `${km(maxRange)}+${km(falloff)}km`;
        return `${km(maxRange)}km`;
    }

    // Missiles: charge maxVelocity × explosionDelay / 1000
    if (item.charge) {
        const vel = attrVal(item.charge, "maxVelocity");
        const ft = attrVal(item.charge, "explosionDelay");
        if (vel && ft && vel > 0 && ft > 0) {
            return `${((vel * ft) / 1000 / 1000).toFixed(1)}km`;
        }
    }

    return null;
}

const SLOT_ORDER: Array<{ type: string; label: string }> = [
    { type: "High", label: "High Slots" },
    { type: "Medium", label: "Med Slots" },
    { type: "Low", label: "Low Slots" },
    { type: "Rig", label: "Rigs" },
    { type: "SubSystem", label: "Subsystems" },
];

const groups = computed<SlotGroup[]>(() => {
    const fit = currentFit.value;
    const s = stats.value;
    const sdeData = sde.value;
    if (!fit || !s || !sdeData) return [];

    const slots = capacity.slotCounts.value;
    const result: SlotGroup[] = [];

    for (const { type, label } of SLOT_ORDER) {
        // T3 strategic cruisers receive their high/medium/low slots from
        // fitted subsystems. During some engine states the hull capacity can
        // therefore report zero even though the loaded fit already contains
        // modules in those slots. Never hide occupied slots: the fit itself
        // is authoritative about the minimum number of rows we must render.
        const highestOccupiedSlot = fit.modules.reduce(
            (highest, module) => module.slot.type === type
                ? Math.max(highest, module.slot.index)
                : highest,
            0,
        );
        const slotCount = Math.max(
            slots[type as keyof typeof slots] ?? 0,
            highestOccupiedSlot,
        );
        if (slotCount === 0) continue;

        const rows: ModuleRow[] = [];

        for (let idx = 1; idx <= slotCount; idx++) {
            const mod = fit.modules.find(
                (m) => m.slot.type === type && m.slot.index === idx,
            );

            if (mod) {
                const typeName = sdeData.types.get(mod.typeId)?.name ?? `#${mod.typeId}`;
                const engineItem = s.items.find(
                    (i) => i.type_id === mod.typeId && i.slot?.type === type && i.slot?.index === mod.slot.index,
                );

                let chargeName: string | null = null;
                if (mod.charge) {
                    chargeName = sdeData.types.get(mod.charge.typeId)?.name ?? null;
                }

                rows.push({
                    typeId: mod.typeId,
                    name: typeName,
                    slotIndex: idx,
                    state: mod.state === "Preview" ? "Active" : mod.state,
                    dps: engineItem ? attrVal(engineItem, "damagePerSecondWithReload") : null,
                    volley: engineItem ? attrVal(engineItem, "damageVolley") : null,
                    cpu: engineItem ? (attrVal(engineItem, "cpu") ?? 0) : 0,
                    power: engineItem ? (attrVal(engineItem, "power") ?? 0) : 0,
                    optimal: engineItem ? fmtRange(engineItem) : null,
                    capUse: engineItem ? attrVal(engineItem, "capacitorNeed") : null,
                    chargeName,
                    chargeTypeId: mod.charge?.typeId ?? null,
                });
            } else {
                // Empty slot
                rows.push({
                    typeId: 0,
                    name: "",
                    slotIndex: idx,
                    state: "Passive",
                    dps: null,
                    volley: null,
                    cpu: 0,
                    power: 0,
                    optimal: null,
                    capUse: null,
                    chargeName: null,
                    chargeTypeId: null,
                });
            }
        }

        result.push({ label, slotType: type, rows });
    }

    return result;
});

const selectedEntry = computed(() => {
    if (!selectedRow.value) return null;
    const group = groups.value.find(item => item.slotType === selectedRow.value!.slotType);
    const row = group?.rows.find(item => item.slotIndex === selectedRow.value!.slotIndex);
    return row && group ? { row, slotType: group.slotType } : null;
});

const selectedCanFillRack = computed(() => {
    const selected = selectedEntry.value;
    return Boolean(selected && selected.slotType === "High" && selected.row.typeId
        && (capacity.needsLauncherHardpoint(selected.row.typeId) || capacity.needsTurretHardpoint(selected.row.typeId))
        && capacity.hasFreeSlot("High") && capacity.hasHardpointFor(selected.row.typeId));
});

function removeSelectedEntry() {
    const selected = selectedEntry.value;
    if (selected) removeRowItem(selected.row, selected.slotType);
}

function fillSelectedRack() {
    const selected = selectedEntry.value;
    if (!selected || !selectedCanFillRack.value) return;
    fitManager.fillHighRack(selected.row.typeId, selected.row.chargeTypeId ?? undefined);
}

const hasDrones = computed(() => (currentFit.value?.drones?.length ?? 0) > 0);

const droneRows = computed(() => {
    const fit = currentFit.value;
    const sdeData = sde.value;
    const s = stats.value;
    if (!fit || !sdeData || !s) return [];

    return fit.drones.map((d) => {
        const typeName = sdeData.types.get(d.typeId)?.name ?? `#${d.typeId}`;
        const count = d.states.Active + d.states.Passive;
        // Find an engine item for this drone type
        const engineItem = s.items.find((i) => i.type_id === d.typeId);
        const dps = engineItem ? attrVal(engineItem, "damagePerSecondWithReload") : null;
        return {
            typeId: d.typeId,
            name: typeName,
            active: d.states.Active,
            passive: d.states.Passive,
            count,
            dps,
            totalDps: dps !== null ? dps * d.states.Active : null,
        };
    });
});
</script>

<template>
    <div
        class="w-full overflow-x-hidden text-xs text-gray-300"
        :class="constrained ? 'h-full overflow-y-auto' : ''"
    >
        <div v-if="selectedEntry?.row.typeId" class="sticky left-0 top-0 z-20 flex items-center gap-2 border-b border-blue-400/10 bg-[#0b0d11]/95 px-3 py-1.5 backdrop-blur sm:hidden">
            <span class="min-w-0 flex-1 truncate text-fine text-gray-400">{{ selectedEntry.row.name }}</span>
            <button v-if="selectedCanFillRack" type="button" class="rounded bg-blue-400/10 px-2 py-1 text-fine text-blue-300" @click="fillSelectedRack">Fill highs</button>
            <button type="button" class="rounded bg-red-400/10 px-2 py-1 text-fine text-red-300" @click="removeSelectedEntry">{{ selectedEntry.row.chargeTypeId !== null ? 'Unload' : 'Remove' }}</button>
        </div>
        <table ref="tableRef" class="w-full border-collapse" style="table-layout: fixed;">
            <colgroup>
                <col
                    v-for="(width, index) in columnWidths"
                    :key="index"
                    :style="width === null ? undefined : { width: `${width}px` }"
                >
            </colgroup>
            <thead class="sticky top-0 z-10" style="background-color: #0a0a0a;">
                <tr class="text-[10px] uppercase tracking-wider text-gray-500 border-b border-white/[0.06]">
                    <th
                        v-for="(label, index) in ['Name', 'DPS', 'Volley', 'CPU', 'PG', 'Range', 'Charge']"
                        :key="label"
                        class="group/column relative py-1 font-medium"
                        :class="index === 0 || index === 6 ? 'px-3 text-left' : 'px-2 text-right'"
                    >
                        {{ label }}
                        <span
                            v-if="index < 6"
                            class="absolute -right-1 top-0 z-20 h-full w-2 cursor-col-resize touch-none after:absolute after:bottom-1 after:left-1/2 after:top-1 after:w-px after:bg-white/0 after:transition-colors hover:after:bg-blue-400/70 group-hover/column:after:bg-white/10"
                            role="separator"
                            :aria-label="`Resize ${label} column`"
                            @pointerdown="startColumnResize($event, index)"
                        />
                    </th>
                </tr>
            </thead>
            <tbody>
                <template v-for="group in groups" :key="group.slotType">
                    <!-- Slot group header -->
                    <tr class="border-t border-white/[0.04]">
                        <td colspan="7" class="py-1 px-3 text-[10px] font-bold uppercase tracking-wider text-gray-500">
                            — {{ group.rows.length }} {{ group.label }} —
                        </td>
                    </tr>
                    <!-- Module rows -->
                    <tr
                        v-for="row in group.rows"
                        :key="`${group.slotType}-${row.slotIndex}`"
                        class="hover:bg-white/[0.03] border-t border-white/[0.02] cursor-context-menu"
                        :class="{
                            'opacity-40': row.state === 'Passive' && row.typeId !== 0,
                            'opacity-25': row.typeId === 0,
                            'bg-blue-400/[0.07] outline outline-1 -outline-offset-1 outline-blue-400/20': selectedRow?.slotType === group.slotType && selectedRow?.slotIndex === row.slotIndex,
                            'bg-cyan-400/[0.12] outline outline-1 -outline-offset-1 outline-cyan-300/50': dropTarget?.slotType === group.slotType && dropTarget?.slotIndex === row.slotIndex,
                        }"
                        tabindex="0"
                        data-fit-table-row
                        @click="selectTableRow($event, row, group.slotType)"
                        @keydown="onRowKeydown($event, row, group.slotType)"
                        @contextmenu="openContextMenu($event, row, group.slotType)"
                        @dblclick="removeRowItem(row, group.slotType)"
                        @dragover="onRowDragOver($event, row, group.slotType)"
                        @dragleave="onRowDragLeave"
                        @drop="onRowDrop($event, row, group.slotType)"
                    >
                        <td class="py-1 px-3 flex items-center gap-1.5">
                            <template v-if="row.typeId !== 0">
                                <img
                                    :src="`https://images.evetech.net/types/${row.typeId}/icon?size=32`"
                                    class="w-5 h-5 rounded-sm flex-none"
                                    loading="lazy"
                                />
                                <span class="truncate">{{ row.name }}</span>
                            </template>
                            <span v-else class="text-gray-600 italic">Empty Slot</span>
                        </td>
                        <td class="text-right py-1 px-2 tabular-nums">{{ fmtNum(row.dps) }}</td>
                        <td class="text-right py-1 px-2 tabular-nums">{{ fmtNum(row.volley, 0) }}</td>
                        <td class="text-right py-1 px-2 tabular-nums">{{ fmtNum(row.cpu, 0) }}</td>
                        <td class="text-right py-1 px-2 tabular-nums">{{ fmtNum(row.power, 0) }}</td>
                        <td class="text-right py-1 px-2 tabular-nums text-gray-400">{{ row.optimal ?? "" }}</td>
                        <td class="py-1 px-3 text-gray-500 truncate">{{ row.chargeName ?? "" }}</td>
                    </tr>
                </template>

                <!-- Drones -->
                <template v-if="hasDrones">
                    <tr class="border-t border-white/[0.04]">
                        <td colspan="7" class="py-1 px-3 text-[10px] font-bold uppercase tracking-wider text-gray-500">
                            — Drones —
                        </td>
                    </tr>
                    <tr
                        v-for="drone in droneRows"
                        :key="drone.typeId"
                        class="hover:bg-white/[0.03] border-t border-white/[0.02] cursor-context-menu"
                        @contextmenu="openDroneContextMenu($event, drone)"
                    >
                        <td class="py-1 px-3">
                            <div class="flex items-center gap-1.5 overflow-visible">
                                <img
                                    :src="`https://images.evetech.net/types/${drone.typeId}/icon?size=32`"
                                    class="w-5 h-5 rounded-sm flex-none"
                                    loading="lazy"
                                />
                                <span class="truncate">{{ drone.name }}</span>
                                <span class="text-gray-500 flex-none">×{{ drone.count }}</span>
                                <span v-if="drone.active > 0" class="text-green-500 flex-none">({{ drone.active }} active)</span>
                            </div>
                        </td>
                        <td class="text-right py-1 px-2 tabular-nums">{{ fmtNum(drone.totalDps) }}</td>
                        <td class="text-right py-1 px-2 tabular-nums"></td>
                        <td class="text-right py-1 px-2 tabular-nums"></td>
                        <td class="text-right py-1 px-2 tabular-nums"></td>
                        <td class="text-right py-1 px-2 tabular-nums"></td>
                        <td class="py-1 px-3"></td>
                    </tr>
                </template>
            </tbody>
            <tfoot>
                <tr><td colspan="7" class="h-4"></td></tr>
            </tfoot>
        </table>
        <!-- Drone context menu -->
        <Teleport to="body">
            <div
                v-if="droneCtx.visible"
                ref="droneMenuEl"
                class="fixed z-[1100] whitespace-nowrap rounded-md border border-white/[0.12] shadow-xl shadow-black/70 text-xs text-gray-200 overflow-visible"
                style="background-color: #1a1a1a;"
                :style="{ left: droneCtx.x + 'px', top: droneCtx.y + 'px' }"
            >
                <div class="flex flex-col py-1">
                    <button
                        v-if="droneCtx.active > 0"
                        type="button"
                        class="block w-full text-left px-3 py-1.5 hover:bg-white/[0.06]"
                        @click="droneDeactivateAll"
                    >
                        Deactivate All
                    </button>
                    <button
                        v-else
                        type="button"
                        class="block w-full text-left px-3 py-1.5 hover:bg-white/[0.06]"
                        @click="droneActivateAll"
                    >
                        Activate All
                    </button>
                </div>
                <div class="border-t border-white/[0.06]"></div>
                <div class="flex flex-col py-1">
                    <button
                        type="button"
                        class="block w-full text-left px-3 py-1.5 hover:bg-white/[0.06]"
                        @click="droneRemoveOne"
                    >
                        Remove One
                    </button>
                    <button
                        type="button"
                        class="block w-full text-left px-3 py-1.5 hover:bg-white/[0.06] text-red-400"
                        @click="droneRemoveAll"
                    >
                        Remove All
                    </button>
                </div>
            </div>
        </Teleport>

        <!-- Module context menu -->
        <FitContextMenu
            :visible="ctxMenu.visible"
            :x="ctxMenu.x"
            :y="ctxMenu.y"
            :type-id="ctxMenu.typeId"
            :slot-type="ctxMenu.slotType"
            :slot-index="ctxMenu.slotIndex"
            :current-state="ctxMenu.currentState"
            :has-charge="ctxMenu.hasCharge"
            :can-fill-rack="ctxMenu.canFillRack"
            :empty="ctxMenu.empty"
            @close="closeContextMenu"
            @fill-rack="fillHighRack"
        />
    </div>
</template>
