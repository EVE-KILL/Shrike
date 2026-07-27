<script setup lang="ts">
// Settings → Appearance section. Extracted verbatim from pages/settings/[[tab]].vue.
// Note: hydrating themeOverrides from the user's backend-saved theme stays in
// the settings page (it must run on every settings load, not just when this
// tab is opened). useTheme() state is shared useState, so both see the same refs.
// ── Appearance ─────────────────────────────────────────────────────────────
const { resolvedTheme, themeOverrides, activePresetId, setPreset, setColor, resetToDefault, DEFAULT_THEME, CSS_VAR_MAP } = useTheme()

// Color picker groups for the UI
const colorGroups = [
    {
        label: 'Brand & Accent',
        colors: [
            { key: 'brandPrimary' as const, label: 'Primary' },
            { key: 'brandPrimaryHover' as const, label: 'Primary Hover' },
            { key: 'brandSecondary' as const, label: 'Secondary' },
            { key: 'brandAccent' as const, label: 'Accent' },
        ],
    },
    {
        label: 'Backgrounds',
        colors: [
            { key: 'bgPrimary' as const, label: 'Primary BG' },
            { key: 'bgTertiary' as const, label: 'Tertiary BG' },
        ],
    },
    {
        label: 'Text',
        colors: [
            { key: 'textPrimary' as const, label: 'Primary' },
            { key: 'textSecondary' as const, label: 'Secondary' },
            { key: 'textTertiary' as const, label: 'Muted' },
        ],
    },
    {
        label: 'Borders',
        colors: [
            { key: 'borderMedium' as const, label: 'Border' },
            { key: 'borderFocus' as const, label: 'Focus Ring' },
        ],
    },
    {
        label: 'Status',
        colors: [
            { key: 'colorSuccess' as const, label: 'Success' },
            { key: 'colorWarning' as const, label: 'Warning' },
            { key: 'colorError' as const, label: 'Error / Danger' },
            { key: 'colorInfo' as const, label: 'Info' },
        ],
    },
    {
        label: 'Kill / Loss',
        colors: [
            { key: 'lossBg' as const, label: 'Loss Background' },
            { key: 'lossHover' as const, label: 'Loss Hover' },
            { key: 'lossBorder' as const, label: 'Loss Border' },
        ],
    },
    {
        label: 'EVE Security Status',
        colors: [
            { key: 'colorHighsec' as const, label: 'Highsec' },
            { key: 'colorLowsec' as const, label: 'Lowsec' },
            { key: 'colorNullsec' as const, label: 'Nullsec' },
        ],
    },
    {
        label: 'Scrollbar',
        colors: [
            { key: 'scrollbarThumb' as const, label: 'Scrollbar Thumb' },
        ],
    },
    {
        label: 'Content',
        colors: [
            { key: 'iskColor' as const, label: 'ISK Values' },
            { key: 'npcColor' as const, label: 'NPC Names' },
        ],
    },
    {
        label: 'Selection',
        colors: [
            { key: 'selectionBg' as const, label: 'Selection Background' },
            { key: 'selectionText' as const, label: 'Selection Text' },
        ],
    },
]

const themeSaving = ref(false)
const themeSaved = ref(false)
const themeError = ref('')
let themeSavedTimer: ReturnType<typeof setTimeout> | null = null

/**
 * Whether the live preview differs from what is stored on the account.
 *
 * Theme edits apply instantly to local state, so without this there is nothing
 * on screen distinguishing "saved" from "looks right but will be gone on your
 * next device". Compared against the snapshot taken at mount and refreshed
 * after each successful save.
 */
const normalise = (overrides: Record<string, string>) => JSON.stringify(
    Object.entries(overrides)
        .filter(([k, v]) => v && v !== DEFAULT_THEME[k as keyof typeof DEFAULT_THEME])
        .sort(([a], [b]) => a.localeCompare(b)),
)
const savedSnapshot = ref(normalise(themeOverrides.value))
const isDirty = computed(() => normalise(themeOverrides.value) !== savedSnapshot.value)

const saveTheme = async () => {
    themeSaving.value = true
    themeError.value = ''
    try {
        // Only save non-default overrides
        const overrides: Record<string, string> = {}
        for (const [key, value] of Object.entries(themeOverrides.value)) {
            if (value && value !== DEFAULT_THEME[key as keyof typeof DEFAULT_THEME]) {
                overrides[key] = value
            }
        }
        await apiFetch('/api/user/theme', {
            method: 'PUT',
            body: { theme: overrides },
        })
        savedSnapshot.value = normalise(themeOverrides.value)
        themeSaved.value = true
        if (themeSavedTimer) clearTimeout(themeSavedTimer)
        themeSavedTimer = setTimeout(() => { themeSaved.value = false }, 2000)
    } catch (err: any) {
        // A theme edit previews instantly from local state, so a failed PUT is
        // invisible without this — the page still looks right, and the change
        // silently vanishes on the next device or after clearing the cookie.
        themeError.value = extractFetchError(err, 'Could not save your theme')
    } finally {
        themeSaving.value = false
    }
}

onUnmounted(() => { if (themeSavedTimer) clearTimeout(themeSavedTimer) })

// Helper: extract hex from resolved value (for color inputs that need #hex)
const toHex = (val: string): string => {
    if (val.startsWith('#')) return val.length <= 7 ? val : val.slice(0, 7)
    return val
}
</script>

<template>
    <div class="space-y-4">
        <!-- Save bar.
             The Save button used to live in the Theme Presets header, while the
             32 colour pickers it applies to sit in the panel below — so editing
             a colour meant scrolling back up to a button you could no longer
             see. Sticky, and it says whether there is anything to save. -->
        <div class="sticky top-2 z-20 glass-panel px-4 py-2.5 flex items-center gap-3">
            <Icon name="lucide:palette" class="text-base text-blue-400 flex-shrink-0" />
            <div class="min-w-0 flex-1">
                <div class="text-sm font-medium text-white">Appearance</div>
                <div class="text-fine truncate" :class="isDirty ? 'text-yellow-400' : 'text-gray-600'">
                    {{ isDirty ? 'Unsaved changes — previewing locally' : 'Previewing your saved theme' }}
                </div>
            </div>
            <Transition
                enter-active-class="transition duration-200"
                enter-from-class="opacity-0"
                leave-active-class="transition duration-200"
                leave-to-class="opacity-0"
            >
                <span v-if="themeSaved" class="text-xs text-green-400 flex-shrink-0">Saved</span>
            </Transition>
            <button
                class="px-3 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer flex-shrink-0 disabled:cursor-not-allowed"
                :class="themeSaving || !isDirty
                    ? 'bg-white/[0.04] text-gray-500'
                    : 'bg-blue-500/15 text-blue-400 hover:bg-blue-500/25'"
                :disabled="themeSaving || !isDirty"
                @click="saveTheme"
            >
                <Icon v-if="themeSaving" name="lucide:loader-2" class="text-xs animate-spin mr-1 inline" />
                {{ themeSaving ? 'Saving...' : 'Save Theme' }}
            </button>
        </div>

        <p v-if="themeError" class="rounded-md border border-red-500/25 bg-red-500/[0.06] px-3 py-2 text-xs text-red-300 flex items-center gap-2">
            <Icon name="lucide:alert-circle" class="text-sm flex-shrink-0" />
            {{ themeError }}
        </p>

        <!-- Preset themes -->
        <div class="glass-panel p-5">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">Theme Presets</h2>
            <p class="text-xs text-gray-600 mb-4">Pick a preset or customize individual colors below. Changes preview instantly — click Save to persist across sessions.</p>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
                <button
                    v-for="preset in THEME_PRESETS" :key="preset.id"
                    class="relative flex flex-col items-center gap-2 p-3 rounded-lg border transition-all cursor-pointer"
                    :class="activePresetId === preset.id
                        ? 'border-blue-500/50 bg-blue-500/10'
                        : 'border-white/[0.08] bg-white/[0.02] hover:bg-blue-500/[0.04] hover:border-white/[0.15]'"
                    @click="setPreset(preset.id)"
                >
                    <!-- Color swatches preview -->
                    <div class="flex gap-1">
                        <span
                            class="w-5 h-5 rounded-full border border-white/10"
                            :style="{ backgroundColor: preset.overrides.brandPrimary || DEFAULT_THEME.brandPrimary }"
                        ></span>
                        <span
                            class="w-5 h-5 rounded-full border border-white/10"
                            :style="{ backgroundColor: preset.overrides.brandAccent || DEFAULT_THEME.brandAccent }"
                        ></span>
                        <span
                            class="w-5 h-5 rounded-full border border-white/10"
                            :style="{ backgroundColor: preset.overrides.brandSecondary || DEFAULT_THEME.brandSecondary }"
                        ></span>
                    </div>
                    <div class="text-center">
                        <div class="text-xs font-medium text-white">{{ preset.name }}</div>
                        <div class="text-fine text-gray-500 leading-tight">{{ preset.description }}</div>
                    </div>
                    <!-- Active indicator -->
                    <span
                        v-if="activePresetId === preset.id"
                        class="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-blue-400"
                    ></span>
                </button>
            </div>
        </div>

        <!-- Custom color pickers -->
        <div class="glass-panel p-5">
            <div class="flex items-center justify-between mb-4">
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Custom Colors</h2>
                <button
                    v-if="Object.keys(themeOverrides).length > 0"
                    class="px-2 py-1 rounded text-xs text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors cursor-pointer"
                    @click="resetToDefault"
                >Reset All</button>
            </div>

            <div class="space-y-5">
                <div v-for="group in colorGroups" :key="group.label">
                    <h3 class="text-xs font-medium text-gray-400 mb-2">{{ group.label }}</h3>
                    <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                        <div v-for="c in group.colors" :key="c.key" class="flex items-center gap-2">
                            <input
                                type="color"
                                :value="toHex(resolvedTheme[c.key])"
                                class="w-8 h-8 rounded border border-white/[0.08] bg-transparent cursor-pointer flex-shrink-0"
                                @input="setColor(c.key, ($event.target as HTMLInputElement).value)"
                            >
                            <div class="flex-1 min-w-0">
                                <div class="text-xs text-gray-300 truncate">{{ c.label }}</div>
                                <div class="text-fine text-gray-600 font-mono truncate">{{ resolvedTheme[c.key] }}</div>
                            </div>
                            <!-- Reset individual color -->
                            <button
                                v-if="themeOverrides[c.key] && themeOverrides[c.key] !== DEFAULT_THEME[c.key]"
                                class="p-1 text-gray-600 hover:text-blue-400 transition-colors flex-shrink-0 cursor-pointer"
                                v-tooltip="'Reset to default'"
                                @click="setColor(c.key, DEFAULT_THEME[c.key])"
                            >
                                <Icon name="lucide:rotate-ccw" class="text-xs" />
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Live preview -->
        <div class="glass-panel p-5">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">Preview</h2>
            <div class="space-y-3">
                <!-- Buttons -->
                <div class="flex flex-wrap gap-2">
                    <button class="px-3 py-1.5 rounded-md text-xs font-medium text-white transition-colors" :style="{ backgroundColor: resolvedTheme.brandPrimary }">Primary Button</button>
                    <button class="px-3 py-1.5 rounded-md text-xs font-medium text-white transition-colors" :style="{ backgroundColor: resolvedTheme.brandSecondary }">Secondary</button>
                    <button class="px-3 py-1.5 rounded-md text-xs font-medium text-white transition-colors" :style="{ backgroundColor: resolvedTheme.brandAccent }">Accent</button>
                </div>
                <!-- Status badges -->
                <div class="flex flex-wrap gap-2">
                    <span class="px-2 py-0.5 rounded text-xs" :style="{ backgroundColor: resolvedTheme.colorSuccess + '20', color: resolvedTheme.colorSuccess }">Success</span>
                    <span class="px-2 py-0.5 rounded text-xs" :style="{ backgroundColor: resolvedTheme.colorWarning + '20', color: resolvedTheme.colorWarning }">Warning</span>
                    <span class="px-2 py-0.5 rounded text-xs" :style="{ backgroundColor: resolvedTheme.colorError + '20', color: resolvedTheme.colorError }">Error</span>
                    <span class="px-2 py-0.5 rounded text-xs" :style="{ backgroundColor: resolvedTheme.colorInfo + '20', color: resolvedTheme.colorInfo }">Info</span>
                </div>
                <!-- Loss row example -->
                <div class="flex items-center gap-3 px-3 py-2 rounded-md border" :style="{ backgroundColor: resolvedTheme.lossBg, borderColor: resolvedTheme.lossBorder }">
                    <span class="text-xs" :style="{ color: resolvedTheme.textPrimary }">Loss Row Preview</span>
                    <span class="text-fine" :style="{ color: resolvedTheme.textSecondary }">-1.5B ISK</span>
                </div>
                <!-- Security status -->
                <div class="flex gap-3">
                    <span class="text-xs font-medium" :style="{ color: resolvedTheme.colorHighsec }">Highsec 1.0</span>
                    <span class="text-xs font-medium" :style="{ color: resolvedTheme.colorLowsec }">Lowsec 0.4</span>
                    <span class="text-xs font-medium" :style="{ color: resolvedTheme.colorNullsec }">Nullsec -1.0</span>
                </div>
                <!-- Text hierarchy -->
                <div class="space-y-1">
                    <div class="text-sm" :style="{ color: resolvedTheme.textPrimary }">Primary text — headings and important content</div>
                    <div class="text-xs" :style="{ color: resolvedTheme.textSecondary }">Secondary text — labels and descriptions</div>
                    <div class="text-fine" :style="{ color: resolvedTheme.textTertiary }">Muted text — timestamps and metadata</div>
                </div>
            </div>
        </div>
    </div>
</template>
