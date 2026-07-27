<script setup lang="ts">
const route = useRoute()
const { isAuthenticated, isAdmin } = useAuth()

// Auth check — logged out gets the login gate (and returns here after SSO);
// a logged-in non-admin is bounced to the frontpage.
const { data: authCheck } = await useApiFetch<{ user: any }>('/api/auth/me')
useState('auth-user').value = authCheck.value?.user ?? null
const isAdminSession = authCheck.value?.user?.isAdmin === true
if (authCheck.value?.user && !isAdminSession) {
    await navigateTo('/')
}

const domainId = Number(route.params.id)
if (!domainId || Number.isNaN(domainId)) {
    throw createError({ statusCode: 400, statusMessage: 'Invalid domain ID' })
}
const { data, refresh, error } = await useApiFetch<{ domain: any; assets: any[] }>(`/api/admin/domains/${domainId}`, { immediate: isAdminSession })

if (isAdminSession && error.value) {
    throw createError({
        statusCode: error.value.statusCode || 404,
        statusMessage: (error.value.data as any)?.message || 'Domain not found',
    })
}

const activeTab = ref<'approvals' | 'files'>('approvals')
const reviewingAsset = ref<number | null>(null)
const rejectReason = ref('')

const domain = computed(() => data.value?.domain)
const assets = computed(() => data.value?.assets || [])

const pendingAssets = computed(() => assets.value.filter((a: any) => a.status === 'pending'))
const allFiles = computed(() => assets.value)

const toast = useToast()

const reviewAsset = async (assetId: number, action: 'approve' | 'reject') => {
    reviewingAsset.value = assetId
    try {
        await apiFetch(`/api/admin/domains/${domainId}/assets/${assetId}/review`, {
            method: 'POST',
            body: {
                action,
                reason: action === 'reject' ? rejectReason.value : undefined,
            },
        })
        rejectReason.value = ''
        await refresh()
    } catch (err: any) {
        // A failed review used to reach only the devtools console: the row
        // stayed pending and looked like nothing had been clicked.
        toast.error(err?.data?.message || err?.statusMessage || err?.message || `Failed to ${action} asset`)
    } finally {
        reviewingAsset.value = null
    }
}

const togglingActive = ref(false)
const toggleActive = async () => {
    togglingActive.value = true
    try {
        await apiFetch(`/api/admin/domains/${domainId}/toggle-active`, { method: 'POST' })
        await refresh()
    } catch (err: any) {
        toast.error(err?.data?.message || err?.statusMessage || err?.message || 'Failed to change domain status')
    } finally {
        togglingActive.value = false
    }
}

const statusColor = (status: string) => {
    if (status === 'approved') return 'text-green-400 bg-green-500/20'
    if (status === 'rejected') return 'text-red-400 bg-red-500/20'
    return 'text-yellow-400 bg-yellow-500/20'
}

// fmtDate comes from utils/adminFormat. The copy that used to live here was
// built from getFullYear/getHours — the viewer's local clock — while the rest
// of admin reads in EVE time (UTC).

const fmtSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

useHead({ title: computed(() => domain.value ? `${domain.value.site_name || domain.value.subdomain} — Admin` : 'Domain — Admin') })
useSeoMeta({ robots: 'noindex, nofollow' })
</script>

<template>
    <LoginGate v-if="!isAuthenticated" message="Log in with EVE Online to access the admin panel" />
    <div v-else-if="isAdmin && domain">
        <!-- Back link + header -->
        <div class="mb-6">
            <NuxtLink to="/admin/domains" class="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-blue-400 transition-colors mb-3">
                <Icon name="lucide:arrow-left" class="text-base" />
                Back to Domains
            </NuxtLink>

            <div class="glass-panel p-5">
                <div class="flex items-start justify-between gap-4">
                    <div>
                        <h1 class="text-lg font-bold text-white">{{ domain.site_name || domain.subdomain }}</h1>
                        <p class="text-sm text-gray-400 mt-0.5">{{ domain.subdomain }}.eve-kill.com</p>
                        <div class="flex items-center gap-4 mt-2 text-xs text-gray-500">
                            <NuxtLink :to="`/character/${domain.user_id}`" class="flex items-center gap-1.5 hover:text-blue-400">
                                <img :src="`/images/characters/${domain.user_id}/portrait?size=32`" class="w-4 h-4 rounded-full">
                                {{ domain.owner_name || domain.user_id }}
                            </NuxtLink>
                            <span>Created {{ fmtDate(domain.created_at) }}</span>
                        </div>
                    </div>
                    <div class="flex items-center gap-2">
                        <span :class="domain.active ? 'text-green-400' : 'text-red-400'" class="text-xs font-medium">
                            {{ domain.active ? 'Active' : 'Inactive' }}
                        </span>
                        <button
                            class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                            :class="domain.active
                                ? 'text-red-400 hover:bg-red-500/10 border border-red-500/20'
                                : 'text-green-400 hover:bg-green-500/10 border border-green-500/20'"
                            :disabled="togglingActive"
                            @click="toggleActive"
                        >
                            <Icon v-if="togglingActive" name="lucide:loader-2" class="text-xs animate-spin mr-1 inline" />
                            {{ domain.active ? 'Deactivate' : 'Activate' }}
                        </button>
                    </div>
                </div>

                <!-- Domain info -->
                <div class="grid grid-cols-2 md:grid-cols-4 gap-3 mt-4 pt-4 border-t border-white/[0.06]">
                    <div>
                        <div class="text-fine text-gray-500 uppercase tracking-wider">Entities</div>
                        <div class="mt-1 flex flex-wrap gap-1">
                            <span v-for="e in (domain.entities || [])" :key="`${e.type}-${e.id}`" class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-fine bg-white/[0.06] text-gray-300">
                                <img :src="`/images/${e.type === 'character' ? 'characters' : e.type === 'corporation' ? 'corporations' : 'alliances'}/${e.id}/${e.type === 'character' ? 'portrait' : 'logo'}?size=32`" class="w-3 h-3 rounded-full">
                                {{ e.name }}
                            </span>
                        </div>
                    </div>
                    <div>
                        <div class="text-fine text-gray-500 uppercase tracking-wider">Description</div>
                        <div class="mt-1 text-xs text-gray-300">{{ domain.site_description || '—' }}</div>
                    </div>
                    <div>
                        <div class="text-fine text-gray-500 uppercase tracking-wider">Navbar Links</div>
                        <div class="mt-1 text-xs text-gray-300">{{ (domain.navbar_links || []).length }} links</div>
                    </div>
                    <div>
                        <div class="text-fine text-gray-500 uppercase tracking-wider">Updated</div>
                        <div class="mt-1 text-xs text-gray-300">{{ fmtDate(domain.updated_at) }}</div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Tab bar -->
        <div class="flex gap-1 mb-4">
            <button
                class="px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer"
                :class="activeTab === 'approvals' ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="activeTab = 'approvals'"
            >
                Approvals
                <span v-if="pendingAssets.length > 0" class="ml-1.5 px-1.5 py-0.5 rounded-full bg-yellow-500/20 text-yellow-400 text-fine font-medium">{{ pendingAssets.length }}</span>
            </button>
            <button
                class="px-4 py-2 rounded-lg text-sm font-medium transition-colors cursor-pointer"
                :class="activeTab === 'files' ? 'bg-blue-500/10 text-blue-400' : 'text-gray-500 hover:text-blue-400 hover:bg-blue-500/[0.04]'"
                @click="activeTab = 'files'"
            >
                Files
                <span class="ml-1.5 text-fine text-gray-600">({{ allFiles.length }})</span>
            </button>
        </div>

        <!-- ── Approvals Tab ── -->
        <div v-if="activeTab === 'approvals'" class="space-y-4">
            <div v-if="pendingAssets.length === 0" class="glass-panel p-8 text-center">
                <Icon name="lucide:check-circle" class="text-3xl text-green-400/50 mb-2" />
                <p class="text-sm text-gray-400">No pending approvals</p>
            </div>

            <div v-for="asset in pendingAssets" :key="asset.id" class="glass-panel overflow-hidden">
                <!-- Preview -->
                <div class="p-4">
                    <div class="flex items-center justify-between mb-3">
                        <div class="flex items-center gap-2">
                            <span class="px-2 py-0.5 rounded-full text-fine font-medium" :class="statusColor(asset.status)">{{ asset.status }}</span>
                            <span class="text-sm font-medium text-white capitalize">{{ asset.type }}</span>
                            <span class="text-fine text-gray-500">{{ fmtSize(asset.file_size) }} &middot; {{ asset.content_type }}</span>
                        </div>
                        <span class="text-fine text-gray-500">{{ fmtDate(asset.created_at) }}</span>
                    </div>

                    <div :class="asset.type === 'logo' ? 'w-24 h-24' : 'w-full h-40'" class="rounded-lg overflow-hidden border border-white/[0.08] bg-black/20">
                        <img :src="`/api/admin/domains/${domainId}/assets/${asset.id}/preview`" :alt="asset.type" class="w-full h-full object-cover">
                    </div>
                </div>

                <!-- Actions -->
                <div class="px-4 py-3 bg-white/[0.02] border-t border-white/[0.06] flex items-center gap-3">
                    <button
                        class="px-4 py-2 rounded-md text-sm font-medium bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors disabled:opacity-50 cursor-pointer"
                        :disabled="reviewingAsset === asset.id"
                        @click="reviewAsset(asset.id, 'approve')"
                    >
                        <Icon name="lucide:check" class="text-base mr-1" />
                        Approve
                    </button>
                    <div class="flex-1 flex items-center gap-2">
                        <input
                            v-model="rejectReason"
                            type="text"
                            placeholder="Reason (optional)"
                            class="flex-1 px-3 py-2 rounded-md bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-gray-600 focus:outline-none focus:border-red-500/50"
                        >
                        <button
                            class="px-4 py-2 rounded-md text-sm font-medium bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors disabled:opacity-50 cursor-pointer"
                            :disabled="reviewingAsset === asset.id"
                            @click="reviewAsset(asset.id, 'reject')"
                        >
                            <Icon name="lucide:x" class="text-base mr-1" />
                            Reject
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- ── Files Tab ── -->
        <div v-if="activeTab === 'files'" class="space-y-4">
            <div v-if="allFiles.length === 0" class="glass-panel p-8 text-center">
                <Icon name="lucide:folder-open" class="text-3xl text-gray-600 mb-2" />
                <p class="text-sm text-gray-400">No files uploaded</p>
            </div>

            <div class="glass-panel overflow-hidden">
                <table v-if="allFiles.length > 0" class="w-full text-sm">
                    <thead>
                        <tr class="border-b border-white/[0.06] text-left text-fine text-gray-500 uppercase tracking-wider">
                            <th class="px-4 py-3">Preview</th>
                            <th class="px-4 py-3">Type</th>
                            <th class="px-4 py-3">Status</th>
                            <th class="px-4 py-3 hidden md:table-cell">Size</th>
                            <th class="px-4 py-3 hidden md:table-cell">Uploaded</th>
                            <th class="px-4 py-3 hidden md:table-cell">Reviewed</th>
                            <th class="px-4 py-3">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="asset in allFiles" :key="asset.id" class="border-b border-white/[0.04] hover:bg-white/[0.02] transition-colors">
                            <td class="px-4 py-3">
                                <div :class="asset.type === 'logo' ? 'w-10 h-10' : 'w-24 h-14'" class="rounded overflow-hidden border border-white/[0.08] bg-black/20">
                                    <img :src="`/api/admin/domains/${domainId}/assets/${asset.id}/preview`" :alt="asset.type" class="w-full h-full object-cover">
                                </div>
                            </td>
                            <td class="px-4 py-3 capitalize text-gray-300">{{ asset.type }}</td>
                            <td class="px-4 py-3">
                                <span class="px-2 py-0.5 rounded-full text-fine font-medium" :class="statusColor(asset.status)">{{ asset.status }}</span>
                                <div v-if="asset.reject_reason" class="text-fine text-red-400/70 mt-1">{{ asset.reject_reason }}</div>
                            </td>
                            <td class="px-4 py-3 hidden md:table-cell text-fine text-gray-400">{{ fmtSize(asset.file_size) }}</td>
                            <td class="px-4 py-3 hidden md:table-cell text-fine text-gray-500">{{ fmtDate(asset.created_at) }}</td>
                            <td class="px-4 py-3 hidden md:table-cell text-fine text-gray-500">
                                {{ asset.reviewed_at ? fmtDate(asset.reviewed_at) : '—' }}
                            </td>
                            <td class="px-4 py-3">
                                <div v-if="asset.status === 'pending'" class="flex items-center gap-1">
                                    <button
                                        class="p-1.5 rounded text-green-400 hover:bg-green-500/10 transition-colors cursor-pointer"
                                        :disabled="reviewingAsset === asset.id"
                                        v-tooltip="'Approve'"
                                        @click="reviewAsset(asset.id, 'approve')"
                                    >
                                        <Icon name="lucide:check" class="text-sm" />
                                    </button>
                                    <button
                                        class="p-1.5 rounded text-red-400 hover:bg-red-500/10 transition-colors cursor-pointer"
                                        :disabled="reviewingAsset === asset.id"
                                        v-tooltip="'Reject'"
                                        @click="reviewAsset(asset.id, 'reject')"
                                    >
                                        <Icon name="lucide:x" class="text-sm" />
                                    </button>
                                </div>
                                <span v-else class="text-fine text-gray-600">—</span>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>
