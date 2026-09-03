<script setup lang="ts">
import type { NotificationItem } from '~/composables/useNotifications'
import type { TrackerNotification } from '~/composables/useTrackerNotifications'

type BellRow =
    | { kind: 'reply'; created_at: string; item: NotificationItem }
    | { kind: 'tracker'; created_at: string; item: TrackerNotification }

const open = ref(false)
const {
    items: replyItems,
    unreadCount: replyUnread,
    isUnread: isReplyUnread,
    markAllRead: markRepliesRead,
} = useNotifications()
const {
    notifications: trackerItems,
    unreadCount: trackerUnread,
    markAllRead: markTrackersRead,
    refreshNotifications: refreshTrackerNotifications,
} = useTrackerNotifications()
const router = useRouter()

const unreadCount = computed(() => replyUnread.value + trackerUnread.value)
const rows = computed<BellRow[]>(() => [
    ...replyItems.value.map(item => ({ kind: 'reply' as const, created_at: item.created_at, item })),
    ...trackerItems.value.map(item => ({ kind: 'tracker' as const, created_at: item.created_at, item })),
].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()))

const TARGET_META: Record<number, { label: string; icon: string; color: string; pathFn: (id: number, commentId: number) => string }> = {
    1: { label: 'Killmail', icon: 'lucide:swords', color: 'text-red-400', pathFn: (id, c) => `/kill/${id}#comment-${c}` },
    2: { label: 'Character', icon: 'lucide:user', color: 'text-blue-400', pathFn: (id, c) => `/character/${id}#comment-${c}` },
    3: { label: 'Corp', icon: 'lucide:building', color: 'text-amber-400', pathFn: (id, c) => `/corporation/${id}#comment-${c}` },
    4: { label: 'Alliance', icon: 'lucide:flag', color: 'text-purple-400', pathFn: (id, c) => `/alliance/${id}#comment-${c}` },
    5: { label: 'System', icon: 'lucide:globe', color: 'text-emerald-400', pathFn: (id, c) => `/system/${id}#comment-${c}` },
    7: { label: 'Battle', icon: 'lucide:shield', color: 'text-orange-400', pathFn: (id, c) => `/battle/${id}/comments#comment-${c}` },
}

function targetFor(targetType: number) {
    return TARGET_META[targetType] ?? { label: 'Page', icon: 'lucide:link', color: 'text-gray-400', pathFn: () => '#' }
}

function timeAgo(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime()
    if (diff < 60_000) return 'just now'
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m`
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h`
    if (diff < 30 * 86_400_000) return `${Math.floor(diff / 86_400_000)}d`
    return new Date(iso).toLocaleDateString()
}

function snippet(html: string, max = 120): string {
    const txt = html.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()
    return txt.length > max ? `${txt.slice(0, max - 1)}…` : txt
}

function trackerSummary(item: TrackerNotification): string {
    const ship = item.ship_name || 'Ship'
    const victim = item.victim_character_name ? ` piloted by ${item.victim_character_name}` : ''
    const system = item.solar_system_name ? ` in ${item.solar_system_name}` : ''
    return `${ship}${victim}${system}`
}

function trackerRole(item: TrackerNotification): string {
    if (item.match_role === 'victim') return 'loss'
    if (item.match_role === 'attacker') return 'kill'
    if (item.match_role === 'both') return 'both sides'
    return 'location'
}

function rowUnread(row: BellRow): boolean {
    return row.kind === 'reply' ? isReplyUnread(row.item) : !row.item.is_read
}

function navigate(row: BellRow) {
    open.value = false
    if (row.kind === 'tracker') {
        void router.push(`/kill/${row.item.killmail_id}`)
        return
    }
    void router.push(targetFor(row.item.target_type).pathFn(row.item.target_id, row.item.id))
}

function onMarkAllRead() {
    void Promise.allSettled([markRepliesRead(), markTrackersRead()])
}

watch(open, value => {
    if (value) void refreshTrackerNotifications()
})
router.afterEach(() => { open.value = false })

const iconBtn = 'flex items-center justify-center w-9 h-9 rounded-md text-white/60 hover:bg-blue-500/10 hover:text-blue-400 transition-colors relative'
</script>

<template>
    <Dropdown v-model="open" align="right">
        <template #trigger>
            <button :class="iconBtn" aria-label="Notifications">
                <Icon name="lucide:bell" class="text-lg" />
                <span v-if="unreadCount > 0" class="absolute -right-0.5 -top-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-red-500 px-1 text-fine font-bold tabular-nums text-white ring-2 ring-black/80">
                    {{ unreadCount > 99 ? '99+' : unreadCount }}
                </span>
            </button>
        </template>

        <template #default>
            <div class="flex max-h-[500px] w-[380px] max-w-[calc(100vw-1rem)] flex-col">
                <div class="flex items-center justify-between border-b border-white/[0.08] px-3 py-2">
                    <div class="flex items-center gap-2">
                        <Icon name="lucide:bell" class="text-sm text-blue-400" />
                        <span class="text-sm font-semibold text-white">Notifications</span>
                        <span v-if="unreadCount > 0" class="rounded-full border border-blue-500/20 bg-blue-500/[0.12] px-1.5 py-px text-fine font-semibold text-blue-300">
                            {{ unreadCount }} new
                        </span>
                    </div>
                    <button v-if="unreadCount > 0" class="cursor-pointer text-xs text-gray-400 transition-colors hover:text-blue-400" @click="onMarkAllRead">
                        Mark all read
                    </button>
                </div>

                <div class="flex-1 overflow-y-auto">
                    <div v-if="rows.length === 0" class="px-4 py-10 text-center">
                        <Icon name="lucide:bell-off" class="mb-2 inline-block text-3xl text-gray-600" />
                        <p class="text-sm text-gray-500">No notifications yet</p>
                        <p class="mt-1 text-xs text-gray-600">Comment replies and opted-in tracker alerts will appear here.</p>
                    </div>

                    <button
                        v-for="row in rows"
                        :key="`${row.kind}-${row.item.id}`"
                        class="relative w-full cursor-pointer border-b border-white/[0.04] px-3 py-2.5 text-left transition-colors last:border-b-0 hover:bg-blue-500/[0.06]"
                        :class="rowUnread(row) ? 'bg-blue-500/[0.03]' : ''"
                        @click="navigate(row)"
                    >
                        <span v-if="rowUnread(row)" class="absolute left-1 top-1/2 h-1.5 w-1.5 -translate-y-1/2 rounded-full bg-blue-400" />

                        <div v-if="row.kind === 'reply'" class="flex items-start gap-2.5 pl-3">
                            <EveImage :src="`/images/characters/${row.item.character_id}/portrait?size=64`" :size="32" :alt="row.item.character_name" class="h-8 w-8 flex-shrink-0 rounded-md ring-1 ring-white/[0.08]" />
                            <div class="min-w-0 flex-1">
                                <div class="flex items-center gap-1.5 text-fine">
                                    <span class="truncate font-semibold text-white">{{ row.item.character_name }}</span>
                                    <span class="text-gray-600">replied</span>
                                    <span class="inline-flex items-center gap-1 rounded px-1.5 py-px text-fine font-medium" :class="targetFor(row.item.target_type).color">
                                        <Icon :name="targetFor(row.item.target_type).icon" class="text-fine" />
                                        {{ targetFor(row.item.target_type).label }}
                                    </span>
                                    <span class="ml-auto tabular-nums text-gray-600">{{ timeAgo(row.created_at) }}</span>
                                </div>
                                <p class="mt-1 line-clamp-2 text-xs text-gray-300">{{ snippet(row.item.body_html) }}</p>
                                <p v-if="row.item.parent_snippet" class="mt-1 line-clamp-1 text-fine italic text-gray-500" v-tooltip="row.item.parent_snippet">
                                    in reply to: {{ row.item.parent_snippet }}
                                </p>
                            </div>
                        </div>

                        <div v-else class="flex items-start gap-2.5 pl-3">
                            <EveImage v-if="row.item.ship_type_id" :src="`/images/types/${row.item.ship_type_id}/icon?size=64`" :size="32" :alt="row.item.ship_name || 'Destroyed ship'" class="h-8 w-8 flex-shrink-0 rounded-md ring-1 ring-white/[0.08]" />
                            <span v-else class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md bg-red-500/10 text-red-400">
                                <Icon name="lucide:crosshair" />
                            </span>
                            <div class="min-w-0 flex-1">
                                <div class="flex items-center gap-1.5 text-fine">
                                    <span class="truncate font-semibold text-white">{{ row.item.target_name }}</span>
                                    <span class="rounded bg-amber-500/10 px-1.5 py-px capitalize text-amber-300">{{ trackerRole(row.item) }}</span>
                                    <span class="ml-auto tabular-nums text-gray-600">{{ timeAgo(row.created_at) }}</span>
                                </div>
                                <p class="mt-1 line-clamp-2 text-xs text-gray-300">{{ trackerSummary(row.item) }}</p>
                                <p class="mt-1 text-fine tabular-nums text-gray-500">{{ formatIsk(row.item.total_value) }} ISK</p>
                            </div>
                        </div>
                    </button>
                </div>

                <div v-if="rows.length > 0" class="flex items-center justify-center gap-4 border-t border-white/[0.08] px-3 py-2">
                    <NuxtLink to="/comments" class="text-xs text-blue-400 transition-colors hover:text-blue-300" @click="open = false">Comments</NuxtLink>
                    <span class="text-gray-700">·</span>
                    <NuxtLink to="/trackers" class="text-xs text-blue-400 transition-colors hover:text-blue-300" @click="open = false">Manage trackers</NuxtLink>
                </div>
            </div>
        </template>
    </Dropdown>
</template>
