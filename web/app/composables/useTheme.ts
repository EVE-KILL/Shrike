import { DEFAULT_THEME, CSS_VAR_MAP, THEME_PRESETS, type ThemeConfig } from '~/utils/themeData'

/** Resolve the domain's default theme overrides from its config. */
function getDomainThemeDefaults(): Partial<ThemeConfig> {
    // Safe to call — returns empty if not in domain mode
    const { domainConfig } = useDomainConfig()
    const dc = domainConfig.value
    if (!dc?.theme) return {}

    // If a preset is set, use its overrides as the base
    const presetId = dc.theme.defaultThemePreset
    let base: Partial<ThemeConfig> = {}
    if (presetId) {
        const preset = THEME_PRESETS.find(p => p.id === presetId)
        if (preset) base = { ...preset.overrides }
    }

    // Layer any explicit overrides on top
    const custom = dc.theme.defaultThemeOverrides
    if (custom && typeof custom === 'object') {
        Object.assign(base, custom)
    }

    return base
}

function detectPresetId(overrides: Partial<ThemeConfig>): string {
    for (const preset of THEME_PRESETS) {
        if (preset.id === 'default' && Object.keys(overrides).length === 0) return 'default'
        if (preset.id !== 'default') {
            const keys = Object.keys(preset.overrides) as (keyof ThemeConfig)[]
            if (keys.length > 0 && keys.every(k => overrides[k] === preset.overrides[k])) {
                return preset.id
            }
        }
    }
    return Object.keys(overrides).length === 0 ? 'default' : 'custom'
}

export function useTheme() {
    const cookie = useCookie<Partial<ThemeConfig> | null>('ek_theme', {
        maxAge: 365 * 86400,
        default: () => null,
    })

    // SSR always renders with the domain default theme (or base default on
    // the main site). This makes the rendered HTML identical for all users
    // of a given host, which is required for Cloudflare edge caching — CF
    // caches one HTML version per URL+host, and a per-user theme baked into
    // the Nuxt payload would poison the cache for everyone else.
    //
    // The user's personal theme is applied in two steps:
    //   1. Inline <head> script (nuxt.config.ts app.head.script) reads the
    //      ek_theme cookie and calls setProperty() on <html> BEFORE the body
    //      is parsed — no flash, no pop-in.
    //   2. Client-side hydration below syncs Vue state from the cookie so
    //      the reactive theme system (settings panel, live preview) works.
    const domainDefaults = getDomainThemeDefaults()

    // Active theme overrides (only non-default values)
    const themeOverrides = useState<Partial<ThemeConfig>>(
        'themeOverrides',
        () => domainDefaults,
    )

    // Active preset ID (tracked separately for UI)
    const activePresetId = useState<string>(
        'themePresetId',
        () => detectPresetId(domainDefaults),
    )

    // Client: sync from cookie so Vue state matches what the inline <head>
    // script already applied. The SSR payload carries the domain default
    // for edge-cacheability; this re-initializes from the real cookie.
    if (import.meta.client) {
        const fromCookie = cookie.value
        if (fromCookie && Object.keys(fromCookie).length > 0) {
            themeOverrides.value = fromCookie
            activePresetId.value = detectPresetId(fromCookie)
        }
    }

    // Compute the full resolved theme
    const resolvedTheme = computed<ThemeConfig>(() => ({
        ...DEFAULT_THEME,
        ...themeOverrides.value,
    }))

    // Apply theme overrides to DOM
    const applyTheme = (overrides: Partial<ThemeConfig>) => {
        if (import.meta.server) return
        const root = document.documentElement
        for (const [key, cssVar] of Object.entries(CSS_VAR_MAP)) {
            const k = key as keyof ThemeConfig
            const value = overrides[k]
            if (value) {
                root.style.setProperty(cssVar, value)
            } else {
                // Reset to stylesheet default
                root.style.removeProperty(cssVar)
            }
        }
    }

    // Watch for changes and sync. CRITICAL: only register on the client.
    // `useTheme()` is called from `layouts/default.vue` on every SSR render,
    // so an unguarded `watch()` here creates a fresh Vue effect subscribed
    // to `themeOverrides`'s Dep on every request. Vue 3 SSR does not
    // aggressively dispose per-request effect scopes, so those subscribers
    // pile up inside the ref's Dep and retain the request's entire
    // component tree across GC cycles. Guarding with `import.meta.client`
    // is safe because the side effects (cookie write, DOM style mutation)
    // only make sense in the browser anyway.
    if (import.meta.client) {
        watch(themeOverrides, (overrides) => {
            // Only store non-default values in cookie
            const minimal: Partial<ThemeConfig> = {}
            for (const [key, value] of Object.entries(overrides)) {
                const k = key as keyof ThemeConfig
                if (value && value !== DEFAULT_THEME[k]) {
                    minimal[k] = value
                }
            }
            cookie.value = Object.keys(minimal).length > 0 ? minimal : null
            applyTheme(overrides)
        }, { deep: true })

        // Apply on mount (cookie is already read by head script for initial paint,
        // but we also apply here for client-side nav into themed pages)
        onMounted(() => {
            if (Object.keys(themeOverrides.value).length > 0) {
                applyTheme(themeOverrides.value)
            }
        })
    }

    const setPreset = (presetId: string) => {
        const preset = THEME_PRESETS.find(p => p.id === presetId)
        if (preset) {
            activePresetId.value = presetId
            themeOverrides.value = { ...preset.overrides }
            useAnalytics().track('theme.change', { preset: presetId })
        }
    }

    const setColor = (key: keyof ThemeConfig, value: string) => {
        activePresetId.value = 'custom'
        themeOverrides.value = { ...themeOverrides.value, [key]: value }
    }

    const resetToDefault = () => {
        activePresetId.value = 'default'
        themeOverrides.value = {}
    }

    return {
        themeOverrides,
        resolvedTheme,
        activePresetId: readonly(activePresetId),
        setPreset,
        setColor,
        resetToDefault,
        DEFAULT_THEME,
        CSS_VAR_MAP,
    }
}
