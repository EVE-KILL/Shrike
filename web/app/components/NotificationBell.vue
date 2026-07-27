<script setup lang="ts">
/**
 * NotificationBell — bell icon + dropdown shown in the navbar when the user
 * is logged in. Surfaces direct replies to the user's comments.
 *
 * State (items, unread count, read cursor) lives in the shared
 * `useNotifications` composable, which manages the global WS connection and
 * the `/api/notifications/replies` backfill.
 */

const open = ref(false)
// Destructure so the template auto-unwraps the refs (Vue only auto-unwraps
// top-level setup-scope refs, not ref properties of plain objects).
const { items, unreadCount, isUnread, markAllRead } = useNotifications()
const router = useRouter()

const TARGET_META: Record<number, { label: string; icon: string; color: string; pathFn: (id: number, commentId: number) => string }> = {
    1: { label: 'Killmail', icon: 'lucide:swords',   color: 'text-red-400',     pathFn: (id, c) => `/kill/${id}#comment-${c}` },
    2: { label: 'Character',icon: 'lucide:user',     color: 'text-blue-400',    pathFn: (id, c) => `/character/${id}#comment-${c}` },
    3: { label: 'Corp',     icon: 'lucide:building', color: 'text-amber-400',   pathFn: (id, c) => `/corporation/${id}#comment-${c}` },
    4: { label: 'Alliance', icon: 'lucide:flag',     color: 'text-purple-400',  pathFn: (id, c) => `/alliance/${id}#comment-${c}` },
    5: { label: 'System',   icon: 'lucide:globe',    color: 'text-emerald-400', pathFn: (id, c) => `/system/${id}#comment-${c}` },
    7: { label: 'Battle',   icon: 'lucide:shield',   color: 'text-orange-400',  pathFn: (id, c) => `/battle/${id}/comments#comment-${c}` },
}

function targetFor(targetType: number) {
    return TARGET_META[targetType] ?? { label: 'Page', icon: 'lucide:link', color: 'text-gray-400', pathFn: () => '#' }
}

function timeAgo(iso: string): string {
    const d = new Date(iso).getTime()
    const diff = Date.now() - d
    if (diff < 60_000) return 'just now'
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m`
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h`
    if (diff < 30 * 86_400_000) return `${Math.floor(diff / 86_400_000)}d`
    return new Date(iso).toLocaleDateString()
}

/** Strip HTML for the snippet preview. */
function snippet(html: string, max = 120): string {
    const txt = html.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()
    return txt.length > max ? txt.slice(0, max - 1) + '…' : txt
}

function navigate(item: { id: number; target_type: number; target_id: number }) {
    const path = targetFor(item.target_type).pathFn(item.target_id, item.id)
    open.value = false
    router.push(path)
}

function onMarkAllRead() {
    markAllRead()
}

// Close dropdown on route change so it doesn't linger over a new page.
router.afterEach(() => { open.value = false })

const iconBtn = 'flex items-center justify-center w-9 h-9 rounded-md text-white/60 hover:bg-blue-500/10 hover:text-blue-400 transition-colors relative'
</script>

<template>
    <Dropdown v-model="open" align="right">
        <template #trigger>
            <button :class="iconBtn" aria-label="Notifications">
                <Icon name="lucide:bell" class="text-lg" />
                <span
                    v-if="unreadCount > 0"
                    class="absolute -top-0.5 -right-0.5 inline-flex items-center justify-center min-w-[1rem] h-4 px-1 rounded-full text-fine font-bold text-white bg-red-500 ring-2 ring-black/80 tabular-nums"
                >
                    {{ unreadCount > 99 ? '99+' : unreadCount }}
                </span>
            </button>
        </template>

        <template #default>
            <div class="w-[360px] max-h-[480px] flex flex-col">
                <!-- Header -->
                <div class="flex items-center justify-between px-3 py-2 border-b border-white/[0.08]">
                    <div class="flex items-center gap-2">
                        <Icon name="lucide:bell" class="text-blue-400 text-sm" />
                        <span class="text-sm font-semibold text-white">Notifications</span>
                        <span
                            v-if="unreadCount > 0"
                            class="text-fine font-semibold text-blue-300 bg-blue-500/[0.12] border border-blue-500/20 rounded-full px-1.5 py-px"
                        >
                            {{ unreadCount }} new
                        </span>
                    </div>
                    <button
                        v-if="unreadCount > 0"
                        class="text-xs text-gray-400 hover:text-blue-400 cursor-pointer transition-colors"
                        @click="onMarkAllRead"
                    >
                        Mark all read
                    </button>
                </div>

                <!-- List -->
                <div class="overflow-y-auto flex-1">
                    <div v-if="items.length === 0" class="px-4 py-10 text-center">
                        <Icon name="lucide:bell-off" class="text-3xl text-gray-600 mb-2 inline-block" />
                        <p class="text-sm text-gray-500">No notifications yet</p>
                        <p class="text-xs text-gray-600 mt-1">Replies to your comments will appear here.</p>
                    </div>

                    <button
                        v-for="item in items"
                        :key="item.id"
                        class="w-full text-left px-3 py-2.5 border-b border-white/[0.04] last:border-b-0 hover:bg-blue-500/[0.06] transition-colors cursor-pointer relative"
                        :class="isUnread(item) ? 'bg-blue-500/[0.03]' : ''"
                        @click="navigate(item)"
                    >
                        <!-- Unread dot -->
                        <span
                            v-if="isUnread(item)"
                            class="absolute left-1 top-1/2 -translate-y-1/2 w-1.5 h-1.5 rounded-full bg-blue-400"
                        />
                        <div class="flex items-start gap-2.5 pl-3">
                            <EveImage
                                :src="`/images/characters/${item.character_id}/portrait?size=64`"
                                :size="32"
                                :alt="item.character_name"
                                class="w-8 h-8 rounded-md ring-1 ring-white/[0.08] flex-shrink-0"
                            />
                            <div class="flex-1 min-w-0">
                                <div class="flex items-center gap-1.5 text-fine">
                                    <span class="text-white font-semibold truncate">{{ item.character_name }}</span>
                                    <span class="text-gray-600">replied</span>
                                    <span
                                        class="inline-flex items-center gap-1 px-1.5 py-px rounded text-fine font-medium"
                                        :class="targetFor(item.target_type).color"
                                    >
                                        <Icon :name="targetFor(item.target_type).icon" class="text-fine" />
                                        {{ targetFor(item.target_type).label }}
                                    </span>
                                    <span class="ml-auto text-gray-600 tabular-nums">{{ timeAgo(item.created_at) }}</span>
                                </div>
                                <p class="text-xs text-gray-300 mt-1 line-clamp-2">{{ snippet(item.body_html) }}</p>
                                <p
                                    v-if="item.parent_snippet"
                                    class="text-fine text-gray-500 mt-1 italic line-clamp-1"
                                    v-tooltip="item.parent_snippet"
                                >
                                    in reply to: {{ item.parent_snippet }}
                                </p>
                            </div>
                        </div>
                    </button>
                </div>

                <!-- Footer -->
                <div
                    v-if="items.length > 0"
                    class="px-3 py-2 border-t border-white/[0.08] text-center"
                >
                    <NuxtLink
                        to="/comments"
                        class="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                        @click="open = false"
                    >
                        See all comments →
                    </NuxtLink>
                </div>
            </div>
        </template>
    </Dropdown>
</template>
