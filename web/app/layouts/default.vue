<script setup lang="ts">
import { THEME_PRESETS } from '~/utils/themeData'
import { siteBackgroundCSS } from '#shared/utils/siteBackground'

const { currentBackground, viewerMode } = useSiteBackground()
const { CSS_VAR_MAP } = useTheme()
const { domainConfig, isDomainMode, isUnknownDomain } = useDomainConfig()

// Initialize global keyboard shortcuts
const { isOpen: searchOpen } = useSpotlightSearch()
const { showHelp } = useKeyboardShortcuts()

const siteName = computed(() => domainConfig.value?.siteName || 'EVE-KILL')

// A board with no banner image can still opt into the header block; the strip
// shows the site background instead of an image (see .content--bleed).
const showTransparentBanner = computed(() =>
    isDomainMode.value
    && !domainConfig.value?.theme?.bannerUrl
    && domainConfig.value?.theme?.transparentBanner === true)

// Either header variant already supplies the space the navbar needs, so the
// main content drops its usual 4.5rem top pad.
const hasHeaderBlock = computed(() =>
    (isDomainMode.value && !!domainConfig.value?.theme?.bannerUrl) || showTransparentBanner.value)

// Panel opacity, as a scale on the stylesheet's defaults. Clamped here rather
// than trusted: theme jsonb is stored unvalidated, and a bad value would
// otherwise reach calc() and blank the panels entirely.
const CONTENT_OPACITY_MIN = 20
const contentStyle = computed(() => {
    if (!isDomainMode.value) return undefined
    const raw = Number(domainConfig.value?.theme?.contentOpacity)
    if (!Number.isFinite(raw)) return undefined
    const percent = Math.min(100, Math.max(CONTENT_OPACITY_MIN, raw))
    if (percent === 100) return undefined
    return { '--content-alpha-scale': String(percent / 100) }
})

// Prevent search engine indexing on all subdomains
if (isDomainMode.value || isUnknownDomain.value) {
    useSeoMeta({ robots: 'noindex, nofollow' })
}

// Canonical URL — always points to https://eve-kill.com with a clean path.
// Strips query params, trailing slashes, and any [[tab]] segment so search engines
// consolidate /character/123, /character/123/kills, /character/123/stats etc. into
// one indexable URL (tabs are UI-only; server-side renders the same content).
const route = useRoute()
const canonicalUrl = computed(() => {
    let path = route.path.replace(/\/+$/, '') || '/'
    const tab = route.params.tab
    if (typeof tab === 'string' && tab.length > 0 && path.endsWith(`/${tab}`)) {
        path = path.slice(0, -(tab.length + 1)) || '/'
    }
    return `https://eve-kill.com${path}`
})
useHead({
    link: [{ rel: 'canonical', href: canonicalUrl }],
})

// Title template:
//   EVE-KILL default mode → `<page> - EVE-KILL.com`
//                           (homepage sets title='Home' → `Home - EVE-KILL.com`)
//   Custom domain mode    → `<page> - <siteName>` (or just `<siteName>` on home)
useHead({
    titleTemplate: (title?: string) => {
        if (isDomainMode.value || isUnknownDomain.value) {
            return title ? `${title} - ${siteName.value}` : siteName.value
        }
        return title ? `${title} - EVE-KILL.com` : 'EVE-KILL.com'
    },
    link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'icon', type: 'image/png', href: '/icon.png' },
        { rel: 'shortcut icon', href: '/icon.ico' },
        { rel: 'apple-touch-icon', href: '/icon.png' },
        { rel: 'alternate', type: 'application/rss+xml', title: 'EVE-KILL — Latest Kills', href: '/feed.xml' },
        { rel: 'manifest', href: '/manifest.json' },
    ],
})

// Inject domain default theme as a pre-paint script during SSR.
// Uses setProperty() on documentElement for inline-style specificity (beats stylesheets).
// Only applies when user has no personal theme cookie (their cookie takes priority).
const domainThemeScript = computed(() => {
    if (!isDomainMode.value || !domainConfig.value?.theme) return ''
    const t = domainConfig.value.theme
    if (!t.defaultThemePreset && !t.defaultThemeOverrides) return ''

    // Resolve the domain's default theme overrides
    let overrides: Record<string, string> = {}
    if (t.defaultThemePreset) {
        const preset = THEME_PRESETS.find(p => p.id === t.defaultThemePreset)
        if (preset) overrides = { ...preset.overrides } as Record<string, string>
    }
    if (t.defaultThemeOverrides) {
        Object.assign(overrides, t.defaultThemeOverrides)
    }

    // Build a map of CSS var -> value to apply
    const entries: string[] = []
    for (const [key, cssVar] of Object.entries(CSS_VAR_MAP)) {
        if (overrides[key]) entries.push(`${JSON.stringify(cssVar)}:${JSON.stringify(overrides[key])}`)
    }
    if (entries.length === 0) return ''

    // Script: skip if user has their own ek_theme cookie, otherwise apply via setProperty
    return `(function(){try{if(document.cookie.match(/ek_theme=([^;]+)/))return;var d={${entries.join(',')}};var r=document.documentElement;for(var k in d)r.style.setProperty(k,d[k])}catch(e){}})()`
})

// Inject the first domain background as a pre-paint background-image override.
// Same priority rule as the theme script: if the visitor has their own
// siteBackground cookie that takes precedence, otherwise we point the html
// element at the first approved domain background to avoid a flash of the
// default bg2.webp on first paint.
const domainBackgroundScript = computed(() => {
    if (!isDomainMode.value) return ''
    const first = domainConfig.value?.backgrounds?.[0]
    if (!first) return ''
    return `(function(){try{if(document.cookie.match(/siteBackground=([^;]+)/))return;document.documentElement.style.setProperty('--site-bg',${JSON.stringify(siteBackgroundCSS(first))})}catch(e){}})()`
})

useHead({
    script: computed(() => {
        const scripts: { innerHTML: string; tagPosition: 'head' }[] = []
        if (domainThemeScript.value) scripts.push({ innerHTML: domainThemeScript.value, tagPosition: 'head' })
        if (domainBackgroundScript.value) scripts.push({ innerHTML: domainBackgroundScript.value, tagPosition: 'head' })
        return scripts
    }),
})

useSeoMeta({
    description: computed(() => domainConfig.value?.siteDescription || 'EVE-KILL is a community-driven killboard for EVE Online — real-time combat data, killmail tracking, ship fitting analysis, and battle reports for New Eden.'),
    ogUrl: canonicalUrl,
    ogDescription: computed(() => domainConfig.value?.siteDescription || 'Community-driven killboard for EVE Online — real-time kills, battle reports, and player statistics.'),
    ogType: 'website',
    ogSiteName: siteName,
    ogImage: 'https://eve-kill.com/og-default.png',
    // Twitter / X card defaults
    twitterCard: 'summary',
    twitterSite: '@eve_kill',
    twitterTitle: computed(() => `${siteName.value} — EVE Online Killboard`),
    twitterDescription: computed(() => domainConfig.value?.siteDescription || 'Community-driven killboard for EVE Online — real-time kills, battle reports, and player statistics.'),
    twitterImage: 'https://eve-kill.com/og-default.png',
})
</script>

<template>
    <div class="min-h-screen relative">
        <!-- Unknown subdomain — show landing page instead of site content -->
        <template v-if="isUnknownDomain">
            <!-- Hidden slot to suppress NuxtPage warning -->
            <div class="hidden"><slot /></div>
            <div class="flex items-center justify-center min-h-screen px-4">
                <div class="text-center max-w-md rounded-xl bg-black/70 backdrop-blur-xl border border-white/[0.08] p-8 shadow-2xl">
                    <div class="mb-6 text-gray-600">
                        <Icon name="lucide:globe" class="text-5xl" />
                    </div>
                    <h1 class="text-2xl font-bold text-white mb-3">No Killboard Found</h1>
                    <p class="text-gray-400 mb-6">
                        There is no killboard configured at this address. Want your own personalized EVE Online killboard?
                    </p>
                    <a
                        href="https://eve-kill.com"
                        class="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-blue-500 text-white font-medium text-sm hover:bg-blue-600 transition-colors"
                    >
                        Visit EVE-KILL
                        <Icon name="lucide:arrow-right" class="text-base" />
                    </a>
                    <p class="text-xs text-gray-600 mt-4">
                        Log in at <a href="https://eve-kill.com" class="text-blue-400/70 hover:text-blue-400">eve-kill.com</a>, contribute your killmails, and create your own custom killboard from your settings.
                    </p>
                </div>
            </div>
        </template>

        <!-- Normal site content -->
        <template v-else>
            <!-- Powered by EVE-KILL badge, grown into a killboard switcher.
                 Shown on every host now, not just custom domains: the E is
                 also where you reach the boards that track you. -->
            <BoardSwitcher />

            <div class="transition-opacity duration-600" :class="viewerMode ? 'opacity-0 pointer-events-none' : ''">
                <Navbar />
            </div>

            <LazySpotlightSearch v-if="searchOpen" />
            <LazyKeyboardShortcutsModal v-if="showHelp" />
            <Toaster />
            <KillWatcherToasts />
            <AnnouncementModal />
            <NewsTicker />

            <!-- Background controls — always visible -->
            <BackgroundControls />

            <!-- Persistent backdrop-blur layer. Standalone fixed div so the
                 blur paints continuously and doesn't flash/redraw across
                 navigations the way a pseudo-element on a transitioning
                 parent did. -->
            <div
                class="content-blur transition-opacity duration-600"
                :class="viewerMode ? 'opacity-0' : ''"
                aria-hidden="true"
            />

            <div class="content mx-auto transition-opacity duration-600"
                :style="contentStyle"
                :class="[
                    viewerMode ? '!bg-transparent opacity-0 pointer-events-none' : '',
                    showTransparentBanner ? 'content--bleed' : '',
                ]">
                <!-- Domain banner (custom subdomain only) -->
                <DomainBanner v-if="isDomainMode && domainConfig?.theme?.bannerUrl" :config="domainConfig" />
                <DomainBanner v-else-if="showTransparentBanner" :config="domainConfig!" transparent />
                <div class="inner-content" :class="viewerMode ? '!bg-transparent' : ''">
                    <main class="main-content" :class="hasHeaderBlock ? '!pt-6' : ''">
                        <AnnouncementBanner />
                        <slot />
                    </main>
                    <Footer />
                </div>
            </div>
        </template>
    </div>
</template>
