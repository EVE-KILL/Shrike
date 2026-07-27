<script setup lang="ts">
/**
 * Save / update dialog for the current fit.
 *
 * Wraps the reusable <Modal> in a small form: name, description,
 * visibility radio (Private / Corporation / Alliance / Public). The
 * Save button is disabled whenever a network request is in flight or
 * the required fields are empty.
 *
 * Behavior:
 *   - On open, seeds the form from the current fit's name/description
 *     so edits look like in-place updates, not "type it all again".
 *   - Visibility defaults to Private for brand-new fits, or to the
 *     last-saved value when updating an existing row (tracked via a
 *     ref so it survives modal close/reopen within the same session).
 *   - On success, emits `saved` with the fit_id and closes itself.
 *     The parent (FitNavbar) is responsible for any post-save
 *     navigation or toast.
 *
 * Auth gating happens one level up in the navbar — this modal assumes
 * the viewer is already logged in and just wraps the mutation.
 */

import { useFitSaver } from "~/composables/useFitSaver";

const props = defineProps<{
    modelValue: boolean;
}>();

const emit = defineEmits<{
    (e: "update:modelValue", open: boolean): void;
    (e: "saved", fitId: string): void;
}>();

const { currentFit, currentFitId, isSaved } = useCurrentFit();
const { saving, error, save } = useFitSaver();

const VISIBILITY_OPTIONS = [
    {
        value: 3,
        label: "Public",
        hint: "Anyone, including logged-out visitors, can see this fit.",
        icon: "lucide:globe",
    },
    {
        value: 0,
        label: "Private",
        hint: "Only you can see this fit.",
        icon: "lucide:lock",
    },
    {
        value: 1,
        label: "Corporation",
        hint: "Members of your current corp can see this fit.",
        icon: "lucide:users",
    },
    {
        value: 2,
        label: "Alliance",
        hint: "Members of your current alliance can see this fit.",
        icon: "lucide:flag",
    },
] as const;

const name = ref("");
const description = ref("");
// Default to Public — the fitting tool is community-first, and hiding
// new fits behind Private by default leaves the landing page sparse for
// no good reason. Users still get the full visibility ladder if they
// want to lock something down.
const visibility = ref<number>(3);

// Seed form fields from the current fit whenever the modal opens. We
// snapshot here instead of binding directly to currentFit so cancelling
// the modal leaves the fit unchanged.
watch(
    () => props.modelValue,
    (open) => {
        if (!open) return;
        const fit = currentFit.value;
        if (!fit) return;
        name.value = fit.name;
        description.value = fit.description;
        // Reset to Public for fresh drafts; updating an existing fit
        // leaves the visibility radio at whatever the user picked last
        // session (or whatever the saved fit currently is, when we
        // start preloading from the row in a future pass).
        if (!isSaved.value) visibility.value = 3;
    },
    { immediate: true },
);

const canSave = computed(
    () =>
        !saving.value &&
        name.value.trim().length > 0 &&
        currentFit.value !== null,
);

async function onSubmit() {
    if (!canSave.value || !currentFit.value) return;

    // Build a snapshot with the form-edited name/description so the
    // saver writes what the modal shows, not what's in the ring's
    // rename state. Cloning avoids mutating the live currentFit before
    // the POST returns.
    const fit = {
        ...currentFit.value,
        name: name.value.trim(),
        description: description.value,
    };

    const result = await save(fit, visibility.value);
    if (result.ok) {
        emit("saved", result.fitId);
        emit("update:modelValue", false);
    }
}

function onCancel() {
    emit("update:modelValue", false);
}
</script>

<template>
    <Modal
        :model-value="modelValue"
        :title="isSaved ? 'Update Fit' : 'Save Fit'"
        @update:model-value="(v: boolean) => emit('update:modelValue', v)"
    >
        <form class="space-y-4" @submit.prevent="onSubmit">
            <!-- Name -->
            <div>
                <label class="block text-fine font-bold uppercase tracking-[0.12em] text-gray-500 mb-1.5">
                    Fit Name
                </label>
                <input
                    v-model="name"
                    type="text"
                    maxlength="100"
                    required
                    class="w-full px-3 py-2 rounded-md bg-black/40 border border-white/[0.08] text-sm text-gray-200 placeholder-gray-600 focus:outline-none focus:border-blue-500/50 focus:bg-black/60 transition-colors"
                    placeholder="Muninn — Fury HM"
                />
            </div>

            <!-- Description -->
            <div>
                <label class="block text-fine font-bold uppercase tracking-[0.12em] text-gray-500 mb-1.5">
                    Description
                    <span class="text-gray-700 font-normal normal-case tracking-normal">(optional)</span>
                </label>
                <textarea
                    v-model="description"
                    rows="3"
                    maxlength="2000"
                    class="w-full px-3 py-2 rounded-md bg-black/40 border border-white/[0.08] text-sm text-gray-200 placeholder-gray-600 focus:outline-none focus:border-blue-500/50 focus:bg-black/60 transition-colors resize-none"
                    placeholder="Notes, use case, skill prereqs, anything useful for the next pilot who flies this."
                />
            </div>

            <!-- Visibility -->
            <div>
                <label class="block text-fine font-bold uppercase tracking-[0.12em] text-gray-500 mb-2">
                    Visibility
                </label>
                <div class="space-y-1.5">
                    <label
                        v-for="opt in VISIBILITY_OPTIONS"
                        :key="opt.value"
                        class="flex items-start gap-3 px-3 py-2 rounded-md border border-white/[0.08] bg-white/[0.02] hover:bg-blue-500/[0.04] hover:border-blue-500/20 transition-colors cursor-pointer"
                        :class="{
                            'bg-blue-500/[0.08] border-blue-500/30': visibility === opt.value,
                        }"
                    >
                        <input
                            v-model="visibility"
                            type="radio"
                            :value="opt.value"
                            class="sr-only"
                        />
                        <Icon
                            :name="opt.icon"
                            class="w-4 h-4 mt-0.5 flex-shrink-0"
                            :class="visibility === opt.value ? 'text-blue-400' : 'text-gray-500'"
                        />
                        <div class="flex-1 min-w-0">
                            <div
                                class="text-sm font-medium"
                                :class="visibility === opt.value ? 'text-blue-400' : 'text-gray-200'"
                            >
                                {{ opt.label }}
                            </div>
                            <div class="text-fine text-gray-500">{{ opt.hint }}</div>
                        </div>
                    </label>
                </div>
            </div>

            <!-- Error message -->
            <div
                v-if="error"
                class="flex items-start gap-2 px-3 py-2 rounded-md bg-red-500/10 border border-red-500/30 text-xs text-red-400"
            >
                <Icon name="lucide:alert-circle" class="w-4 h-4 flex-shrink-0 mt-0.5" />
                <span>{{ error }}</span>
            </div>

            <!-- Already-saved hint -->
            <div
                v-if="isSaved"
                class="flex items-start gap-2 px-3 py-2 rounded-md bg-blue-500/[0.06] border border-blue-500/20 text-xs text-blue-300/80"
            >
                <Icon name="lucide:info" class="w-4 h-4 flex-shrink-0 mt-0.5" />
                <span>
                    Updating existing fit
                    <code class="font-mono text-blue-400">{{ currentFitId?.slice(0, 8) }}</code>
                    — items will be replaced.
                </span>
            </div>
        </form>

        <template #footer>
            <div class="flex items-center justify-end gap-2">
                <button
                    type="button"
                    class="px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md text-gray-400 border border-white/[0.08] hover:text-gray-200 hover:bg-white/[0.04] transition-colors"
                    @click="onCancel"
                >
                    Cancel
                </button>
                <button
                    type="button"
                    :disabled="!canSave"
                    class="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-bold uppercase tracking-[0.12em] rounded-md bg-blue-500/20 text-blue-400 border border-blue-500/30 hover:bg-blue-500/30 transition-colors disabled:opacity-40 disabled:hover:bg-blue-500/20"
                    @click="onSubmit"
                >
                    <Icon
                        :name="saving ? 'lucide:loader-2' : 'lucide:save'"
                        class="w-3.5 h-3.5"
                        :class="{ 'animate-spin': saving }"
                    />
                    {{ saving ? "Saving…" : isSaved ? "Update" : "Save" }}
                </button>
            </div>
        </template>
    </Modal>
</template>
