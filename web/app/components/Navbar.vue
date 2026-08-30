<script lang="ts" setup>
import type { DropdownColumn, DropdownItem } from './Dropdown.vue'
import { DEFAULT_NAVBAR } from '~/composables/useDomainConfig'
import type { NavbarLink } from '~/composables/useDomainConfig'

const { openSearch } = useSpotlightSearch()
const { user, isAuthenticated, authHint, login, logout } = useAuth()
const { domainConfig, isDomainMode } = useDomainConfig()

// Scroll detection — for darker navbar background
const isScrolled = ref(false)
let _observer: IntersectionObserver | null = null
let _sentinel: HTMLElement | null = null

onMounted(() => {
    _sentinel = document.createElement('div')
    _sentinel.style.cssText = 'position:absolute;top:0;height:1px;width:100%;pointer-events:none;z-index:-1'
    document.body.prepend(_sentinel)
    _observer = new IntersectionObserver(
        (entries) => { isScrolled.value = !entries[0]?.isIntersecting },
        { threshold: 0, rootMargin: '-1px 0px 0px 0px' },
    )
    _observer.observe(_sentinel)
})

onUnmounted(() => {
    _observer?.disconnect()
    _sentinel?.remove()
})

// Mobile
const isMobileMenuOpen = ref(false)
const expandedMobile = ref<Record<string, boolean>>({})
const toggleMobileSection = (key: string) => {
    expandedMobile.value[key] = !expandedMobile.value[key]
}

// ── Navbar config ──
// Domain mode uses the domain's navbarLinks, default mode uses DEFAULT_NAVBAR
const navItems = computed<NavbarLink[]>(() => {
    if (isDomainMode.value && domainConfig.value?.navbarLinks?.length) {
        return domainConfig.value.navbarLinks as NavbarLink[]
    }
    return DEFAULT_NAVBAR
})

// Dropdown open states — one per top-level item
const dropdownStates = ref<Record<number, boolean>>({})

// Convert NavbarLink children to Dropdown component props
const toDropdownColumns = (link: NavbarLink): DropdownColumn[] | undefined => {
    if (!link.children?.length) return undefined
    // Multiple groups with labels = mega menu columns
    if (link.children.length > 1 || link.children[0]?.label) {
        return link.children.map(g => ({
            label: g.label || '',
            items: g.items.map(i => ({ name: i.label, to: i.href, icon: i.icon })),
        }))
    }
    return undefined
}

const toDropdownItems = (link: NavbarLink): DropdownItem[] | undefined => {
    if (!link.children?.length) return undefined
    // Single group with no label = simple dropdown
    if (link.children.length === 1 && !link.children[0]?.label) {
        return link.children[0]!.items.map(i => ({ name: i.label, to: i.href, icon: i.icon }))
    }
    return undefined
}

const hasChildren = (link: NavbarLink) => !!link.children?.length

// Background system
const bgOpen = ref(false)
const infoOpen = ref(false)
const userOpen = ref(false)
const loginOpen = ref(false)

interface PersonalWalletBalance {
    balance: string
    availableBalance: string
}

const personalWalletBalance = ref<number | null>(null)
const walletBalanceLoading = ref(false)
let walletBalanceRequest = 0

const loadPersonalWalletBalance = async () => {
    const characterId = user.value?.characterId
    if (!characterId) return

    const request = ++walletBalanceRequest
    walletBalanceLoading.value = true
    try {
        const data = await apiFetch<PersonalWalletBalance>('/api/user/wallet/balance')
        if (request === walletBalanceRequest && user.value?.characterId === characterId) {
            const balance = Number(data.availableBalance ?? data.balance)
            personalWalletBalance.value = Number.isFinite(balance) ? balance : null
        }
    } catch {
        if (request === walletBalanceRequest) personalWalletBalance.value = null
    } finally {
        if (request === walletBalanceRequest) walletBalanceLoading.value = false
    }
}

watch(
    () => user.value?.characterId,
    characterId => {
        walletBalanceRequest++
        personalWalletBalance.value = null
        walletBalanceLoading.value = false
        if (characterId) void loadPersonalWalletBalance()
    },
    { immediate: true },
)

watch(userOpen, open => {
    if (open) void loadPersonalWalletBalance()
})

watch(isMobileMenuOpen, open => {
    if (open && user.value) void loadPersonalWalletBalance()
})

const personalWalletDisplay = computed(() => {
    const value = personalWalletBalance.value
    if (value === null) return walletBalanceLoading.value ? '…' : '— ISK'

    const absolute = Math.abs(value)
    const units = [
        { value: 1e12, suffix: 't' },
        { value: 1e9, suffix: 'b' },
        { value: 1e6, suffix: 'm' },
        { value: 1e3, suffix: 'k' },
    ]
    const unit = units.find(candidate => absolute >= candidate.value)
    const amount = unit ? `${Math.round(value / unit.value)}${unit.suffix}` : Math.round(value).toLocaleString('en-US')
    return `${amount} ISK`
})

// Login preferences — persisted in cookie, but READ/WRITTEN CLIENT-SIDE ONLY.
// Using Nuxt's useCookie() here would call setCookie() on every SSR render
// (because of the `default:` fallback), which stamps a Set-Cookie header on
// every HTML response and makes Cloudflare BYPASS the edge cache for all
// pages. The login UI is gated behind <ClientOnly> anyway, so SSR never needs
// to see these values.
interface LoginPrefs { charKm: boolean; corpKm: boolean; delay: number }
const LOGIN_PREFS_COOKIE = 'ek_login_prefs'
const LOGIN_PREFS_DEFAULTS: LoginPrefs = { charKm: true, corpKm: true, delay: 0 }
const loginPrefs = ref<LoginPrefs>({ ...LOGIN_PREFS_DEFAULTS })

onMounted(() => {
    const match = document.cookie
        .split('; ')
        .find(c => c.startsWith(`${LOGIN_PREFS_COOKIE}=`))
    if (!match) return
    try {
        const parsed = JSON.parse(decodeURIComponent(match.slice(LOGIN_PREFS_COOKIE.length + 1))) as Partial<LoginPrefs>
        loginPrefs.value = {
            charKm: typeof parsed.charKm === 'boolean' ? parsed.charKm : LOGIN_PREFS_DEFAULTS.charKm,
            corpKm: typeof parsed.corpKm === 'boolean' ? parsed.corpKm : LOGIN_PREFS_DEFAULTS.corpKm,
            delay: typeof parsed.delay === 'number' ? parsed.delay : LOGIN_PREFS_DEFAULTS.delay,
        }
    } catch {
        // Malformed cookie — keep the defaults, user will re-set on next interaction.
    }
})

const writeLoginPrefsCookie = (prefs: LoginPrefs) => {
    const value = encodeURIComponent(JSON.stringify(prefs))
    const maxAge = 60 * 60 * 24 * 365
    document.cookie = `${LOGIN_PREFS_COOKIE}=${value}; Max-Age=${maxAge}; Path=/; SameSite=Lax`
}

const ALLOWED_DELAYS = [0, 1, 3, 6, 12, 24, 72]
const setLoginPref = (key: keyof LoginPrefs, value: boolean | number) => {
    loginPrefs.value = { ...loginPrefs.value, [key]: value } as LoginPrefs
    writeLoginPrefsCookie(loginPrefs.value)
}
const doLogin = () => {
    login({
        charKm: loginPrefs.value.charKm,
        corpKm: loginPrefs.value.corpKm,
        delay: loginPrefs.value.delay,
    })
}

const { currentBackground, isRedditBackground, redditMeta, setBackground, setRedditBackground } = useSiteBackground()
const bgLoading = ref(false)

const localBackgrounds = [
    { path: '/backgrounds/bg1.webp', thumb: '/backgrounds/thumbs/bg1.webp' },
    { path: '/backgrounds/bg2.webp', thumb: '/backgrounds/thumbs/bg2.webp' },
    { path: '/backgrounds/bg3.webp', thumb: '/backgrounds/thumbs/bg3.webp' },
    { path: '/backgrounds/bg4.webp', thumb: '/backgrounds/thumbs/bg4.webp' },
    { path: '/backgrounds/bg5.webp', thumb: '/backgrounds/thumbs/bg5.webp' },
    { path: '/backgrounds/bg6.webp', thumb: '/backgrounds/thumbs/bg6.webp' },
    { path: '/backgrounds/bg7.webp', thumb: '/backgrounds/thumbs/bg7.webp' },
    { path: '/backgrounds/bg8.webp', thumb: '/backgrounds/thumbs/bg8.webp' },
]

const pickReddit = async () => {
    bgLoading.value = true
    await setRedditBackground()
    bgLoading.value = false
}

const pickLocal = (path: string) => {
    setBackground(path)
    bgOpen.value = false
}

// Info items (always present on right side)
const infoItems: DropdownItem[] = [
    { name: 'FAQ', to: '/faq' },
    { name: 'API Docs', to: '/docs' },
    { name: 'MCP', to: '/mcp' },
    { name: 'Status', to: '/status' },
    { name: 'Wallet', to: '/wallet' },
    { name: 'About', to: '/about' },
    { name: 'Donate', to: '/donate' },
]

// Close everything on navigation
const router = useRouter()
router.afterEach(() => {
    isMobileMenuOpen.value = false
    dropdownStates.value = {}
    bgOpen.value = false
    infoOpen.value = false
    userOpen.value = false
    loginOpen.value = false
})

// Shared trigger button classes
const navBtn = 'flex items-center gap-2 px-3 py-2 rounded-md text-sm font-medium text-white/90 hover:bg-blue-500/10 hover:text-blue-400 transition-colors'
const navBtnDropdown = 'flex items-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium text-white/90 hover:bg-blue-500/10 hover:text-blue-400 transition-colors'
const iconBtn = 'flex items-center justify-center w-9 h-9 rounded-md text-white/60 hover:bg-blue-500/10 hover:text-blue-400 transition-colors'
</script>

<template>
    <!-- ==================== DESKTOP NAVBAR ==================== -->
    <nav class="hidden md:flex fixed top-0 left-0 right-0 z-50 h-16 justify-center transition-all duration-300">
        <!-- Backdrop: constrained to content width -->
        <!-- transition-colors, not transition-all: transition-all animates the
             backdrop-filter blur radius on every isScrolled threshold
             crossing, re-blurring at intermediate radii each frame for 500ms.
             The blur switches instantly; only the tint fades. -->
        <div
            class="absolute inset-y-0 w-full max-w-[var(--max-width-content)] mx-auto pointer-events-none transition-colors duration-500 [mask-image:linear-gradient(to_right,transparent_0%,black_15%,black_85%,transparent_100%)]"
            :class="isScrolled
                ? 'bg-black/70 backdrop-blur-xl'
                : 'bg-gradient-to-b from-black/60 via-black/30 to-transparent backdrop-blur-[2px]'"
        ></div>
        <div class="relative max-w-[80rem] mx-auto px-4 flex items-center justify-between w-full h-full">
            <!-- LEFT -->
            <div class="flex items-center gap-1">
                <NuxtLink to="/" :class="navBtn">
                    <Icon name="lucide:house" class="text-lg" />
                    <span>{{ isDomainMode && domainConfig?.siteName ? domainConfig.siteName : 'Home' }}</span>
                </NuxtLink>

                <!-- Dynamic nav items from config -->
                <template v-for="(item, idx) in navItems" :key="idx">
                    <!-- Dropdown (has children) -->
                    <Dropdown
                        v-if="hasChildren(item)"
                        v-model="dropdownStates[idx]"
                        :columns="toDropdownColumns(item)"
                        :items="toDropdownItems(item)"
                    >
                        <template #trigger>
                            <button :class="navBtnDropdown">
                                <Icon v-if="item.icon" :name="item.icon" class="text-sm opacity-60" />
                                <span>{{ item.label }}</span>
                                <Icon name="lucide:chevron-down" class="text-xs opacity-60 transition-transform duration-200" :class="dropdownStates[idx] ? 'rotate-180' : ''" />
                            </button>
                        </template>
                    </Dropdown>

                    <!-- Plain link -->
                    <a v-else-if="item.external" :href="item.href" target="_blank" rel="noopener" :class="navBtn">
                        <Icon v-if="item.icon" :name="item.icon" class="text-sm opacity-60" />
                        {{ item.label }}
                        <Icon name="lucide:external-link" class="text-xs opacity-40" />
                    </a>
                    <NuxtLink v-else :to="item.href" :class="navBtn">
                        <Icon v-if="item.icon" :name="item.icon" class="text-sm opacity-60" />
                        {{ item.label }}
                    </NuxtLink>
                </template>
            </div>

            <!-- CENTER: Search -->
            <div class="flex-1 flex justify-center mx-6">
                <button class="flex items-center gap-3 w-full max-w-md h-10 px-4 rounded-lg text-sm text-gray-500 bg-white/[0.04] border border-white/[0.08] hover:bg-blue-500/[0.08] hover:border-white/[0.15] hover:text-blue-400 transition-all cursor-pointer" @click="openSearch">
                    <Icon name="lucide:search" class="text-base" />
                    <span class="flex-1 text-left">Search...</span>
                    <div class="flex items-center gap-0.5">
                        <kbd class="inline-flex items-center justify-center min-w-[1.25rem] h-5 px-1 rounded-sm text-fine font-medium text-gray-500 bg-white/[0.06] border border-white/[0.1]">⌘</kbd>
                        <kbd class="inline-flex items-center justify-center min-w-[1.25rem] h-5 px-1 rounded-sm text-fine font-medium text-gray-500 bg-white/[0.06] border border-white/[0.1]">K</kbd>
                    </div>
                </button>
            </div>

            <!-- RIGHT -->
            <div class="flex items-center gap-0.5">
                <NuxtLink to="/post" :class="iconBtn" aria-label="Post killmail">
                    <Icon name="lucide:upload" class="text-lg" />
                </NuxtLink>

                <Dropdown v-model="bgOpen" align="right">
                    <template #trigger>
                        <button :class="iconBtn" aria-label="Change background">
                            <Icon name="lucide:image" class="text-lg" />
                        </button>
                    </template>
                    <template #default>
                        <div class="w-64 p-3">
                            <!-- Reddit info -->
                            <div v-if="isRedditBackground && redditMeta" class="mb-3 pb-2 border-b border-white/[0.08]">
                                <div class="text-fine uppercase tracking-wider text-orange-400/80 mb-1">r/{{ redditMeta.subreddit }}</div>
                                <div class="text-xs text-gray-400 line-clamp-2">{{ redditMeta.title }}</div>
                            </div>

                            <!-- Reddit button -->
                            <button
                                class="w-full flex items-center gap-2 px-2 py-2 rounded-md text-sm transition-colors mb-3"
                                :class="isRedditBackground ? 'bg-orange-500/20 text-orange-400' : 'text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]'"
                                @click="pickReddit"
                                :disabled="bgLoading"
                            >
                                <Icon name="lucide:shuffle" class="w-4 h-4" :class="{ 'animate-spin': bgLoading }" />
                                Random from r/eveporn
                            </button>

                            <!-- Domain Backgrounds (custom subdomain only) -->
                            <template v-if="isDomainMode && domainConfig?.backgrounds?.length">
                                <div class="text-fine uppercase tracking-wider text-gray-600 mb-2">{{ domainConfig.siteName || 'Domain' }} Backgrounds</div>
                                <div class="grid grid-cols-2 gap-1.5 mb-3">
                                    <button
                                        v-for="bg in domainConfig.backgrounds"
                                        :key="bg"
                                        class="relative aspect-video rounded overflow-hidden border-2 transition-all"
                                        :class="currentBackground === bg ? 'border-blue-400' : 'border-transparent hover:border-white/20'"
                                        @click="pickLocal(bg)"
                                    >
                                        <img :src="bg" class="w-full h-full object-cover" loading="lazy" alt="">
                                        <div v-if="currentBackground === bg" class="absolute inset-0 bg-blue-500/20 flex items-center justify-center">
                                            <Icon name="lucide:check" class="w-4 h-4 text-blue-400" />
                                        </div>
                                    </button>
                                </div>
                            </template>

                            <div class="text-fine uppercase tracking-wider text-gray-600 mb-2">Local Backgrounds</div>

                            <!-- Grid -->
                            <div class="grid grid-cols-2 gap-1.5">
                                <button v-for="bg in localBackgrounds" :key="bg.path"
                                    class="relative aspect-video rounded overflow-hidden border-2 transition-all"
                                    :class="currentBackground === bg.path ? 'border-blue-400' : 'border-transparent hover:border-white/20'"
                                    @click="pickLocal(bg.path)">
                                    <NuxtImg :src="bg.thumb" width="128" height="72" class="w-full h-full object-cover" loading="lazy" alt="" />
                                    <div v-if="currentBackground === bg.path" class="absolute inset-0 bg-blue-500/20 flex items-center justify-center">
                                        <Icon name="lucide:check" class="w-4 h-4 text-blue-400" />
                                    </div>
                                </button>
                            </div>
                        </div>
                    </template>
                </Dropdown>

                <!-- Info (simple list) -->
                <Dropdown v-model="infoOpen" :items="infoItems" align="right">
                    <template #trigger>
                        <button :class="iconBtn" aria-label="Information">
                            <Icon name="lucide:info" class="text-lg" />
                        </button>
                    </template>
                </Dropdown>

                <!-- CSS-driven auth hint: shows correct icon before Vue loads.
                     The inline <script> adds .is-authed to <html> which toggles
                     .auth-login-hint / .auth-avatar-hint via CSS. Once Vue hydrates,
                     ClientOnly swaps in the real interactive dropdown. -->
                <ClientOnly>
                <NotificationBell v-if="isAuthenticated" />

                <Dropdown v-if="isAuthenticated" v-model="userOpen" align="right">
                    <template #trigger>
                        <button class="flex items-center justify-center w-9 h-9 rounded-md hover:bg-blue-500/10 transition-colors overflow-hidden" aria-label="Account">
                            <EveImage
                                :src="`/images/characters/${user!.characterId}/portrait?size=64`"
                                :size="32"
                                :alt="user!.characterName"
                                class="w-8 h-8 rounded-full"
                                loading="eager"
                            />
                        </button>
                    </template>
                    <template #default>
                        <div class="w-64">
                            <!-- Character card — like kill info box -->
                            <NuxtLink :to="`/character/${user!.characterId}`" class="block relative overflow-hidden rounded-t-lg" @click="userOpen = false">
                                <EveImage
                                    :src="`/images/characters/${user!.characterId}/portrait?size=256`"
                                    :size="256"
                                    :alt="user!.characterName"
                                    class="w-full aspect-[4/3] object-cover object-top"
                                    loading="eager"
                                />
                                <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent"></div>
                                <div
                                    class="absolute top-3 right-3 px-2.5 py-1 rounded-md border border-amber-400/20 bg-black/70 text-fine font-bold text-amber-300 tabular-nums backdrop-blur-sm"
                                    v-tooltip="'Available EK Wallet balance'"
                                >
                                    {{ personalWalletDisplay }}
                                </div>
                                <div class="absolute bottom-0 left-0 right-0 p-3">
                                    <div class="text-sm font-bold text-white">{{ user!.characterName }}</div>
                                    <div class="flex items-center gap-3 mt-1.5">
                                        <NuxtLink v-if="user!.corporationId" :to="`/corporation/${user!.corporationId}`" class="flex items-center gap-1.5 z-10 pointer-events-auto" @click.stop="userOpen = false">
                                            <EveImage :src="`/images/corporations/${user!.corporationId}/logo?size=32`" :size="32" :alt="user!.corporationName ?? ''" class="w-5 h-5 rounded hover:scale-125 transition-transform" />
                                            <span class="text-xs text-gray-300 hover:text-blue-400 transition-colors">{{ user!.corporationName }}</span>
                                        </NuxtLink>
                                        <NuxtLink v-if="user!.allianceId" :to="`/alliance/${user!.allianceId}`" class="flex items-center gap-1.5 z-10 pointer-events-auto" @click.stop="userOpen = false">
                                            <EveImage :src="`/images/alliances/${user!.allianceId}/logo?size=32`" :size="32" :alt="user!.allianceName ?? ''" class="w-5 h-5 rounded hover:scale-125 transition-transform" />
                                            <span class="text-xs text-gray-300 hover:text-blue-400 transition-colors">{{ user!.allianceName }}</span>
                                        </NuxtLink>
                                    </div>
                                </div>
                            </NuxtLink>

                            <!-- Menu items -->
                            <div class="p-1.5 border-t border-white/[0.08]">
                                <NuxtLink v-if="user!.isAdmin" to="/admin" class="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-colors" @click="userOpen = false">
                                    <Icon name="lucide:shield-alert" class="text-base opacity-60" />
                                    Admin
                                </NuxtLink>
                                <NuxtLink to="/settings" class="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors" @click="userOpen = false">
                                    <Icon name="lucide:settings" class="text-base opacity-60" />
                                    Settings
                                </NuxtLink>
                                <NuxtLink to="/settings/wallet" class="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors" @click="userOpen = false">
                                    <Icon name="lucide:wallet-cards" class="text-base opacity-60" />
                                    Wallet
                                </NuxtLink>
                                <NuxtLink to="/settings/domains" class="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] transition-colors" @click="userOpen = false">
                                    <Icon name="lucide:globe" class="text-base opacity-60" />
                                    Domains
                                </NuxtLink>
                                <button class="w-full flex items-center gap-2 px-3 py-2 rounded-md text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-colors" @click="logout()">
                                    <Icon name="lucide:log-out" class="text-base opacity-60" />
                                    Logout
                                </button>
                            </div>
                        </div>
                    </template>
                </Dropdown>

                <!-- User: logged out -->
                <Dropdown v-else v-model="loginOpen" align="right">
                    <template #trigger>
                        <button :class="iconBtn" aria-label="Login with EVE Online">
                            <Icon name="lucide:log-in" class="text-lg" />
                        </button>
                    </template>
                    <template #default>
                        <LoginForm
                            :prefs="loginPrefs"
                            :delay-options="ALLOWED_DELAYS"
                            @set="(k, v) => setLoginPref(k, v)"
                            @login="doLogin"
                        />
                    </template>
                </Dropdown>

                <template #fallback>
                    <!-- Login button (hidden when .is-authed) -->
                    <button class="auth-login-hint" :class="iconBtn" aria-label="Login with EVE Online">
                        <Icon name="lucide:log-in" class="text-lg" />
                    </button>
                    <!-- Avatar hint (shown when .is-authed, uses CSS variable set by inline script) -->
                    <div class="auth-avatar-hint hover:bg-blue-500/10 transition-colors" aria-label="Account">
                        <div style="width: 2rem; height: 2rem; border-radius: 9999px; background-image: var(--auth-portrait); background-size: cover;"></div>
                    </div>
                </template>
                </ClientOnly>
            </div>
        </div>
    </nav>

    <!-- ==================== MOBILE NAVBAR ==================== -->
    <nav
        class="md:hidden fixed top-0 left-0 right-0 z-50 transition-all duration-300"
        :class="isMobileMenuOpen ? '-translate-y-full opacity-0 pointer-events-none' : ''"
    >
        <div
            class="absolute inset-0 pointer-events-none transition-all duration-500"
            :class="isScrolled
                ? 'bg-black/80'
                : 'bg-gradient-to-b from-black/70 via-black/40 to-transparent'"
        ></div>
        <div class="relative z-10 flex items-center gap-2 h-12 px-3">
            <NuxtLink to="/" class="flex-shrink-0 flex items-center justify-center w-8 h-8 text-white text-lg" aria-label="EVE-KILL home">
                <Icon name="lucide:house" />
            </NuxtLink>

            <button class="flex-1 min-w-0 flex items-center gap-2 h-8 px-3 rounded-md bg-white/5 border border-white/10 text-xs text-gray-400 box-border" @click="openSearch">
                <Icon name="lucide:search" class="flex-shrink-0 text-sm" />
                <span class="truncate">Search</span>
            </button>

            <div class="flex-shrink-0 flex items-center">
                <button class="flex items-center justify-center w-8 h-8 text-white/70 text-lg" @click="isMobileMenuOpen = true" aria-label="Menu">
                    <Icon name="lucide:menu" />
                </button>
            </div>
        </div>
    </nav>

    <!-- ==================== MOBILE FULLSCREEN MENU ==================== -->
    <Teleport to="body">
        <Transition
            enter-active-class="transition duration-300 ease-out"
            enter-from-class="opacity-0"
            enter-to-class="opacity-100"
            leave-active-class="transition duration-200 ease-in"
            leave-from-class="opacity-100"
            leave-to-class="opacity-0"
        >
            <div v-if="isMobileMenuOpen" class="fixed inset-0 z-[9999] bg-black/50" @click.self="isMobileMenuOpen = false">
                <div class="absolute inset-0 bg-gray-950 overflow-y-auto pointer-events-auto">
                    <!-- Header -->
                    <div class="flex items-center justify-between h-16 px-6 border-b border-white/[0.08]">
                        <h2 class="text-xl font-bold text-white">Menu</h2>
                        <button class="flex items-center justify-center w-10 h-10 rounded-lg text-white hover:bg-blue-500/10" @click="isMobileMenuOpen = false">
                            <Icon name="lucide:x" class="text-xl" />
                        </button>
                    </div>

                    <div class="p-6 space-y-8">
                        <!-- Navigation (from config) -->
                        <section>
                            <h3 class="text-lg font-bold text-white mb-3">Navigation</h3>
                            <div class="space-y-1">
                                <NuxtLink to="/" class="flex items-center gap-3 px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]">
                                    <Icon name="lucide:house" class="text-lg opacity-60" />
                                    {{ isDomainMode && domainConfig?.siteName ? domainConfig.siteName : 'Home' }}
                                </NuxtLink>

                                <template v-for="(item, idx) in navItems" :key="idx">
                                    <!-- Item with children (expandable) -->
                                    <div v-if="hasChildren(item)">
                                        <button class="flex items-center justify-between w-full px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]" @click="toggleMobileSection('nav-' + idx)">
                                            <span class="flex items-center gap-3">
                                                <Icon v-if="item.icon" :name="item.icon" class="text-lg opacity-60" />
                                                {{ item.label }}
                                            </span>
                                            <Icon :name="expandedMobile['nav-' + idx] ? 'lucide:chevron-up' : 'lucide:chevron-down'" class="text-sm opacity-50" />
                                        </button>
                                        <div v-show="expandedMobile['nav-' + idx]" class="ml-4 mt-1 pl-4 border-l border-white/10 space-y-0.5">
                                            <template v-for="group in item.children" :key="group.label">
                                                <div v-if="group.label" class="pt-2 pb-1 px-3 text-fine font-bold uppercase tracking-wider text-blue-400/60">{{ group.label }}</div>
                                                <template v-for="child in group.items" :key="child.href">
                                                    <a v-if="child.external" :href="child.href" target="_blank" rel="noopener" class="flex items-center gap-2 px-3 py-2 rounded text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]">
                                                        <Icon v-if="child.icon" :name="child.icon" class="opacity-60" />
                                                        {{ child.label }}
                                                        <Icon name="lucide:external-link" class="text-xs opacity-40" />
                                                    </a>
                                                    <NuxtLink v-else :to="child.href" class="flex items-center gap-2 px-3 py-2 rounded text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06]">
                                                        <Icon v-if="child.icon" :name="child.icon" class="opacity-60" />
                                                        {{ child.label }}
                                                    </NuxtLink>
                                                </template>
                                            </template>
                                        </div>
                                    </div>

                                    <!-- Plain link -->
                                    <a v-else-if="item.external" :href="item.href" target="_blank" rel="noopener" class="flex items-center gap-3 px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]">
                                        <Icon v-if="item.icon" :name="item.icon" class="text-lg opacity-60" />
                                        {{ item.label }}
                                        <Icon name="lucide:external-link" class="text-xs opacity-40" />
                                    </a>
                                    <NuxtLink v-else :to="item.href" class="flex items-center gap-3 px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]">
                                        <Icon v-if="item.icon" :name="item.icon" class="text-lg opacity-60" />
                                        {{ item.label }}
                                    </NuxtLink>
                                </template>
                                <NuxtLink to="/post" class="flex items-center gap-3 px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]">
                                    <Icon name="lucide:upload" class="text-lg opacity-60" />
                                    Post Killmail
                                </NuxtLink>
                            </div>
                        </section>

                        <!-- Information -->
                        <section>
                            <h3 class="text-lg font-bold text-white mb-3">Information</h3>
                            <div class="space-y-1">
                                <NuxtLink v-for="item in infoItems" :key="item.to" :to="item.to" class="flex items-center px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]">
                                    {{ item.name }}
                                </NuxtLink>
                            </div>
                        </section>

                        <!-- Account -->
                        <section>
                            <h3 class="text-lg font-bold text-white mb-3">Account</h3>
                            <div class="space-y-1">
                                <template v-if="isAuthenticated">
                                    <!-- Character card -->
                                    <NuxtLink :to="`/character/${user!.characterId}`" class="block relative overflow-hidden rounded-lg" @click="isMobileMenuOpen = false">
                                        <EveImage :src="`/images/characters/${user!.characterId}/portrait?size=256`" :size="512" :alt="user!.characterName" class="w-full aspect-[4/3] object-cover object-top" loading="eager" />
                                        <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent"></div>
                                        <div
                                            class="absolute top-4 right-4 px-2.5 py-1 rounded-md border border-amber-400/20 bg-black/70 text-xs font-bold text-amber-300 tabular-nums backdrop-blur-sm"
                                            v-tooltip="'Available EK Wallet balance'"
                                        >
                                            {{ personalWalletDisplay }}
                                        </div>
                                        <div class="absolute bottom-0 left-0 right-0 p-4">
                                            <div class="text-base font-bold text-white">{{ user!.characterName }}</div>
                                            <div class="flex items-center gap-3 mt-1.5">
                                                <span v-if="user!.corporationName" class="flex items-center gap-1.5">
                                                    <EveImage :src="`/images/corporations/${user!.corporationId}/logo?size=32`" :size="32" :alt="user!.corporationName ?? ''" class="w-5 h-5 rounded" />
                                                    <span class="text-xs text-gray-300">{{ user!.corporationName }}</span>
                                                </span>
                                                <span v-if="user!.allianceName" class="flex items-center gap-1.5">
                                                    <EveImage :src="`/images/alliances/${user!.allianceId}/logo?size=32`" :size="32" :alt="user!.allianceName ?? ''" class="w-5 h-5 rounded" />
                                                    <span class="text-xs text-gray-300">{{ user!.allianceName }}</span>
                                                </span>
                                            </div>
                                        </div>
                                    </NuxtLink>
                                    <NuxtLink v-if="user!.isAdmin" to="/admin" class="flex items-center gap-3 px-4 py-3 rounded-lg text-red-400 hover:bg-red-500/10" @click="isMobileMenuOpen = false">
                                        <Icon name="lucide:shield-alert" class="text-lg opacity-60" />
                                        Admin
                                    </NuxtLink>
                                    <NuxtLink to="/settings" class="flex items-center gap-3 px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]" @click="isMobileMenuOpen = false">
                                        <Icon name="lucide:settings" class="text-lg opacity-60" />
                                        Settings
                                    </NuxtLink>
                                    <NuxtLink to="/settings/wallet" class="flex items-center gap-3 px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]" @click="isMobileMenuOpen = false">
                                        <Icon name="lucide:wallet-cards" class="text-lg opacity-60" />
                                        Wallet
                                    </NuxtLink>
                                    <NuxtLink to="/settings/domains" class="flex items-center gap-3 px-4 py-3 rounded-lg text-white hover:bg-blue-500/[0.06]" @click="isMobileMenuOpen = false">
                                        <Icon name="lucide:globe" class="text-lg opacity-60" />
                                        Domains
                                    </NuxtLink>
                                    <button class="w-full flex items-center gap-3 px-4 py-3 rounded-lg text-red-400 hover:bg-red-500/10" @click="logout()">
                                        <Icon name="lucide:log-out" class="text-lg opacity-60" />
                                        Logout
                                    </button>
                                </template>
                                <LoginForm
                                    v-else
                                    :prefs="loginPrefs"
                                    :delay-options="ALLOWED_DELAYS"
                                    mobile
                                    @set="(k, v) => setLoginPref(k, v)"
                                    @login="doLogin"
                                />
                            </div>
                        </section>
                    </div>
                </div>
            </div>
        </Transition>
    </Teleport>
</template>
