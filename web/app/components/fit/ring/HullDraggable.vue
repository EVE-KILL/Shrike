<script setup lang="ts">
/**
 * Invisible 76% × 76% circular div over the hull image that accepts
 * drops from the hardware sidebar. Dropping a module anywhere on the
 * ship is a convenience shortcut — the drop auto-routes to the right
 * place based on the `application/esf-slot-type` data transfer key:
 *
 *   - Module slot types → addModule() to first free slot
 *   - "DroneBay"        → addDrone() as an active drone
 *   - Anything else     → ignored
 *
 * Dragging a module that's already fitted (i.e. `esf-slot-index` is
 * set) onto the hull is a no-op — the React lib treats hull-drop as
 * "add a new item from the listing", not "reshuffle an existing one".
 */
import styles from "./ShipFit.module.css";
import type { FitSlotType } from "../../../composables/fit/types";

const fitManager = useFitManager();

function parseNumber(raw: string): number | undefined {
    const n = Number.parseInt(raw, 10);
    return Number.isInteger(n) ? n : undefined;
}

function onDragOver(e: DragEvent) {
    e.preventDefault();
}

function onDrop(e: DragEvent) {
    e.preventDefault();
    if (!e.dataTransfer) return;

    const draggedTypeId = parseNumber(e.dataTransfer.getData("application/esf-type-id"));
    const draggedSlotIndex = parseNumber(e.dataTransfer.getData("application/esf-slot-index"));
    const draggedSlotType = e.dataTransfer.getData("application/esf-slot-type");

    if (draggedTypeId === undefined) return;
    // Dragging an already-fitted module back to the hull is a no-op.
    if (draggedSlotIndex !== undefined) return;

    if (draggedSlotType === "DroneBay") {
        fitManager.addDrone(draggedTypeId, 1, "Active");
        return;
    }

    const moduleSlotTypes: FitSlotType[] = ["High", "Medium", "Low", "Rig", "SubSystem"];
    if (moduleSlotTypes.includes(draggedSlotType as FitSlotType)) {
        fitManager.addModule(draggedTypeId, draggedSlotType as FitSlotType);
    }
}
</script>

<template>
    <div :class="styles.hullDraggable" @dragover="onDragOver" @drop="onDrop" />
</template>
