<script setup lang="ts">
// Users section state
const userSearch = ref('')
const userPage = ref(1)
const userSort = ref('last_login')
const userDir = ref('desc')
const { data: usersData, refresh: refreshUsers } = useApiFetch('/api/admin/users', {
    query: { search: userSearch, page: userPage, sort: userSort, dir: userDir, limit: 50 },
    immediate: true,
    lazy: true,
    watch: [userPage, userSort, userDir],
})

const debouncedUserSearch = refDebounced(userSearch, 300)

watch(debouncedUserSearch, () => {
    userPage.value = 1
    refreshUsers()
})

const toggleUserSort = (col: string) => {
    if (userSort.value === col) {
        userDir.value = userDir.value === 'desc' ? 'asc' : 'desc'
    } else {
        userSort.value = col
        userDir.value = 'desc'
    }
}

// The parent keeps this component alive (<KeepAlive>) across tab switches.
// Matching the old tab-switch watcher: on re-activation, refetch only if no
// data ever loaded (e.g. the first fetch errored).
let activatedOnce = false
onActivated(() => {
    if (!activatedOnce) { activatedOnce = true; return }
    if (!usersData.value) refreshUsers()
})

// Helpers


</script>

<template>
    <!-- ═══════════════ USERS ═══════════════ -->
    <div class="space-y-4">
        <!-- Search -->
        <div class="glass-panel p-4">
            <div class="flex items-center gap-3">
                <Icon name="lucide:search" class="text-gray-500" />
                <input
                    v-model="userSearch"
                    type="text"
                    placeholder="Search by name or character ID..."
                    class="flex-1 bg-transparent text-sm text-white placeholder-gray-600 outline-none"
                >
                <span v-if="usersData" class="text-xs text-gray-500 tabular-nums">{{ fmt(usersData.total) }} users</span>
            </div>
        </div>

        <!-- User table -->
        <div class="glass-panel overflow-hidden">
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead>
                        <tr class="border-b border-white/[0.06]">
                            <th class="text-left px-4 py-3 text-fine font-bold uppercase tracking-[0.15em] text-gray-500 cursor-pointer hover:text-blue-400" @click="toggleUserSort('character_name')">
                                Character
                                <Icon v-if="userSort === 'character_name'" :name="userDir === 'asc' ? 'lucide:arrow-up' : 'lucide:arrow-down'" class="text-xs ml-1 inline" />
                            </th>
                            <th class="text-left px-4 py-3 text-fine font-bold uppercase tracking-[0.15em] text-gray-500 cursor-pointer hover:text-blue-400" @click="toggleUserSort('last_login')">
                                Last Login
                                <Icon v-if="userSort === 'last_login'" :name="userDir === 'asc' ? 'lucide:arrow-up' : 'lucide:arrow-down'" class="text-xs ml-1 inline" />
                            </th>
                            <th class="text-left px-4 py-3 text-fine font-bold uppercase tracking-[0.15em] text-gray-500 cursor-pointer hover:text-blue-400" @click="toggleUserSort('created_at')">
                                Joined
                                <Icon v-if="userSort === 'created_at'" :name="userDir === 'asc' ? 'lucide:arrow-up' : 'lucide:arrow-down'" class="text-xs ml-1 inline" />
                            </th>
                            <th class="text-center px-4 py-3 text-fine font-bold uppercase tracking-[0.15em] text-gray-500 w-[60px]">Admin</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr
                            v-for="u in usersData?.users" :key="u.character_id"
                            class="border-b border-white/[0.04] hover:bg-blue-500/[0.04] cursor-pointer transition-colors"
                            @click="navigateTo(`/admin/users/${u.character_id}`)"
                        >
                            <td class="px-4 py-2.5">
                                <div class="flex items-center gap-2.5 text-gray-300">
                                    <img :src="`/images/characters/${u.character_id}/portrait?size=32`" class="w-6 h-6 rounded-full" loading="lazy">
                                    {{ u.character_name }}
                                </div>
                            </td>
                            <td class="px-4 py-2.5 text-gray-500 tabular-nums text-xs">{{ timeSince(u.last_login) }}</td>
                            <td class="px-4 py-2.5 text-gray-500 tabular-nums text-xs">{{ fmtDateShort(u.created_at) }}</td>
                            <td class="px-4 py-2.5 text-center">
                                <span v-if="u.is_admin" class="inline-block w-2 h-2 rounded-full bg-green-400"></span>
                                <span v-else class="inline-block w-2 h-2 rounded-full bg-gray-700"></span>
                            </td>
                        </tr>
                        <tr v-if="!usersData?.users?.length">
                            <td colspan="4" class="px-4 py-8 text-center text-gray-600">No users found.</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Pagination -->
        <div v-if="usersData && usersData.pages > 1" class="flex items-center justify-between">
            <button
                class="px-3 py-1.5 rounded text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                :disabled="userPage <= 1"
                @click="userPage--"
            >Previous</button>
            <span class="text-xs text-gray-500 tabular-nums">Page {{ userPage }} of {{ usersData.pages }}</span>
            <button
                class="px-3 py-1.5 rounded text-sm text-gray-400 hover:text-blue-400 hover:bg-blue-500/[0.06] disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                :disabled="userPage >= usersData.pages"
                @click="userPage++"
            >Next</button>
        </div>

    </div>
</template>
