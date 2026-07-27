/**
 * Global keyboard shortcuts with ? to show help modal.
 * Uses a single global listener to avoid duplicate registrations.
 */
const _showHelp = ref(false)
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
    if (_isTypingInField(event)) return

    // ? to toggle keyboard shortcuts help
    if (event.key === '?') {
        event.preventDefault()
        event.stopPropagation()
        _showHelp.value = !_showHelp.value
        return
    }

    // Escape to close help
    if (event.key === 'Escape' && _showHelp.value) {
        event.stopPropagation()
        _showHelp.value = false
    }
}

export const useKeyboardShortcuts = () => {
    onMounted(() => {
        if (import.meta.client && !_listenerRegistered) {
            document.addEventListener('keydown', _handleKeyboard)
            _listenerRegistered = true
        }
    })

    return {
        showHelp: _showHelp,
        closeHelp: () => { _showHelp.value = false },
    }
}
