/**
 * Static theme data: the ThemeConfig shape, default values, CSS variable
 * mapping, and all faction presets. Pure data — the logic lives in
 * composables/useTheme.ts.
 */
export interface ThemeConfig {
    // Brand / accent
    brandPrimary: string
    brandPrimaryHover: string
    brandSecondary: string
    brandAccent: string

    // Backgrounds
    bgPrimary: string
    bgSecondary: string
    bgTertiary: string
    bgHover: string

    // Text
    textPrimary: string
    textSecondary: string
    textTertiary: string

    // Borders
    borderLight: string
    borderMedium: string
    borderFocus: string

    // Surfaces
    surfaceAlpha: string
    surfaceHover: string

    // Status
    colorSuccess: string
    colorWarning: string
    colorError: string
    colorInfo: string

    // Kill/Loss
    lossBg: string
    lossHover: string
    lossBorder: string

    // EVE security
    colorHighsec: string
    colorLowsec: string
    colorNullsec: string

    // Scrollbar
    scrollbarThumb: string

    // Content-specific
    iskColor: string
    npcColor: string
    selectionBg: string
    selectionText: string
}

// Default theme — matches variables.css
export const DEFAULT_THEME: ThemeConfig = {
    brandPrimary: '#3b82f6',
    brandPrimaryHover: '#2563eb',
    brandSecondary: '#10b981',
    brandAccent: '#00aaff',

    bgPrimary: '#111827',
    bgSecondary: 'rgba(26, 26, 26, 0.8)',
    bgTertiary: '#1f2937',
    bgHover: 'rgba(45, 45, 45, 0.6)',

    textPrimary: '#ffffff',
    textSecondary: '#9ca3af',
    textTertiary: '#6b7280',

    borderLight: 'rgba(75, 85, 99, 0.2)',
    borderMedium: '#374151',
    borderFocus: '#3b82f6',

    surfaceAlpha: 'rgba(26, 26, 26, 0.5)',
    surfaceHover: '#374151',

    colorSuccess: '#10b981',
    colorWarning: '#f59e0b',
    colorError: '#ef4444',
    colorInfo: '#3b82f6',

    lossBg: 'rgba(80, 0, 0, 0.6)',
    lossHover: 'rgba(90, 0, 0, 0.65)',
    lossBorder: 'rgba(139, 46, 46, 0.8)',

    colorHighsec: '#2f75fe',
    colorLowsec: '#f9a825',
    colorNullsec: '#d32f2f',

    scrollbarThumb: '#3b82f6',

    iskColor: '#4ade80',
    npcColor: '#fcd34d',
    selectionBg: '#3b82f6',
    selectionText: '#ffffff',
}

// Map from ThemeConfig keys to CSS variable names
export const CSS_VAR_MAP: Record<keyof ThemeConfig, string> = {
    brandPrimary: '--color-brand-primary',
    brandPrimaryHover: '--color-brand-primary-hover',
    brandSecondary: '--color-brand-secondary',
    brandAccent: '--color-brand-accent',

    bgPrimary: '--color-bg-primary',
    bgSecondary: '--color-bg-secondary',
    bgTertiary: '--color-bg-tertiary',
    bgHover: '--color-bg-hover',

    textPrimary: '--color-text-primary',
    textSecondary: '--color-text-secondary',
    textTertiary: '--color-text-tertiary',

    borderLight: '--color-border-light',
    borderMedium: '--color-border-medium',
    borderFocus: '--color-border-focus',

    surfaceAlpha: '--color-surface-alpha',
    surfaceHover: '--color-surface-hover',

    colorSuccess: '--color-success',
    colorWarning: '--color-warning',
    colorError: '--color-error',
    colorInfo: '--color-info',

    lossBg: '--color-loss-bg',
    lossHover: '--color-loss-hover',
    lossBorder: '--color-loss-border',

    colorHighsec: '--color-highsec',
    colorLowsec: '--color-lowsec',
    colorNullsec: '--color-nullsec',

    scrollbarThumb: '--scrollbar-thumb-color',

    iskColor: '--color-isk-value',
    npcColor: '--color-npc-text',
    selectionBg: '--color-selection-bg',
    selectionText: '--color-selection-text',
}

// Preset themes
export interface ThemePreset {
    id: string
    name: string
    description: string
    overrides: Partial<ThemeConfig>
}

export const THEME_PRESETS: ThemePreset[] = [
    {
        id: 'default',
        name: 'Default',
        description: 'Classic EVE-KILL blue',
        overrides: {},
    },
    {
        id: 'amarr',
        name: 'Amarr Gold',
        description: 'Golden empire aesthetic',
        overrides: {
            brandPrimary: '#d4a017',
            brandPrimaryHover: '#b8860b',
            brandAccent: '#ffd700',
            borderFocus: '#d4a017',
            colorInfo: '#d4a017',
            scrollbarThumb: '#d4a017',
            selectionBg: '#d4a017',
            iskColor: '#fbbf24',
        },
    },
    {
        id: 'caldari',
        name: 'Caldari Steel',
        description: 'Cool corporate grey-blue',
        overrides: {
            brandPrimary: '#5b8fa8',
            brandPrimaryHover: '#4a7a91',
            brandAccent: '#7ec8e3',
            borderFocus: '#5b8fa8',
            colorInfo: '#5b8fa8',
            scrollbarThumb: '#5b8fa8',
            selectionBg: '#5b8fa8',
        },
    },
    {
        id: 'gallente',
        name: 'Gallente Green',
        description: 'Federation emerald tones',
        overrides: {
            brandPrimary: '#22c55e',
            brandPrimaryHover: '#16a34a',
            brandAccent: '#4ade80',
            brandSecondary: '#06b6d4',
            borderFocus: '#22c55e',
            colorInfo: '#22c55e',
            scrollbarThumb: '#22c55e',
            selectionBg: '#22c55e',
            iskColor: '#4ade80',
        },
    },
    {
        id: 'minmatar',
        name: 'Minmatar Rust',
        description: 'Tribal rust and orange',
        overrides: {
            brandPrimary: '#c2410c',
            brandPrimaryHover: '#9a3412',
            brandAccent: '#ea580c',
            brandSecondary: '#d97706',
            borderFocus: '#c2410c',
            colorInfo: '#c2410c',
            scrollbarThumb: '#c2410c',
            selectionBg: '#c2410c',
            iskColor: '#fb923c',
            npcColor: '#f97316',
        },
    },
    {
        id: 'blood-raider',
        name: 'Blood Raider',
        description: 'Deep crimson darkness',
        overrides: {
            brandPrimary: '#dc2626',
            brandPrimaryHover: '#b91c1c',
            brandAccent: '#f87171',
            brandSecondary: '#991b1b',
            borderFocus: '#dc2626',
            colorInfo: '#dc2626',
            scrollbarThumb: '#dc2626',
            selectionBg: '#dc2626',
            iskColor: '#f87171',
        },
    },
    {
        id: 'guristas',
        name: 'Guristas',
        description: 'Pirate grey and teal',
        overrides: {
            brandPrimary: '#14b8a6',
            brandPrimaryHover: '#0d9488',
            brandAccent: '#5eead4',
            brandSecondary: '#64748b',
            borderFocus: '#14b8a6',
            colorInfo: '#14b8a6',
            scrollbarThumb: '#14b8a6',
            selectionBg: '#14b8a6',
            iskColor: '#5eead4',
        },
    },
    {
        id: 'wormhole',
        name: 'Wormhole',
        description: 'Eerie purple void',
        overrides: {
            brandPrimary: '#a855f7',
            brandPrimaryHover: '#9333ea',
            brandAccent: '#c084fc',
            brandSecondary: '#7c3aed',
            borderFocus: '#a855f7',
            colorInfo: '#a855f7',
            scrollbarThumb: '#a855f7',
            selectionBg: '#a855f7',
            iskColor: '#c084fc',
        },
    },
    {
        id: 'sansha',
        name: 'Sansha\'s Nation',
        description: 'Sickly yellow corruption',
        overrides: {
            brandPrimary: '#ca8a04',
            brandPrimaryHover: '#a16207',
            brandAccent: '#facc15',
            brandSecondary: '#854d0e',
            borderFocus: '#ca8a04',
            colorInfo: '#ca8a04',
            scrollbarThumb: '#ca8a04',
            selectionBg: '#ca8a04',
            iskColor: '#fde047',
            npcColor: '#eab308',
        },
    },
    {
        id: 'serpentis',
        name: 'Serpentis',
        description: 'Toxic green syndicate',
        overrides: {
            brandPrimary: '#65a30d',
            brandPrimaryHover: '#4d7c0f',
            brandAccent: '#a3e635',
            brandSecondary: '#84cc16',
            borderFocus: '#65a30d',
            colorInfo: '#65a30d',
            scrollbarThumb: '#65a30d',
            selectionBg: '#65a30d',
            iskColor: '#a3e635',
        },
    },
    {
        id: 'angels',
        name: 'Angel Cartel',
        description: 'Hot chrome and crimson',
        overrides: {
            brandPrimary: '#e11d48',
            brandPrimaryHover: '#be123c',
            brandAccent: '#fb7185',
            brandSecondary: '#f43f5e',
            borderFocus: '#e11d48',
            colorInfo: '#e11d48',
            scrollbarThumb: '#e11d48',
            selectionBg: '#e11d48',
            iskColor: '#fda4af',
            npcColor: '#fb923c',
        },
    },
    {
        id: 'soe',
        name: 'Sisters of EVE',
        description: 'Clean white and red',
        overrides: {
            brandPrimary: '#e5e7eb',
            brandPrimaryHover: '#d1d5db',
            brandAccent: '#f9fafb',
            brandSecondary: '#ef4444',
            borderFocus: '#e5e7eb',
            colorInfo: '#e5e7eb',
            scrollbarThumb: '#9ca3af',
            selectionBg: '#e5e7eb',
            selectionText: '#111827',
            iskColor: '#86efac',
        },
    },
    {
        id: 'triglavian',
        name: 'Triglavian',
        description: 'Abyssal red geometry',
        overrides: {
            brandPrimary: '#b91c1c',
            brandPrimaryHover: '#991b1b',
            brandAccent: '#dc2626',
            brandSecondary: '#7f1d1d',
            colorError: '#f87171',
            borderFocus: '#b91c1c',
            colorInfo: '#b91c1c',
            scrollbarThumb: '#b91c1c',
            selectionBg: '#b91c1c',
            iskColor: '#fca5a5',
            lossBg: 'rgba(127, 29, 29, 0.5)',
            lossBorder: 'rgba(185, 28, 28, 0.6)',
        },
    },
    {
        id: 'edencom',
        name: 'EDENCOM',
        description: 'Electric blue defense',
        overrides: {
            brandPrimary: '#0ea5e9',
            brandPrimaryHover: '#0284c7',
            brandAccent: '#38bdf8',
            brandSecondary: '#06b6d4',
            borderFocus: '#0ea5e9',
            colorInfo: '#0ea5e9',
            scrollbarThumb: '#0ea5e9',
            selectionBg: '#0ea5e9',
            iskColor: '#67e8f9',
        },
    },
    {
        id: 'concord',
        name: 'CONCORD',
        description: 'Authoritative dark blue',
        overrides: {
            brandPrimary: '#1e40af',
            brandPrimaryHover: '#1e3a8a',
            brandAccent: '#3b82f6',
            brandSecondary: '#1d4ed8',
            borderFocus: '#1e40af',
            colorInfo: '#1e40af',
            scrollbarThumb: '#1e40af',
            selectionBg: '#1e40af',
        },
    },
    {
        id: 'ore',
        name: 'ORE',
        description: 'Industrial amber glow',
        overrides: {
            brandPrimary: '#d97706',
            brandPrimaryHover: '#b45309',
            brandAccent: '#fbbf24',
            brandSecondary: '#92400e',
            borderFocus: '#d97706',
            colorInfo: '#d97706',
            scrollbarThumb: '#d97706',
            selectionBg: '#d97706',
            iskColor: '#fcd34d',
            npcColor: '#fbbf24',
        },
    },
    {
        id: 'mordu',
        name: 'Mordu\'s Legion',
        description: 'Military olive drab',
        overrides: {
            brandPrimary: '#78716c',
            brandPrimaryHover: '#57534e',
            brandAccent: '#a8a29e',
            brandSecondary: '#65a30d',
            borderFocus: '#78716c',
            colorInfo: '#78716c',
            scrollbarThumb: '#78716c',
            selectionBg: '#78716c',
            iskColor: '#bef264',
        },
    },
    {
        id: 'thera',
        name: 'Thera',
        description: 'Signal Cartel cyan',
        overrides: {
            brandPrimary: '#06b6d4',
            brandPrimaryHover: '#0891b2',
            brandAccent: '#22d3ee',
            brandSecondary: '#14b8a6',
            borderFocus: '#06b6d4',
            colorInfo: '#06b6d4',
            scrollbarThumb: '#06b6d4',
            selectionBg: '#06b6d4',
            iskColor: '#67e8f9',
        },
    },
    {
        id: 'pochven',
        name: 'Pochven',
        description: 'Crimson triangle void',
        overrides: {
            brandPrimary: '#9f1239',
            brandPrimaryHover: '#881337',
            brandAccent: '#e11d48',
            brandSecondary: '#be123c',
            borderFocus: '#9f1239',
            colorInfo: '#9f1239',
            scrollbarThumb: '#9f1239',
            selectionBg: '#9f1239',
            iskColor: '#fda4af',
            colorHighsec: '#fb7185',
            colorLowsec: '#f43f5e',
            colorNullsec: '#9f1239',
        },
    },
    {
        id: 'jove',
        name: 'Jove Empire',
        description: 'Alien silver radiance',
        overrides: {
            brandPrimary: '#a1a1aa',
            brandPrimaryHover: '#71717a',
            brandAccent: '#e4e4e7',
            brandSecondary: '#a78bfa',
            borderFocus: '#a1a1aa',
            colorInfo: '#a1a1aa',
            scrollbarThumb: '#a1a1aa',
            selectionBg: '#a1a1aa',
            selectionText: '#18181b',
            iskColor: '#d4d4d8',
        },
    },
    {
        id: 'abyssal',
        name: 'Abyssal',
        description: 'Deep ocean abyss',
        overrides: {
            brandPrimary: '#7c3aed',
            brandPrimaryHover: '#6d28d9',
            brandAccent: '#a78bfa',
            brandSecondary: '#4f46e5',
            colorError: '#f472b6',
            borderFocus: '#7c3aed',
            colorInfo: '#7c3aed',
            scrollbarThumb: '#7c3aed',
            selectionBg: '#7c3aed',
            iskColor: '#c4b5fd',
            lossBg: 'rgba(76, 29, 149, 0.4)',
            lossBorder: 'rgba(124, 58, 237, 0.5)',
        },
    },
    {
        id: 'nyx',
        name: 'Nyx',
        description: 'Dark supercapital indigo',
        overrides: {
            brandPrimary: '#4f46e5',
            brandPrimaryHover: '#4338ca',
            brandAccent: '#818cf8',
            brandSecondary: '#6366f1',
            borderFocus: '#4f46e5',
            colorInfo: '#4f46e5',
            scrollbarThumb: '#4f46e5',
            selectionBg: '#4f46e5',
            iskColor: '#a5b4fc',
        },
    },
    {
        id: 'carbon',
        name: 'Carbon',
        description: 'Minimal monochrome',
        overrides: {
            brandPrimary: '#737373',
            brandPrimaryHover: '#525252',
            brandAccent: '#a3a3a3',
            brandSecondary: '#525252',
            borderFocus: '#737373',
            colorInfo: '#737373',
            scrollbarThumb: '#525252',
            selectionBg: '#737373',
            iskColor: '#d4d4d4',
            npcColor: '#a3a3a3',
            colorSuccess: '#a3a3a3',
            colorWarning: '#a3a3a3',
            colorError: '#a3a3a3',
        },
    },
    {
        id: 'drifter',
        name: 'Drifter',
        description: 'Mysterious ghostly blue',
        overrides: {
            brandPrimary: '#38bdf8',
            brandPrimaryHover: '#0ea5e9',
            brandAccent: '#7dd3fc',
            brandSecondary: '#0284c7',
            borderFocus: '#38bdf8',
            colorInfo: '#38bdf8',
            scrollbarThumb: '#38bdf8',
            selectionBg: '#38bdf8',
            iskColor: '#bae6fd',
            lossBg: 'rgba(3, 105, 161, 0.35)',
            lossBorder: 'rgba(56, 189, 248, 0.4)',
        },
    },
]
