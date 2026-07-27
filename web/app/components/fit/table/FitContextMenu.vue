<script setup lang="ts">
/**
 * Right-click context menu for fitted modules in the table view.
 *
 * Shows:
 *   - State toggles (Offline / Online / Active / Overload)
 *   - Unfit (remove module)
 *   - Compatible charges submenu (for modules that accept ammo/charges)
 *   - Remove charge (if one is loaded)
 */

import type { FitSlotType, FitState } from "~/composables/fit/types";

const props = defineProps<{
    visible: boolean;
    x: number;
    y: number;
    typeId: number;
    slotType: FitSlotType;
    slotIndex: number;
    currentState: string;
    hasCharge: boolean;
    empty?: boolean;
}>();

const emit = defineEmits<{
    close: [];
}>();

const { sde } = useEveData();
const fitManager = useFitManager();

// Close on outside click or Escape. The menu itself stops propagation
// so clicks inside don't trigger this.
const menuEl = ref<HTMLElement | null>(null);

function onClickOutside(e: MouseEvent) {
    if (menuEl.value?.contains(e.target as Node)) return;
    emit("close");
}

function onKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") emit("close");
}

onMounted(() => {
    document.addEventListener("mousedown", onClickOutside);
    document.addEventListener("keydown", onKeydown);
});

onUnmounted(() => {
    document.removeEventListener("mousedown", onClickOutside);
    document.removeEventListener("keydown", onKeydown);
});

// State actions
const stateOptions = computed(() => {
    const current = props.currentState;
    const opts: Array<{ label: string; state: FitState; active: boolean }> = [
        { label: "Offline", state: "Passive", active: current === "Passive" },
        { label: "Online", state: "Online", active: current === "Online" },
        { label: "Active", state: "Active", active: current === "Active" },
        { label: "Overload", state: "Overload", active: current === "Overload" },
    ];
    return opts;
});

function setState(state: FitState) {
    fitManager.setModuleState(props.slotType, props.slotIndex, state);
    emit("close");
}

function overloadRack() {
    fitManager.toggleRackOverload(props.slotType);
    emit("close");
}

function unfit() {
    fitManager.removeModule(props.slotType, props.slotIndex);
    emit("close");
}

function removeCharge() {
    fitManager.removeCharge(props.slotType, props.slotIndex);
    emit("close");
}

function loadCharge(chargeTypeId: number) {
    fitManager.setCharge(props.slotType, props.slotIndex, chargeTypeId);
    emit("close");
}

function loadChargeAll(chargeTypeId: number) {
    fitManager.setChargeAuto(chargeTypeId);
    emit("close");
}

// Find compatible charges for this module
const showCharges = ref(false);

const compatibleCharges = computed(() => {
    const sdeData = sde.value;
    if (!sdeData) return [];

    const attrId = (name: string) => sdeData.attributeNameToId.get(name);
    const chargeSizeAttr = attrId("chargeSize");
    const chargeGroupAttrs = [
        attrId("chargeGroup1"),
        attrId("chargeGroup2"),
        attrId("chargeGroup3"),
        attrId("chargeGroup4"),
        attrId("chargeGroup5"),
    ];
    if (chargeSizeAttr === undefined) return [];

    const td = sdeData.typeDogma.get(props.typeId);
    if (!td?.dogmaAttributes) return [];
    const modAttrs = td.dogmaAttributes as Array<{ attributeID: number; value: number }>;

    // Collect accepted charge group IDs
    const acceptedGroups = new Set<number>();
    for (const ga of chargeGroupAttrs) {
        if (ga === undefined) continue;
        const found = modAttrs.find((a) => a.attributeID === ga);
        if (found) acceptedGroups.add(Number(found.value));
    }
    if (acceptedGroups.size === 0) return [];

    // Module's charge size constraint
    const modChargeSizeEntry = modAttrs.find((a) => a.attributeID === chargeSizeAttr);
    const modChargeSize = modChargeSizeEntry?.value ?? null;

    // Scan all published types for matching charges
    const charges: Array<{ typeId: number; name: string }> = [];
    for (const [typeId, type] of sdeData.types) {
        if (!type.published) continue;
        if (type.categoryID !== 8) continue; // category 8 = Charge
        if (!acceptedGroups.has(type.groupID)) continue;

        // Size match
        if (modChargeSize !== null) {
            const chargeTd = sdeData.typeDogma.get(typeId);
            if (chargeTd?.dogmaAttributes) {
                const csAttr = (chargeTd.dogmaAttributes as Array<{ attributeID: number; value: number }>)
                    .find((a) => a.attributeID === chargeSizeAttr!);
                if (csAttr && csAttr.value !== modChargeSize) continue;
            }
        }

        charges.push({ typeId, name: type.name });
    }

    charges.sort((a, b) => a.name.localeCompare(b.name));
    return charges;
});

const hasCompatibleCharges = computed(() => compatibleCharges.value.length > 0);
</script>

<template>
    <Teleport to="body">
        <div
            v-if="visible"
            ref="menuEl"
            class="fixed z-[100] whitespace-nowrap rounded-md border border-white/[0.12] shadow-xl shadow-black/70 text-xs text-gray-200 overflow-visible"
            style="background-color: #1a1a1a;"
            :style="{ left: x + 'px', top: y + 'px' }"
        >
            <template v-if="!empty">
                <!-- State toggles -->
                <div class="py-1">
                    <button
                        v-for="opt in stateOptions"
                        :key="opt.state"
                        type="button"
                        class="w-full text-left px-3 py-1.5 hover:bg-white/[0.06] flex items-center justify-between"
                        :class="{ 'text-blue-400': opt.active }"
                        @click="setState(opt.state)"
                    >
                        <span>{{ opt.label }}</span>
                        <span v-if="opt.active" class="text-[10px] text-gray-500">current</span>
                    </button>
                </div>

                <div class="border-t border-white/[0.06]"></div>

                <!-- Overload rack toggle -->
                <div class="py-1">
                    <button
                        type="button"
                        class="w-full text-left px-3 py-1.5 hover:bg-white/[0.06] text-orange-400"
                        @click="overloadRack"
                    >
                        Overload Rack
                    </button>
                </div>

                <div class="border-t border-white/[0.06]"></div>
            </template>

            <template v-if="!empty">
            <!-- Charge submenus — hover to open flyout -->
            <div v-if="hasCompatibleCharges" class="py-1">
                <!-- Load Charge (this module only) -->
                <div class="relative group/charge">
                    <button
                        type="button"
                        class="w-full text-left px-3 py-1.5 hover:bg-white/[0.06] group-hover/charge:bg-white/[0.06] flex items-center justify-between gap-4"
                    >
                        <span>Load Charge</span>
                        <Icon name="lucide:chevron-right" class="w-3 h-3 text-gray-500" />
                    </button>
                    <div
                        class="hidden group-hover/charge:block absolute left-full top-0 ml-0.5 min-w-[220px] max-h-[300px] overflow-y-auto rounded-md border border-white/[0.12] shadow-xl shadow-black/70"
                        style="background-color: #1a1a1a;"
                    >
                        <div class="py-1">
                            <button
                                v-for="charge in compatibleCharges"
                                :key="charge.typeId"
                                type="button"
                                class="w-full text-left px-3 py-1 hover:bg-white/[0.06] flex items-center gap-1.5"
                                @click="loadCharge(charge.typeId)"
                            >
                                <img
                                    :src="`https://images.evetech.net/types/${charge.typeId}/icon?size=32`"
                                    class="w-4 h-4 rounded-sm flex-none"
                                    loading="lazy"
                                />
                                <span>{{ charge.name }}</span>
                            </button>
                        </div>
                    </div>
                </div>
                <!-- Load Charge (All) — applies to all compatible modules -->
                <div class="relative group/chargeall">
                    <button
                        type="button"
                        class="w-full text-left px-3 py-1.5 hover:bg-white/[0.06] group-hover/chargeall:bg-white/[0.06] flex items-center justify-between gap-4"
                    >
                        <span>Load Charge Rack</span>
                        <Icon name="lucide:chevron-right" class="w-3 h-3 text-gray-500" />
                    </button>
                    <div
                        class="hidden group-hover/chargeall:block absolute left-full top-0 ml-0.5 min-w-[220px] max-h-[300px] overflow-y-auto rounded-md border border-white/[0.12] shadow-xl shadow-black/70"
                        style="background-color: #1a1a1a;"
                    >
                        <div class="py-1">
                            <button
                                v-for="charge in compatibleCharges"
                                :key="charge.typeId"
                                type="button"
                                class="w-full text-left px-3 py-1 hover:bg-white/[0.06] flex items-center gap-1.5"
                                @click="loadChargeAll(charge.typeId)"
                            >
                                <img
                                    :src="`https://images.evetech.net/types/${charge.typeId}/icon?size=32`"
                                    class="w-4 h-4 rounded-sm flex-none"
                                    loading="lazy"
                                />
                                <span>{{ charge.name }}</span>
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Remove charge -->
            <div v-if="hasCharge" class="py-1 border-t border-white/[0.06]">
                <button
                    type="button"
                    class="w-full text-left px-3 py-1.5 hover:bg-white/[0.06] text-yellow-400"
                    @click="removeCharge"
                >
                    Remove Charge
                </button>
            </div>

            <div class="border-t border-white/[0.06]"></div>

            <!-- Unfit -->
            <div class="py-1">
                <button
                    type="button"
                    class="w-full text-left px-3 py-1.5 hover:bg-white/[0.06] text-red-400"
                    @click="unfit"
                >
                    Unfit Module
                </button>
            </div>
            </template>

            <!-- Empty slot -->
            <div v-if="empty" class="py-2 px-3 text-gray-500">
                Empty {{ slotType }} Slot
            </div>
        </div>
    </Teleport>
</template>
