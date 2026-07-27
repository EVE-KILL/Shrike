<script setup lang="ts">
// Settings → Preferences section. Extracted verbatim from pages/settings/[[tab]].vue.
const { user } = useAuth()

/**
 * Which clock date/time pickers draw in.
 *
 * This preference already existed — `useTimeZonePref`, backed by the `ek_tz`
 * cookie — but its only control lived inside the DateTimePicker popover. So a
 * setting that persists for a year could only be changed by opening a date
 * field somewhere on the site and knowing to look in the corner of it. It
 * belongs here, alongside the other preferences.
 *
 * Cookie-backed and shared via useState, so flipping it here updates every
 * picker already on the page. Applies to display only: pickers always store an
 * EVE wall-clock value, so switching zones never moves an instant you picked.
 */
const zone = useTimeZonePref()
const mounted = ref(false)
onMounted(() => { mounted.value = true })
// The viewer's IANA zone is unknowable during SSR, so label generically until mount.
const localName = computed(() => (mounted.value ? localZoneLabel() : 'Local'))

const zoneOptions = computed(() => [
    { value: 'eve' as const, label: 'EVE time', hint: 'UTC — the clock the game and the rest of the site use' },
    { value: 'local' as const, label: 'Local time', hint: localName.value },
])

// ── Preferences ─────────────────────────────────────────────────────────────
const defaultTabOptions: Record<string, { value: string; label: string }[]> = {
    character: [
        { value: 'dashboard', label: 'Dashboard' },
        { value: 'combined', label: 'Combined' },
        { value: 'kills', label: 'Kills' },
        { value: 'losses', label: 'Losses' },
        { value: 'history', label: 'History' },
        { value: 'achievements', label: 'Achievements' },
    ],
    corporation: [
        { value: 'dashboard', label: 'Dashboard' },
        { value: 'combined', label: 'Combined' },
        { value: 'kills', label: 'Kills' },
        { value: 'losses', label: 'Losses' },
        { value: 'members', label: 'Members' },
        { value: 'history', label: 'History' },
    ],
    alliance: [
        { value: 'dashboard', label: 'Dashboard' },
        { value: 'combined', label: 'Combined' },
        { value: 'kills', label: 'Kills' },
        { value: 'losses', label: 'Losses' },
        { value: 'corporations', label: 'Corporations' },
        { value: 'members', label: 'Members' },
    ],
    faction: [
        { value: 'combined', label: 'Combined' },
        { value: 'dashboard', label: 'Dashboard' },
        { value: 'kills', label: 'Kills' },
        { value: 'losses', label: 'Losses' },
    ],
    item: [
        { value: 'dashboard', label: 'Dashboard' },
        { value: 'kills', label: 'Kills' },
    ],
    system: [
        { value: 'dashboard', label: 'Dashboard' },
        { value: 'kills', label: 'Kills' },
    ],
    constellation: [
        { value: 'dashboard', label: 'Dashboard' },
        { value: 'kills', label: 'Kills' },
    ],
    region: [
        { value: 'dashboard', label: 'Dashboard' },
        { value: 'kills', label: 'Kills' },
    ],
}

const defaultTabLabels: Record<string, string> = {
    character: 'Character',
    corporation: 'Corporation',
    alliance: 'Alliance',
    faction: 'Faction',
    item: 'Items / Ships',
    system: 'System',
    constellation: 'Constellation',
    region: 'Region',
}

// Seed from user settings
const savedTabs = (user.value?.settings?.defaultTabs ?? {}) as Record<string, string>
const defaultTabs = reactive<Record<string, string>>(
    Object.fromEntries(
        Object.entries(defaultTabOptions).map(([key, opts]) => [key, savedTabs[key] ?? opts[0]?.value ?? ''])
    )
)

const prefsCookie = useCookie<Record<string, Record<string, string>>>('ek_prefs', { maxAge: 365 * 86400 })
const prefsSaving = ref(false)
const prefsSaved = ref(false)
const prefsError = ref('')
let prefsSavedTimer: ReturnType<typeof setTimeout> | null = null

const savePreferences = async () => {
    prefsSaving.value = true
    prefsError.value = ''
    try {
        const result = await apiFetch<{ defaultTabs: Record<string, string> }>('/api/user/preferences', {
            method: 'PUT',
            body: { defaultTabs: { ...defaultTabs } },
        })
        // Update auth state so changes take effect immediately
        useState<AuthUser | null>('auth-user').value = {
            ...user.value!,
            settings: { ...user.value!.settings, defaultTabs: result.defaultTabs },
        }
        // Sync to cookie for instant SSR access on future page loads
        prefsCookie.value = { ...prefsCookie.value, defaultTabs: result.defaultTabs }
        prefsSaved.value = true
        if (prefsSavedTimer) clearTimeout(prefsSavedTimer)
        prefsSavedTimer = setTimeout(() => { prefsSaved.value = false }, 2000)
    } catch (err: any) {
        // Without this the button just flips back to "Save" and the user has no
        // way to tell the PUT failed — they walk away believing it saved.
        prefsError.value = extractFetchError(err, 'Could not save preferences')
    } finally {
        prefsSaving.value = false
    }
}

onUnmounted(() => { if (prefsSavedTimer) clearTimeout(prefsSavedTimer) })
</script>

<template>
    <div class="space-y-4">
        <div class="glass-panel p-5">
            <div class="flex items-center justify-between mb-4">
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Default Page</h2>
                <div class="flex items-center gap-2">
                    <Transition
                        enter-active-class="transition duration-200"
                        enter-from-class="opacity-0"
                        enter-to-class="opacity-100"
                        leave-active-class="transition duration-200"
                        leave-from-class="opacity-100"
                        leave-to-class="opacity-0"
                    >
                        <span v-if="prefsSaved" class="text-xs text-green-400">Saved</span>
                    </Transition>
                    <button
                        class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer"
                        :class="prefsSaving
                            ? 'bg-white/[0.04] text-gray-500'
                            : 'bg-blue-500/15 text-blue-400 hover:bg-blue-500/25'"
                        :disabled="prefsSaving"
                        @click="savePreferences"
                    >{{ prefsSaving ? 'Saving...' : 'Save' }}</button>
                </div>
            </div>
            <p v-if="prefsError" class="mb-4 rounded-md border border-red-500/25 bg-red-500/[0.06] px-3 py-2 text-xs text-red-300 flex items-center gap-2">
                <Icon name="lucide:alert-circle" class="text-sm flex-shrink-0" />
                {{ prefsError }}
            </p>
            <p class="text-xs text-gray-600 mb-4">Choose which tab opens by default when navigating to an entity page without a specific tab.</p>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
                <div v-for="(opts, key) in defaultTabOptions" :key="key">
                    <div class="text-xs text-gray-500 mb-1.5">{{ defaultTabLabels[key] }}</div>
                    <SelectMenu
                        class="w-full [&>button]:w-full [&>button]:justify-between"
                        :model-value="defaultTabs[key] ?? ''"
                        :options="opts"
                        @update:model-value="defaultTabs[key] = $event"
                    />
                </div>
            </div>
        </div>

        <!-- Time zone -->
        <div class="glass-panel p-5">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">Time Zone</h2>
            <p class="text-xs text-gray-600 mb-4">
                Which clock date and time pickers are drawn in. Saved to this browser instantly — it changes
                only what you see, never the instant a picker stores.
            </p>
            <div class="flex flex-col sm:flex-row gap-2">
                <button
                    v-for="z in zoneOptions" :key="z.value"
                    class="flex-1 text-left px-4 py-3 rounded-lg border transition-colors cursor-pointer"
                    :class="zone === z.value
                        ? 'bg-blue-500/10 border-blue-500/40 text-blue-300'
                        : 'bg-white/[0.02] border-white/[0.08] text-gray-400 hover:border-white/[0.15] hover:bg-blue-500/[0.04]'"
                    @click="zone = z.value"
                >
                    <div class="flex items-center gap-2 text-sm font-medium">
                        <Icon :name="zone === z.value ? 'lucide:circle-check' : 'lucide:circle'" class="text-base flex-shrink-0" />
                        {{ z.label }}
                    </div>
                    <div class="text-xs text-gray-600 mt-1 pl-6 truncate">{{ z.hint }}</div>
                </button>
            </div>
        </div>
    </div>
</template>
