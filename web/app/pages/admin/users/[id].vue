<script setup lang="ts">
const route = useRoute()
const { isAuthenticated, isAdmin } = useAuth()

// Auth check — logged out gets the login gate (and returns here after SSO);
// a logged-in non-admin is bounced to the frontpage.
const { data: authCheck } = await useApiFetch('/auth/me')
useState('auth-user').value = authCheck.value?.user ?? null
const isAdminSession = authCheck.value?.user?.isAdmin === true
if (authCheck.value?.user && !isAdminSession) {
    await navigateTo("/")
}

const characterId = Number(route.params.id)
if (!characterId || Number.isNaN(characterId)) {
    throw createError({ statusCode: 400, statusMessage: 'Invalid user ID' })
}

const { data, refresh, error } = await useApiFetch<any>(`/api/admin/users/${characterId}`, { immediate: isAdminSession })

if (isAdminSession) {
    if (error.value) {
        throw createError({
            statusCode: error.value.statusCode || 404,
            statusMessage: (error.value.data as any)?.message || 'User not found',
        })
    }

    // Legacy API shape: endpoint returns 200 with `{ user: null }` for not-found.
    // Treat that as a 404 too so we hit the same error page instead of silently
    // redirecting away.
    if (!data.value?.user) {
        throw createError({ statusCode: 404, statusMessage: 'User not found' })
    }
}

const u = computed(() => data.value?.user)
const esiToken = computed(() => data.value?.esiToken)
const config = computed(() => data.value?.config ?? [])
const esiStats = computed(() => data.value?.esiStats)

const toast = useToast()

const togglingAdmin = ref(false)
const toggleAdmin = async () => {
    if (!u.value) return
    togglingAdmin.value = true
    try {
        await apiFetch(`/api/admin/users/${characterId}/toggle-admin`, { method: 'POST' })
        await refresh()
    } catch (err: any) {
        // Granting or revoking admin is not something to fail quietly into the
        // devtools console — the button looks like it worked either way.
        toast.error(err?.data?.message || err?.statusMessage || err?.message || 'Failed to change admin status')
    } finally {
        togglingAdmin.value = false
    }
}

// Discord link state — mirrors u.discord_user_id but decoupled so the input
// doesn't reset on every refresh. Kept in sync via watchEffect below.
const discordIdInput = ref<string>('')
const savingDiscordId = ref(false)
const discordIdError = ref<string | null>(null)
watchEffect(() => {
    discordIdInput.value = u.value?.discord_user_id ?? ''
})
const saveDiscordId = async () => {
    savingDiscordId.value = true
    discordIdError.value = null
    try {
        const raw = discordIdInput.value.trim()
        await apiFetch(`/api/admin/users/${characterId}/set-discord`, {
            method: 'POST',
            body: { discord_user_id: raw === '' ? null : raw },
        })
        await refresh()
    } catch (err: any) {
        discordIdError.value = err.data?.message || err.statusMessage || err.message || 'Failed to save Discord id'
    } finally {
        savingDiscordId.value = false
    }
}
const clearDiscordId = async () => {
    discordIdInput.value = ''
    await saveDiscordId()
}

const tokenActive = computed(() => {
    if (!esiToken.value?.token_expiry) return false
    return new Date(esiToken.value.token_expiry) > new Date()
})

// fmt / fmtDate come from utils/adminFormat. This page used to define its own
// pair: `fmt` was a byte-for-byte copy, and `fmtDate` was built from
// getFullYear/getHours — i.e. the viewer's local clock — so every timestamp
// here disagreed with the rest of admin, which reads in EVE time (UTC).

useHead({ title: computed(() => u.value ? `${u.value.character_name} — Admin` : 'Admin') })
useSeoMeta({ robots: 'noindex, nofollow' })
</script>

<template>
    <LoginGate v-if="!isAuthenticated" message="Log in with EVE Online to access the admin panel" />
    <div v-else-if="isAdmin && u">
        <!-- Back link -->
        <NuxtLink to="/admin/users" class="inline-flex items-center gap-1.5 text-xs text-gray-500 hover:text-blue-400 transition-colors mb-4">
            <Icon name="lucide:arrow-left" class="text-sm" />
            Back to Users
        </NuxtLink>

        <!-- Header card -->
        <div class="glass-panel overflow-hidden mb-6">
            <div class="relative">
                <img
                    :src="`/images/characters/${u.character_id}/portrait?size=512`"
                    :alt="u.character_name"
                    class="w-full h-32 object-cover object-top"
                >
                <div class="absolute inset-0 bg-gradient-to-t from-black/90 via-black/40 to-transparent"></div>
                <div class="absolute bottom-0 left-0 right-0 p-4 flex items-end gap-4">
                    <img
                        :src="`/images/characters/${u.character_id}/portrait?size=128`"
                        :alt="u.character_name"
                        class="w-16 h-16 rounded-full border-2 border-black/50 flex-shrink-0"
                    >
                    <div class="flex-1 min-w-0 pb-1">
                        <div class="flex items-center gap-2">
                            <span class="text-lg font-bold text-white truncate">{{ u.character_name }}</span>
                            <span v-if="u.is_admin" class="px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-green-500/20 text-green-400 border border-green-500/20">Admin</span>
                        </div>
                        <div class="flex items-center gap-3 mt-0.5">
                            <span v-if="u.corporation_name" class="flex items-center gap-1.5 text-xs text-gray-300">
                                <img :src="`/images/corporations/${u.corporation_id}/logo?size=32`" class="w-4 h-4 rounded">
                                {{ u.corporation_name }}
                            </span>
                            <span v-if="u.alliance_name" class="flex items-center gap-1.5 text-xs text-gray-300">
                                <img :src="`/images/alliances/${u.alliance_id}/logo?size=32`" class="w-4 h-4 rounded">
                                {{ u.alliance_name }}
                            </span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- Account Info -->
            <div class="glass-panel p-5">
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">Account</h2>
                <div class="space-y-3">
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">Character ID</span>
                        <span class="text-gray-300 tabular-nums">{{ u.character_id }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">Owner Hash</span>
                        <span class="text-gray-300 font-mono text-xs truncate ml-4 max-w-[200px]">{{ u.character_owner_hash }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">Last Login</span>
                        <span class="text-gray-300 tabular-nums">{{ fmtDate(u.last_login) }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">Joined</span>
                        <span class="text-gray-300 tabular-nums">{{ fmtDate(u.created_at) }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">Updated</span>
                        <span class="text-gray-300 tabular-nums">{{ fmtDate(u.updated_at) }}</span>
                    </div>
                    <div class="flex justify-between items-center text-sm pt-2 border-t border-white/[0.06]">
                        <span class="text-gray-500">Admin</span>
                        <button
                            class="px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer"
                            :class="u.is_admin
                                ? 'bg-red-500/15 text-red-400 border border-red-500/20 hover:bg-red-500/25'
                                : 'bg-green-500/15 text-green-400 border border-green-500/20 hover:bg-green-500/25'"
                            :disabled="togglingAdmin"
                            @click="toggleAdmin"
                        >
                            <Icon v-if="togglingAdmin" name="lucide:loader-2" class="text-xs animate-spin mr-1 inline" />
                            {{ u.is_admin ? 'Remove Admin' : 'Make Admin' }}
                        </button>
                    </div>
                    <div class="pt-2 border-t border-white/[0.06]">
                        <div class="flex justify-between items-center text-sm mb-2">
                            <span class="text-gray-500">Discord ID</span>
                            <span v-if="u.discord_user_id" class="text-xs text-gray-400 font-mono">linked</span>
                            <span v-else class="text-xs text-gray-600">not linked</span>
                        </div>
                        <div class="flex gap-2">
                            <input
                                v-model="discordIdInput"
                                type="text"
                                inputmode="numeric"
                                pattern="[0-9]*"
                                placeholder="Discord snowflake (numeric)"
                                class="flex-1 px-2 py-1 rounded bg-black/30 border border-white/[0.08] text-sm font-mono text-gray-200 focus:outline-none focus:border-blue-500/40"
                                :disabled="savingDiscordId"
                                @keydown.enter.prevent="saveDiscordId"
                            />
                            <button
                                class="px-3 py-1 rounded-lg text-xs font-medium bg-blue-500/15 text-blue-400 border border-blue-500/20 hover:bg-blue-500/25 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                                :disabled="savingDiscordId"
                                @click="saveDiscordId"
                            >
                                <Icon v-if="savingDiscordId" name="lucide:loader-2" class="text-xs animate-spin mr-1 inline" />
                                Save
                            </button>
                            <button
                                v-if="u.discord_user_id"
                                class="px-3 py-1 rounded-lg text-xs font-medium bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20 transition-colors cursor-pointer disabled:opacity-50"
                                :disabled="savingDiscordId"
                                @click="clearDiscordId"
                            >
                                Clear
                            </button>
                        </div>
                        <p v-if="discordIdError" class="text-xs text-red-400 mt-1">{{ discordIdError }}</p>
                    </div>
                </div>
            </div>

            <!-- ESI Token -->
            <div class="glass-panel p-5">
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">ESI Token</h2>
                <template v-if="esiToken">
                    <div class="space-y-3">
                        <div class="flex justify-between text-sm">
                            <span class="text-gray-500">Status</span>
                            <span :class="tokenActive ? 'text-green-400' : 'text-red-400'">{{ tokenActive ? 'Active' : 'Expired' }}</span>
                        </div>
                        <div class="flex justify-between text-sm">
                            <span class="text-gray-500">Expires</span>
                            <span class="text-gray-300 tabular-nums">{{ fmtDate(esiToken.token_expiry) }}</span>
                        </div>
                        <div class="flex justify-between text-sm">
                            <span class="text-gray-500">Last Fetched</span>
                            <span class="text-gray-300 tabular-nums">{{ fmtDate(esiToken.last_fetched) }}</span>
                        </div>
                        <div class="flex justify-between text-sm">
                            <span class="text-gray-500">Created</span>
                            <span class="text-gray-300 tabular-nums">{{ fmtDate(esiToken.created_at) }}</span>
                        </div>
                    </div>
                    <div class="mt-4 pt-4 border-t border-white/[0.06]">
                        <div class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-2">Scopes</div>
                        <div v-if="esiToken.scopes?.length" class="flex flex-wrap gap-1.5">
                            <span v-for="scope in esiToken.scopes" :key="scope"
                                class="px-2 py-0.5 text-xs rounded bg-blue-500/10 text-blue-400 border border-blue-500/20">
                                {{ scope }}
                            </span>
                        </div>
                        <p v-else class="text-sm text-gray-600">No scopes granted.</p>
                    </div>
                </template>
                <p v-else class="text-sm text-gray-600">No ESI token found.</p>
            </div>

            <!-- ESI Activity -->
            <div class="glass-panel p-5">
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">ESI Activity</h2>
                <div class="space-y-3">
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">Total Requests</span>
                        <span class="text-gray-300 tabular-nums">{{ fmt(esiStats?.total_requests) }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">Errors</span>
                        <span class="tabular-nums" :class="(esiStats?.total_errors ?? 0) > 0 ? 'text-red-400' : 'text-gray-300'">{{ fmt(esiStats?.total_errors) }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">New Items Found</span>
                        <span class="tabular-nums" :class="(esiStats?.total_new_items ?? 0) > 0 ? 'text-blue-400' : 'text-gray-300'">{{ fmt(esiStats?.total_new_items) }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-gray-500">Last Request</span>
                        <span class="text-gray-300 tabular-nums">{{ fmtDate(esiStats?.last_request) }}</span>
                    </div>
                </div>
                <div class="mt-4 pt-4 border-t border-white/[0.06]">
                    <NuxtLink
                        :to="{ path: '/admin/esi', query: { character_id: u.character_id, character_name: u.character_name } }"
                        class="text-xs text-gray-500 hover:text-blue-400 transition-colors"
                    >
                        View in ESI logs →
                    </NuxtLink>
                </div>
            </div>

            <!-- Config -->
            <div v-if="config.length" class="glass-panel p-5 md:col-span-2">
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">User Config</h2>
                <div class="overflow-x-auto">
                    <table class="w-full text-sm">
                        <thead>
                            <tr class="border-b border-white/[0.06]">
                                <th class="text-left px-3 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Key</th>
                                <th class="text-left px-3 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Value</th>
                                <th class="text-right px-3 py-2 text-fine font-bold uppercase tracking-[0.15em] text-gray-600">Updated</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="c in config" :key="c.key" class="border-b border-white/[0.04]">
                                <td class="px-3 py-2 text-gray-300 font-mono text-xs">{{ c.key }}</td>
                                <td class="px-3 py-2 text-gray-400 font-mono text-xs truncate max-w-[400px]">{{ JSON.stringify(c.value) }}</td>
                                <td class="px-3 py-2 text-right text-gray-500 tabular-nums text-xs">{{ fmtDate(c.updated_at) }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <!-- Quick links -->
        <div class="mt-4 flex items-center gap-3">
            <NuxtLink
                :to="`/character/${u.character_id}`"
                class="px-3 py-1.5 rounded-lg text-xs font-medium text-gray-400 bg-white/[0.04] border border-white/[0.06] hover:text-blue-400 hover:bg-blue-500/[0.08] transition-colors"
            >
                View Public Profile
            </NuxtLink>
        </div>
    </div>
</template>
