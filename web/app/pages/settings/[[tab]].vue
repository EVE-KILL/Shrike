<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const { user, isAuthenticated, logout } = useAuth()

// Await real auth state before rendering — the client-side auth hint can be
// stale (expired session), so it must not decide between page and login gate.
const { data: authCheck } = await useApiFetch('/auth/me')
useState('auth-user').value = authCheck.value?.user ?? null

// Sections
const sections = [
    { id: 'overview', label: 'Overview', icon: 'lucide:user' },
    { id: 'wallet', label: 'Wallet', icon: 'lucide:wallet-cards' },
    { id: 'sessions', label: 'Sessions', icon: 'lucide:monitor-smartphone' },
    { id: 'description', label: 'Description', icon: 'lucide:user-pen' },
    { id: 'esi', label: 'ESI', icon: 'lucide:activity' },
    { id: 'appearance', label: 'Appearance', icon: 'lucide:palette' },
    { id: 'preferences', label: 'Preferences', icon: 'lucide:sliders-horizontal' },
    { id: 'domains', label: 'Domains', icon: 'lucide:globe' },
    { id: 'comments', label: 'Comments', icon: 'lucide:message-square' },
] as const
type SectionId = typeof sections[number]['id']
const sectionIds = new Set(sections.map(s => s.id))

const activeSection = computed<SectionId>(() => {
    const param = route.params.tab as string
    if (param && sectionIds.has(param as SectionId)) return param as SectionId
    return 'overview'
})

const setSection = (id: SectionId) => {
    const path = id === 'overview' ? '/settings' : `/settings/${id}`
    router.push(path)
}

// ── Data fetching ───────────────────────────────────────────────────────────
const { data: overview } = await useApiFetch('/api/user/overview')

const { data: tokenInfo } = await useApiFetch('/auth/token-info', {
    immediate: isAuthenticated.value,
}) as { data: Ref<{ scopes: string[]; token_expiry: string | null } | null> }

// ── Appearance (global theme hydration) ─────────────────────────────────────
// Runs on every settings load regardless of tab — kept in the page so a user
// with an empty theme cookie gets their backend-saved theme applied even if
// they never open the Appearance tab. The Appearance section component calls
// useTheme() itself for its editing UI (shared useState under the hood).
const { themeOverrides } = useTheme()
// Hydrate theme from backend settings if cookie is empty (e.g. new browser)
const savedTheme = (user.value?.settings?.theme ?? null) as Record<string, string> | null
if (savedTheme && Object.keys(savedTheme).length > 0 && Object.keys(themeOverrides.value).length === 0) {
    themeOverrides.value = { ...savedTheme }
}

// Formatters shared by the Overview and ESI sections (passed down as props)
useHead({ title: computed(() => {
    const s = sections.find(s => s.id === activeSection.value)
    return s ? `${s.label} — Settings` : 'Settings'
})})
useSeoMeta({ robots: 'noindex, nofollow' })
</script>

<template>
    <div v-if="isAuthenticated && user" class="w-full">
        <!-- Header card -->
        <div class="hero-surface glass-panel overflow-hidden mb-6">
            <NuxtLink :to="`/character/${user.characterId}`" class="block relative">
                <img
                    :src="`/images/characters/${user.characterId}/portrait?size=512`"
                    :alt="user.characterName"
                    class="w-full h-32 object-cover object-top"
                >
                <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent"></div>
                <div class="absolute bottom-0 left-0 right-0 p-4 flex items-end gap-4">
                    <img
                        :src="`/images/characters/${user.characterId}/portrait?size=128`"
                        :alt="user.characterName"
                        class="w-16 h-16 rounded-full border-2 border-black/50 flex-shrink-0"
                    >
                    <div class="flex-1 min-w-0 pb-1">
                        <div class="text-lg font-bold text-white truncate">{{ user.characterName }}</div>
                        <div class="flex items-center gap-3 mt-0.5">
                            <span v-if="user.corporationName" class="flex items-center gap-1.5 text-xs text-gray-300">
                                <img :src="`/images/corporations/${user.corporationId}/logo?size=32`" class="w-4 h-4 rounded">
                                {{ user.corporationName }}
                            </span>
                            <span v-if="user.allianceName" class="flex items-center gap-1.5 text-xs text-gray-300">
                                <img :src="`/images/alliances/${user.allianceId}/logo?size=32`" class="w-4 h-4 rounded">
                                {{ user.allianceName }}
                            </span>
                        </div>
                    </div>
                </div>
            </NuxtLink>
        </div>

        <!-- Layout: sidebar + content -->
        <div class="flex flex-col md:flex-row gap-6">
            <SectionNav :sections="sections" :active="activeSection" @select="setSection($event as SectionId)">
                <template #footer>
                    <NuxtLink
                        v-if="user.isAdmin"
                        to="/admin"
                        class="flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04] transition-colors mb-0.5"
                    >
                        <Icon name="lucide:shield-alert" class="text-base" />
                        Admin Panel
                    </NuxtLink>
                    <button
                        class="flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-colors w-full text-left cursor-pointer"
                        @click="logout()"
                    >
                        <Icon name="lucide:log-out" class="text-base" />
                        Logout
                    </button>
                </template>
            </SectionNav>

            <!-- Content -->
            <div class="flex-1 min-w-0">
                <!-- Sections. Each is KeepAlive'd so a visited section keeps its
                     state when switching away (all of this state used to live in
                     this page), and each section's fetches fire on first visit
                     (mount) — matching the old immediate/watch lazy-loading. -->
                <KeepAlive>
                    <LazySettingsOverview v-if="activeSection === 'overview'" :overview="overview" />
                    <LazySettingsWallet v-else-if="activeSection === 'wallet'" />
                    <LazySettingsSessions v-else-if="activeSection === 'sessions'" />
                    <LazySettingsEsi v-else-if="activeSection === 'esi'" :token-info="tokenInfo" />
                    <LazySettingsDescription v-else-if="activeSection === 'description'" />
                    <LazySettingsPreferences v-else-if="activeSection === 'preferences'" />
                    <LazySettingsAppearance v-else-if="activeSection === 'appearance'" />
                    <LazySettingsDomains v-else-if="activeSection === 'domains'" />
                    <LazySettingsComments v-else-if="activeSection === 'comments'" />
                </KeepAlive>
            </div>
        </div>
    </div>
    <LoginGate v-else message="Log in with EVE Online to access your settings" />
</template>
