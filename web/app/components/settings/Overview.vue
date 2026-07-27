<script setup lang="ts">
// Settings → Overview section. Extracted verbatim from pages/settings/[[tab]].vue.
defineProps<{
    /** `/api/user/overview` payload — fetched eagerly (awaited) by the settings page. */
    overview: any
    // Formatters shared with the ESI section — owned by the settings page.
}>()

const { user, logout } = useAuth()

// timeSince comes from utils/adminFormat — the local copy that used to sit here
// was the same function plus a 'just now' case, which has been folded into the
// shared one so both admin and settings say the same thing.
</script>

<template>
    <div class="space-y-4">
        <!-- Summary cards -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div class="glass-panel p-4 text-center">
                <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">ESI Token</div>
                <div class="text-base font-bold" :class="overview?.esiToken
                    ? (new Date(overview.esiToken.tokenExpiry) > new Date() ? 'text-green-400' : 'text-yellow-400')
                    : 'text-red-400'">
                    {{ !overview?.esiToken ? 'None' : new Date(overview.esiToken.tokenExpiry) > new Date() ? 'Active' : 'Refreshing' }}
                </div>
                <div class="text-xs text-gray-500">{{ overview?.esiToken?.scopeCount ?? 0 }} scopes</div>
            </div>
            <div class="glass-panel p-4 text-center">
                <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">Requests (24h)</div>
                <div class="text-base font-bold text-white tabular-nums">{{ fmt(overview?.esiStats?.requests_24h) }}</div>
                <div class="text-xs text-gray-500">{{ fmt(overview?.esiStats?.total_requests) }} total</div>
            </div>
            <div class="glass-panel p-4 text-center">
                <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">New Items (24h)</div>
                <div class="text-base font-bold text-blue-400 tabular-nums">{{ fmt(overview?.esiStats?.new_items_24h) }}</div>
                <div class="text-xs text-gray-500">{{ fmt(overview?.esiStats?.total_new_items) }} total</div>
            </div>
            <div class="glass-panel p-4 text-center">
                <div class="text-fine uppercase tracking-wider text-gray-500 mb-1">Errors (24h)</div>
                <div class="text-base font-bold tabular-nums" :class="(overview?.esiStats?.errors_24h ?? 0) > 0 ? 'text-red-400' : 'text-white'">{{ fmt(overview?.esiStats?.errors_24h) }}</div>
                <div class="text-xs text-gray-500">{{ fmt(overview?.esiStats?.total_errors) }} total</div>
            </div>
        </div>

        <!-- Account details -->
        <div class="glass-panel p-5">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">Account</h2>
            <div class="space-y-3">
                <div class="flex justify-between text-sm">
                    <span class="text-gray-500">Character ID</span>
                    <span class="text-gray-300 tabular-nums">{{ user.characterId }}</span>
                </div>
                <div class="flex justify-between text-sm">
                    <span class="text-gray-500">Owner Hash</span>
                    <span class="text-gray-300 font-mono text-xs truncate ml-4 max-w-[200px]">{{ user.characterOwnerHash }}</span>
                </div>
                <div class="flex justify-between text-sm">
                    <span class="text-gray-500">Member Since</span>
                    <span class="text-gray-300 tabular-nums">{{ fmtDate(overview?.account?.createdAt) }}</span>
                </div>
                <div class="flex justify-between text-sm">
                    <span class="text-gray-500">Last Login</span>
                    <span class="text-gray-300 tabular-nums">{{ timeSince(overview?.account?.lastLogin) }}</span>
                </div>
                <div class="flex justify-between text-sm">
                    <span class="text-gray-500">Last ESI Fetch</span>
                    <span class="text-gray-300 tabular-nums">{{ timeSince(overview?.esiToken?.lastFetched) }}</span>
                </div>
                <div class="flex justify-between text-sm">
                    <span class="text-gray-500">Admin</span>
                    <span :class="user.isAdmin ? 'text-green-400' : 'text-gray-600'">{{ user.isAdmin ? 'Yes' : 'No' }}</span>
                </div>
            </div>
        </div>

        <!-- Quick links -->
        <div class="glass-panel p-5">
            <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500 mb-4">Quick Links</h2>
            <div class="space-y-1">
                <NuxtLink :to="`/character/${user.characterId}`" class="flex items-center gap-2 px-3 py-2 -mx-3 rounded-md text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.04] transition-colors">
                    <Icon name="lucide:external-link" class="text-base opacity-60" />
                    View public profile
                </NuxtLink>
                <NuxtLink v-if="user.corporationId" :to="`/corporation/${user.corporationId}`" class="flex items-center gap-2 px-3 py-2 -mx-3 rounded-md text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.04] transition-colors">
                    <Icon name="lucide:building" class="text-base opacity-60" />
                    {{ user.corporationName }}
                </NuxtLink>
                <NuxtLink v-if="user.allianceId" :to="`/alliance/${user.allianceId}`" class="flex items-center gap-2 px-3 py-2 -mx-3 rounded-md text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.04] transition-colors">
                    <Icon name="lucide:shield" class="text-base opacity-60" />
                    {{ user.allianceName }}
                </NuxtLink>
            </div>
        </div>

        <!-- Mobile logout -->
        <button
            class="md:hidden flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium text-red-400 bg-red-500/10 border border-red-500/20 hover:bg-red-500/20 transition-colors cursor-pointer"
            @click="logout()"
        >
            <Icon name="lucide:log-out" class="text-base" />
            Logout
        </button>
    </div>
</template>
