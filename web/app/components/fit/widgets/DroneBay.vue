<script setup lang="ts">
/**
 * Drone bay drawer — renders below the ResourceBar when the user
 * clicks the "Drone Bay" cell. The cell itself shows the summary
 * fill bar + bandwidth numbers; this drawer is the detail list +
 * drag-drop target.
 *
 * Interactions:
 *   - Drag drones from the sidebar onto this whole panel → addDrone
 *   - Click the zap badge on a row → toggle one drone active↔passive
 *   - "−" on a row → remove one drone (passive first, then active)
 *   - "×" on a row → remove the whole group
 *
 * The bay is always rendered when the parent mounts this drawer (it's
 * only mounted when `expanded === 'drones'`), so there's no
 * `v-if="isVisible"` check here — the drawer being open is the
 * trigger. Empty state shows a drop-here prompt.
 */

const { currentFit } = useCurrentFit();
const { sde } = useEveData();
const capacity = useHullCapacity();
const { totalActiveDrones, maxActiveDrones } = capacity;
const fitManager = useFitManager();

const dropActive = ref(false);
const dropRejected = ref(false);
let rejectTimer: ReturnType<typeof setTimeout> | null = null;

function flashReject() {
    dropRejected.value = true;
    if (rejectTimer) clearTimeout(rejectTimer);
    rejectTimer = setTimeout(() => {
        dropRejected.value = false;
    }, 1200);
}

interface DroneRow {
    typeId: number;
    name: string;
    volume: number;
    bandwidth: number;
    active: number;
    passive: number;
}

const rows = computed<DroneRow[]>(() => {
    const fit = currentFit.value;
    const sdeData = sde.value;
    if (!fit || !sdeData) return [];
    const bwAttrId = sdeData.attributeNameToId.get("droneBandwidthUsed");
    return fit.drones.map((d) => {
        const type = sdeData.types.get(d.typeId);
        const td = sdeData.typeDogma.get(d.typeId);
        const bwAttr = td?.dogmaAttributes
            ? (td.dogmaAttributes as Array<{ attributeID: number; value: number }>).find(
                (a) => a.attributeID === bwAttrId,
            )
            : undefined;
        return {
            typeId: d.typeId,
            name: String(type?.name ?? `Type ${d.typeId}`),
            volume: (type?.volume as number | undefined) ?? 0,
            bandwidth: bwAttr?.value ?? 0,
            active: d.states.Active,
            passive: d.states.Passive,
        };
    });
});

function parseNum(s: string | null | undefined): number | null {
    if (s === null || s === undefined || s === "") return null;
    const n = Number(s);
    return Number.isFinite(n) ? n : null;
}

function onDragOver(e: DragEvent) {
    if (!e.dataTransfer) return;
    e.dataTransfer.dropEffect = "copy";
    e.preventDefault();
    dropActive.value = true;
}

function onDragLeave() {
    dropActive.value = false;
}

function onDrop(e: DragEvent) {
    e.preventDefault();
    dropActive.value = false;
    if (!e.dataTransfer) return;
    const slotType = e.dataTransfer.getData("application/esf-slot-type");
    const typeId = parseNum(e.dataTransfer.getData("application/esf-type-id"));
    if (slotType !== "DroneBay" || typeId === null) return;
    // Volume check — addDrone would silently no-op, but an explicit
    // red-flash makes it clear WHY the drop did nothing. Bandwidth
    // isn't a refusal reason; overflow splits to Passive inside
    // addDrone, which is the expected "extra drones sitting in the
    // bay" behavior.
    if (!capacity.canAddDrone(typeId, 1)) {
        flashReject();
        return;
    }
    fitManager.addDrone(typeId, 1, "Active");
}

function onDecrement(row: DroneRow) {
    fitManager.removeDrone(row.typeId, 1);
}

function onRemoveAll(row: DroneRow) {
    fitManager.removeDrone(row.typeId, row.active + row.passive);
}

/** Move one drone from passive → active. Refuses (with a flash) when
 *  bandwidth is exhausted so the user understands why nothing moved. */
function onActivateOne(row: DroneRow) {
    if (row.passive <= 0) return;
    if (!capacity.canActivateDrone(row.typeId, 1)) {
        flashReject();
        return;
    }
    fitManager.activateDrones(row.typeId, 1);
}

/** Move one drone from active → passive. No cap check — recalling
 *  drones to the bay always succeeds. */
function onDeactivateOne(row: DroneRow) {
    if (row.active <= 0) return;
    fitManager.activateDrones(row.typeId, -1);
}
</script>

<template>
    <div
        class="rounded-lg bg-white/[0.04] border overflow-hidden transition-colors"
        :class="
            dropRejected
                ? 'border-red-500/60 ring-1 ring-red-500/40 bg-red-500/[0.04]'
                : dropActive
                    ? 'border-blue-400/60 ring-1 ring-blue-400/40 bg-blue-500/[0.04]'
                    : 'border-white/[0.08]'
        "
        @dragover="onDragOver"
        @dragleave="onDragLeave"
        @drop="onDrop"
    >
        <!-- Header -->
        <div
            class="flex items-center justify-between gap-3 px-3 py-2 border-b border-white/[0.06]"
        >
            <div class="flex items-center gap-2">
                <Icon name="lucide:radar" class="w-3.5 h-3.5 text-blue-400/80" />
                <span class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">
                    Drone Bay
                </span>
            </div>
            <div class="flex items-center gap-3 text-fine">
                <!-- In-space count vs the 5-drone hard cap. Tints amber
                     when the cap is hit so a refused "launch" click has
                     a clear on-screen reason. -->
                <span
                    class="tabular-nums"
                    :class="
                        totalActiveDrones >= maxActiveDrones
                            ? 'text-amber-300'
                            : 'text-gray-500'
                    "
                >
                    <Icon name="lucide:zap" class="w-3 h-3 inline-block -mt-0.5" />
                    {{ totalActiveDrones }} /
                    <span class="text-gray-400">{{ maxActiveDrones }}</span>
                    <span class="text-gray-600">in space</span>
                </span>
                <span class="w-px h-3 bg-white/[0.08]"></span>
                <span class="text-gray-600 flex items-center gap-1">
                    <Icon name="lucide:mouse-pointer-click" class="w-3 h-3" />
                    Drag from sidebar
                </span>
            </div>
        </div>

        <!-- Rows / empty state -->
        <div v-if="rows.length === 0" class="px-3 py-8 text-center">
            <Icon
                name="lucide:package-plus"
                class="w-8 h-8 text-gray-700 mx-auto mb-2"
            />
            <div class="text-fine text-gray-500">Drone bay is empty</div>
            <div class="text-fine text-gray-600 mt-0.5">
                Drag drones here from the Modules sidebar
            </div>
        </div>
        <div v-else>
            <div
                v-for="row in rows"
                :key="row.typeId"
                class="flex items-center gap-3 px-3 py-2 border-b border-white/[0.04] last:border-b-0 hover:bg-white/[0.02] transition-colors"
            >
                <img
                    :src="`/images/types/${row.typeId}/icon?size=32`"
                    :alt="row.name"
                    class="w-8 h-8 rounded flex-shrink-0 bg-black/30"
                    loading="lazy"
                />
                <div class="flex-1 min-w-0">
                    <NuxtLink
                        :to="`/item/${row.typeId}`"
                        class="text-xs text-gray-200 hover:text-blue-400 transition-colors truncate block"
                    >
                        {{ row.name }}
                    </NuxtLink>
                    <div class="text-fine text-gray-600 tabular-nums flex items-center gap-1.5">
                        <span>{{ row.volume }}m³</span>
                        <span class="text-gray-700">·</span>
                        <span>{{ row.bandwidth }}Mbit</span>
                    </div>
                </div>

                <!-- Active / passive split: two explicit badges so the
                     "in space" vs "in bay" states are visible at a glance.
                     Clicking the active badge recalls one (always allowed);
                     clicking the passive badge launches one (refused with
                     a flash when bandwidth is exhausted). -->
                <div class="flex items-center gap-1 flex-shrink-0">
                    <button
                        type="button"
                        v-tooltip="'In space — click to recall one to the bay'"
                        :disabled="row.active === 0"
                        class="flex items-center gap-1 px-2 py-0.5 rounded-md text-fine font-bold tabular-nums border transition-colors"
                        :class="
                            row.active > 0
                                ? 'bg-blue-500/15 text-blue-400 border-blue-500/30 hover:bg-blue-500/25'
                                : 'text-gray-700 border-white/[0.04]'
                        "
                        @click="onDeactivateOne(row)"
                    >
                        <Icon name="lucide:zap" class="w-3 h-3" />
                        {{ row.active }}
                    </button>
                    <button
                        type="button"
                        v-tooltip="'In bay — click to launch one into space'"
                        :disabled="row.passive === 0"
                        class="flex items-center gap-1 px-2 py-0.5 rounded-md text-fine font-bold tabular-nums border transition-colors"
                        :class="
                            row.passive > 0
                                ? 'bg-white/[0.04] text-gray-300 border-white/[0.08] hover:bg-white/[0.08] hover:text-gray-100'
                                : 'text-gray-700 border-white/[0.04]'
                        "
                        @click="onActivateOne(row)"
                    >
                        <Icon name="lucide:archive" class="w-3 h-3" />
                        {{ row.passive }}
                    </button>
                </div>

                <!-- −1 drone -->
                <button
                    type="button"
                    v-tooltip="'Remove one drone'"
                    class="flex items-center justify-center w-6 h-6 rounded-md text-gray-500 hover:text-gray-200 hover:bg-white/[0.06] transition-colors"
                    @click="onDecrement(row)"
                >
                    <Icon name="lucide:minus" class="w-3 h-3" />
                </button>

                <!-- Remove group -->
                <button
                    type="button"
                    v-tooltip="'Remove all'"
                    class="flex items-center justify-center w-6 h-6 rounded-md text-gray-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                    @click="onRemoveAll(row)"
                >
                    <Icon name="lucide:x" class="w-3 h-3" />
                </button>
            </div>
        </div>
    </div>
</template>
