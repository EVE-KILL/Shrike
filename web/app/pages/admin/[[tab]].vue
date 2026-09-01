<script setup lang="ts">
import type { Component } from 'vue'
import AdminOverview from '~/components/admin/Overview.vue'
import AdminUsers from '~/components/admin/Users.vue'
import AdminDomains from '~/components/admin/Domains.vue'
import AdminCampaigns from '~/components/admin/Campaigns.vue'
import AdminModeration from '~/components/admin/Moderation.vue'
import AdminComments from '~/components/admin/Comments.vue'
import AdminEsi from '~/components/admin/Esi.vue'
import AdminWallet from '~/components/admin/CorporationWallet.vue'
import AdminAnnouncements from '~/components/admin/Announcements.vue'
import AdminBlog from '~/components/admin/Blog.vue'
import AdminRiver from '~/components/admin/River.vue'

const route = useRoute()
const router = useRouter()
const { user, isAuthenticated, isAdmin } = useAuth()

// On direct navigation the auth hint seeds isAdmin=false before /auth/me resolves.
// Await the real auth state from the server before deciding to redirect.
const { data: authCheck } = await useApiFetch('/auth/me')
useState('auth-user').value = authCheck.value?.user ?? null
const isAdminSession = authCheck.value?.user?.isAdmin === true
// Logged out gets the login gate (and returns here after SSO); a logged-in
// non-admin is bounced to the frontpage.
if (authCheck.value?.user && !authCheck.value.user.isAdmin) {
    await navigateTo("/")
}

// Sections
const sections = [
    { id: 'overview', label: 'Overview', icon: 'lucide:layout-dashboard' },
    { id: 'users', label: 'Users', icon: 'lucide:users' },
    { id: 'domains', label: 'Domains', icon: 'lucide:globe' },
    { id: 'campaigns', label: 'Campaigns', icon: 'lucide:flag' },
    { id: 'moderation', label: 'Moderation', icon: 'lucide:shield-check' },
    { id: 'comments', label: 'Comments', icon: 'lucide:message-square' },
    { id: 'esi', label: 'ESI Monitoring', icon: 'lucide:activity' },
    { id: 'river', label: 'River Workers', icon: 'lucide:list-restart' },
    { id: 'wallet', label: 'Wallet', icon: 'lucide:wallet-cards' },
    { id: 'announcements', label: 'Announcements', icon: 'lucide:megaphone' },
    { id: 'blog', label: 'Blog', icon: 'lucide:newspaper' },
] as const
type SectionId = typeof sections[number]['id']
const sectionIds = new Set(sections.map(s => s.id))

const activeSection = computed<SectionId>(() => {
    const param = route.params.tab as string
    if (param && sectionIds.has(param as SectionId)) return param as SectionId
    return 'overview'
})

const setSection = (id: SectionId) => {
    const path = id === 'overview' ? '/admin' : `/admin/${id}`
    router.push(path)
}

// Section components — rendered via <KeepAlive> so a visited section's state
// (filters, pagination, loaded data) persists when switching away.
const sectionComponents: Record<SectionId, Component> = {
    overview: AdminOverview,
    users: AdminUsers,
    domains: AdminDomains,
    campaigns: AdminCampaigns,
    moderation: AdminModeration,
    comments: AdminComments,
    esi: AdminEsi,
    river: AdminRiver,
    wallet: AdminWallet,
    announcements: AdminAnnouncements,
    blog: AdminBlog,
}

// ── Data fetching ───────────────────────────────────────────────────────────
// Overview is the only blocking fetch — each section fetches its own data
// lazily inside its component. esiData lives here because it is shared
// between the Overview and ESI sections.
const { data: overview, refresh: refreshOverview } = await useApiFetch('/api/admin/overview', {
    immediate: isAdminSession,
})
const { data: esiData, refresh: refreshEsi } = useApiFetch('/api/admin/esi', {
    immediate: isAdminSession && (activeSection.value === 'esi' || activeSection.value === 'overview'),
    lazy: true,
})

// Lazy-load the shared ESI summary when switching to the ESI tab
watch(activeSection, (section) => {
    if (isAdminSession && section === 'esi' && !esiData.value) refreshEsi()
}, { immediate: true })

// Only the Overview and ESI sections take the shared data as props
const sectionProps = computed(() =>
    activeSection.value === 'overview' || activeSection.value === 'esi'
        ? { overview: overview.value, esiData: esiData.value }
        : {},
)

/**
 * Sections, annotated with how much work is waiting in each queue.
 *
 * The counts ride along on the overview payload the page already awaits, so
 * this costs no extra request — and it means an admin landing on any tab can
 * see that Moderation has a backlog without going looking for it.
 */
const badgedSections = computed(() => {
    const a = overview.value?.attention
    if (!a) return sections.map(s => ({ ...s }))
    const badges: Partial<Record<SectionId, number>> = {
        moderation: a.moderation ?? 0,
        domains: a.domainAssets ?? 0,
        campaigns: (a.campaignsPaused ?? 0) + (a.campaignsFailed ?? 0),
    }
    return sections.map(s => ({ ...s, badge: badges[s.id] || undefined }))
})

useHead({ title: computed(() => {
    const s = sections.find(s => s.id === activeSection.value)
    return s ? `${s.label} — Admin` : 'Admin'
})})
useSeoMeta({ robots: 'noindex, nofollow' })
</script>

<template>
    <div v-if="isAuthenticated && isAdmin">
        <!-- Header -->
        <div class="glass-panel p-5 mb-6">
            <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-red-500/20 flex items-center justify-center">
                    <Icon name="lucide:shield-alert" class="text-lg text-red-400" />
                </div>
                <div>
                    <h1 class="text-lg font-bold text-white">Admin Panel</h1>
                    <p class="text-xs text-gray-500">Logged in as {{ user!.characterName }}</p>
                </div>
            </div>
        </div>

        <!-- Layout: sidebar + content -->
        <div class="flex flex-col md:flex-row gap-6">
            <SectionNav :sections="badgedSections" :active="activeSection" @select="setSection($event as SectionId)">
                <template #footer>
                    <NuxtLink to="/settings" class="flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04] transition-colors">
                        <Icon name="lucide:arrow-left" class="text-base" />
                        Back to Settings
                    </NuxtLink>
                </template>
            </SectionNav>

            <!-- Content -->
            <div class="flex-1 min-w-0">

                <KeepAlive>
                    <component :is="sectionComponents[activeSection]" v-bind="sectionProps" />
                </KeepAlive>
            </div>
        </div>
    </div>
    <LoginGate v-else-if="!isAuthenticated" message="Log in with EVE Online to access the admin panel" />
</template>
