export interface Toast {
    id: number
    message: string
    type: 'success' | 'error' | 'info'
}

let _next = 0

export function useToast() {
    const toasts = useState<Toast[]>('toasts', () => [])

    function add(message: string, type: Toast['type'] = 'info', durationMs = 4000) {
        const id = ++_next
        toasts.value = [...toasts.value, { id, message, type }]
        if (durationMs > 0) {
            setTimeout(() => remove(id), durationMs)
        }
        return id
    }

    function remove(id: number) {
        toasts.value = toasts.value.filter(t => t.id !== id)
    }

    return {
        toasts: readonly(toasts),
        add,
        success: (m: string, d?: number) => add(m, 'success', d),
        error: (m: string, d?: number) => add(m, 'error', d),
        info: (m: string, d?: number) => add(m, 'info', d),
        remove,
    }
}
