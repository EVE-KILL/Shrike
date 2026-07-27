<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'

/**
 * Bottom-right killboard switcher — the "Powered by EVE-KILL" badge grown into
 * a shortlist.
 *
 * Collapsed it is only the E, on every host including the main site. Opening it
 * folds a panel up: the boards that track you or that you pinned, then a row to
 * pin the board you are standing on, with the E itself still the click-through
 * to eve-kill.com.
 *
 * Nothing loads until it is first opened, so a visitor who never touches it
 * never pays for the request.
 */
const { isDomainMode } = useDomainConfig()
const { boards, current, authenticated, loading, loaded, load, pin, remove, maxEntries } = useBoards()

const open = ref(false)
const rootRef = ref<HTMLElement | null>(null)
let closeTimer: ReturnType<typeof setTimeout> | null = null

const cancelClose = () => {
    if (closeTimer) { clearTimeout(closeTimer); closeTimer = null }
}

const show = () => {
    cancelClose()
    open.value = true
    void load()
}

// Small grace period: the panel sits above the badge, and a diagonal cursor
// path between them briefly leaves both.
const scheduleClose = () => {
    cancelClose()
    closeTimer = setTimeout(() => { open.value = false }, 220)
}

onUnmounted(cancelClose)
onClickOutside(rootRef, () => { open.value = false })

const canPinCurrent = computed(() => !!current.value && !current.value.listed)
const showEmptyHint = computed(() =>
    loaded.value && !loading.value && boards.value.length === 0 && !canPinCurrent.value)
</script>

<template>
    <div
        ref="rootRef"
        class="board-switcher"
        @mouseenter="show"
        @mouseleave="scheduleClose"
    >
        <!-- Panel, above the badge. Auth and cookie only resolve on the client,
             so the list never renders during SSR. -->
        <ClientOnly>
            <div v-if="open" class="board-panel" @mouseenter="cancelClose" @mouseleave="scheduleClose">
                <div v-if="loading && !loaded" class="board-row board-row--muted">
                    <Icon name="lucide:loader-2" class="text-fine animate-spin" />
                    Loading…
                </div>

                <a
                    v-for="board in boards"
                    :key="board.key"
                    :href="board.url"
                    class="board-row board-row--link"
                >
                    <Icon
                        :name="board.tracked ? 'lucide:crosshair' : 'lucide:pin'"
                        class="text-fine flex-shrink-0"
                        :class="board.tracked ? 'text-blue-400/80' : 'text-gray-500'"
                    />
                    <span class="board-row-name">{{ board.name }}</span>
                    <button
                        class="board-row-remove"
                        :aria-label="`Remove ${board.name}`"
                        v-tooltip="board.tracked ? 'Hide this board from your list' : 'Unpin this board'"
                        @click.stop.prevent="remove(board.key)"
                    >
                        <Icon name="lucide:x" class="text-fine" />
                    </button>
                </a>

                <button
                    v-if="canPinCurrent"
                    class="board-row board-row--action"
                    :disabled="boards.length >= maxEntries"
                    @click="pin(current!.key)"
                >
                    <Icon name="lucide:plus" class="text-fine flex-shrink-0" />
                    <span class="board-row-name">
                        {{ boards.length >= maxEntries ? `List is full (${maxEntries})` : `Add ${current!.name}` }}
                    </span>
                </button>

                <div v-if="showEmptyHint" class="board-row board-row--muted">
                    {{ authenticated
                        ? 'No killboards track you yet'
                        : 'Sign in to see boards that track you' }}
                </div>
            </div>
        </ClientOnly>

        <a
            href="https://eve-kill.com"
            :target="isDomainMode ? '_blank' : undefined"
            :rel="isDomainMode ? 'noopener' : undefined"
            class="powered-badge"
            aria-label="Powered by EVE-KILL"
        >
            <span class="powered-badge-icon">E</span>
            <span class="powered-badge-text">Powered by EVE-KILL</span>
        </a>
    </div>
</template>

<style scoped>
.board-switcher {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.375rem;
}

.board-panel {
    display: flex;
    flex-direction: column;
    min-width: 12rem;
    max-width: min(18rem, calc(100vw - 2rem));
    max-height: 60vh;
    overflow-y: auto;
    border-radius: 0.5rem;
    background: rgba(0, 0, 0, 0.85);
    backdrop-filter: blur(12px);
    border: 1px solid rgba(255, 255, 255, 0.08);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
    padding: 0.25rem;
}

.board-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.4rem 0.5rem;
    border-radius: 0.375rem;
    font-size: 12px;
    text-align: left;
    text-decoration: none;
    color: rgb(209, 213, 219);
    transition: background 0.15s, color 0.15s;
}
.board-row--link:hover,
.board-row--action:hover:not(:disabled) {
    background: rgba(59, 130, 246, 0.1);
    color: rgb(147, 197, 253);
}
.board-row--action {
    color: rgb(156, 163, 175);
    cursor: pointer;
}
.board-row--action:disabled {
    opacity: 0.5;
    cursor: default;
}
.board-row--muted {
    color: rgb(107, 114, 128);
    font-size: 11px;
}

.board-row-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.board-row-remove {
    flex-shrink: 0;
    padding: 0.125rem;
    border-radius: 0.25rem;
    color: rgb(107, 114, 128);
    opacity: 0;
    cursor: pointer;
    transition: opacity 0.15s, color 0.15s, background 0.15s;
}
.board-row:hover .board-row-remove {
    opacity: 1;
}
.board-row-remove:hover {
    color: rgb(248, 113, 113);
    background: rgba(239, 68, 68, 0.12);
}

/* Unchanged from the original badge — the E still widens on hover and still
   links to eve-kill.com. */
.powered-badge {
    display: flex;
    align-items: center;
    height: 2.25rem;
    border-radius: 0.5rem;
    background: rgba(0, 0, 0, 0.5);
    text-decoration: none;
    overflow: hidden;
    transition: background 0.2s, color 0.2s;
    color: rgb(156, 163, 175);
}
.powered-badge:hover {
    background: rgba(0, 0, 0, 0.7);
    color: rgb(96, 165, 250);
}
.powered-badge-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.25rem;
    height: 2.25rem;
    flex-shrink: 0;
    font-size: 13px;
    font-weight: 800;
}
.powered-badge-text {
    max-width: 0;
    overflow: hidden;
    white-space: nowrap;
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    padding-right: 0;
    opacity: 0;
    transition: max-width 0.3s ease, padding-right 0.3s ease, opacity 0.2s ease;
}
.board-switcher:hover .powered-badge-text {
    max-width: 10rem;
    padding-right: 0.75rem;
    opacity: 1;
}
</style>
