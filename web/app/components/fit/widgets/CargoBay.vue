<script setup lang="ts">
/**
 * Cargo hold drawer — renders below the ResourceBar when the user
 * clicks the "Cargo" cell. The cell itself shows the summary fill
 * bar; this drawer is the detail list + drag-drop target.
 *
 * Interactions:
 *   - Drag any typed item from the sidebar → addCargoItem (merges by
 *     type id, so dropping the same type twice increments)
 *   - "−" / "+" on a row → decrement / increment quantity
 *   - "×" on a row → remove the entry entirely
 *
 * Cargo doesn't have a dedicated sidebar tab yet, so the supported
 * drop source is "whatever's in the sidebar" — modules, charges, and
 * drones all get dumped into cargo as-is. Useful for "I want a few
 * spare modules + some extra ammo in the hold". When we add a Cargo
 * tab with sensible filters, the drop handler can tighten up.
 *
 * Always mounted by the parent only when the drawer is expanded, so
 * no v-if guard here — the component being alive is the trigger.
 */

const { currentFit, setFit } = useCurrentFit();
const { sde } = useEveData();
const capacity = useHullCapacity();

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

interface CargoRow {
    typeId: number;
    name: string;
    volume: number;
    quantity: number;
    totalVolume: number;
}

const rows = computed<CargoRow[]>(() => {
    const fit = currentFit.value;
    const sdeData = sde.value;
    if (!fit || !sdeData) return [];
    return fit.cargo.map((c) => {
        const type = sdeData.types.get(c.typeId);
        const vol = (type?.volume as number | undefined) ?? 0;
        return {
            typeId: c.typeId,
            name: String(type?.name ?? `Type ${c.typeId}`),
            volume: vol,
            quantity: c.quantity,
            totalVolume: vol * c.quantity,
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
    const typeId = parseNum(e.dataTransfer.getData("application/esf-type-id"));
    if (typeId === null) return;
    // Hard cargo capacity check — refuse drops that would overflow
    // the hold with a red flash instead of silently discarding.
    if (!capacity.canAddCargo(typeId, 1)) {
        flashReject();
        return;
    }
    addOrIncrement(typeId, 1);
}

/**
 * Merge-by-type add. Used for drops and the "+1" row button. setFit
 * preserves currentFitId so saved fits stay saved after cargo edits.
 */
function addOrIncrement(typeId: number, delta: number) {
    const fit = currentFit.value;
    if (!fit) return;
    const idx = fit.cargo.findIndex((c) => c.typeId === typeId);
    const nextCargo = [...fit.cargo];
    if (idx >= 0) {
        const current = nextCargo[idx]!;
        const nextQty = current.quantity + delta;
        if (nextQty <= 0) {
            nextCargo.splice(idx, 1);
        } else {
            nextCargo[idx] = { ...current, quantity: nextQty };
        }
    } else if (delta > 0) {
        nextCargo.push({ typeId, quantity: delta });
    }
    setFit({ ...fit, cargo: nextCargo });
}

function onIncrement(row: CargoRow) {
    if (!capacity.canAddCargo(row.typeId, 1)) {
        flashReject();
        return;
    }
    addOrIncrement(row.typeId, 1);
}

function onDecrement(row: CargoRow) {
    addOrIncrement(row.typeId, -1);
}

function onRemove(row: CargoRow) {
    const fit = currentFit.value;
    if (!fit) return;
    setFit({
        ...fit,
        cargo: fit.cargo.filter((c) => c.typeId !== row.typeId),
    });
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
            class="flex items-center justify-between px-3 py-2 border-b border-white/[0.06]"
        >
            <div class="flex items-center gap-2">
                <Icon name="lucide:package" class="w-3.5 h-3.5 text-blue-400/80" />
                <span class="text-fine font-bold uppercase tracking-[0.15em] text-blue-400/80">
                    Cargo Hold
                </span>
            </div>
            <span class="text-fine text-gray-600 flex items-center gap-1">
                <Icon name="lucide:mouse-pointer-click" class="w-3 h-3" />
                Drag items from sidebar
            </span>
        </div>

        <!-- Rows / empty state -->
        <div v-if="rows.length === 0" class="px-3 py-8 text-center">
            <Icon
                name="lucide:package-plus"
                class="w-8 h-8 text-gray-700 mx-auto mb-2"
            />
            <div class="text-fine text-gray-500">Cargo hold is empty</div>
            <div class="text-fine text-gray-600 mt-0.5">
                Drag any item here from the sidebar
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
                    <div class="text-fine text-gray-600 tabular-nums">
                        {{ row.volume }}m³ each
                    </div>
                </div>

                <!-- Quantity stepper -->
                <div class="flex items-center gap-1 flex-shrink-0">
                    <button
                        type="button"
                        v-tooltip="'Remove one'"
                        class="flex items-center justify-center w-6 h-6 rounded-md text-gray-500 hover:text-gray-200 hover:bg-white/[0.06] transition-colors"
                        @click="onDecrement(row)"
                    >
                        <Icon name="lucide:minus" class="w-3 h-3" />
                    </button>
                    <span class="text-xs text-gray-300 tabular-nums min-w-[2ch] text-center">
                        {{ row.quantity.toLocaleString("en-US") }}
                    </span>
                    <button
                        type="button"
                        v-tooltip="'Add one'"
                        class="flex items-center justify-center w-6 h-6 rounded-md text-gray-500 hover:text-gray-200 hover:bg-white/[0.06] transition-colors"
                        @click="onIncrement(row)"
                    >
                        <Icon name="lucide:plus" class="w-3 h-3" />
                    </button>
                </div>

                <div class="text-fine text-gray-500 tabular-nums flex-shrink-0 min-w-[4ch] text-right">
                    {{ row.totalVolume.toFixed(1) }}m³
                </div>

                <button
                    type="button"
                    v-tooltip="'Remove'"
                    class="flex items-center justify-center w-6 h-6 rounded-md text-gray-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                    @click="onRemove(row)"
                >
                    <Icon name="lucide:x" class="w-3 h-3" />
                </button>
            </div>
        </div>
    </div>
</template>
