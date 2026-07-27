/**
 * Global spotlight search state and keyboard shortcuts (Cmd/Ctrl+K).
 * Uses a single global listener to avoid duplicate registrations.
 */
const _isOpen = ref(false)
let _listenerRegistered = false

const _isTypingInField = (event: KeyboardEvent): boolean => {
    const target = event.target as HTMLElement
    return (
        target?.tagName === 'INPUT' ||
        target?.tagName === 'TEXTAREA' ||
        target?.contentEditable === 'true'
    )
}

const _handleKeyboard = (event: KeyboardEvent) => {
    // Cmd/Ctrl + K to toggle search
    if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault()
        event.stopPropagation()
        if (!_isTypingInField(event)) {
            _isOpen.value = !_isOpen.value
        }
        return
    }

    // Escape to close search
    if (event.key === 'Escape' && _isOpen.value) {
        event.stopPropagation()
        _isOpen.value = false
    }
}

export const useSpotlightSearch = () => {
    const openSearch = () => { _isOpen.value = true }
    const closeSearch = () => { _isOpen.value = false }

    onMounted(() => {
        if (import.meta.client && !_listenerRegistered) {
            document.addEventListener('keydown', _handleKeyboard)
            _listenerRegistered = true
        }
    })

    return {
        isOpen: readonly(_isOpen),
        openSearch,
        closeSearch,
    }
}
