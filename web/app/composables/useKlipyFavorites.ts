/**
 * Persistent localStorage favorites for the Klipy GIF picker.
 *
 * Stored under `ek_klipy_favs` as a JSON array of GIF objects so the picker
 * can render the favorites tab without re-querying the API.
 *
 * The same favorite added twice (same id) is deduplicated. Newest favorites
 * appear first.
 */

export interface KlipyFavorite {
    id: number
    title: string
    /** Picker thumbnail (animated webp/gif). */
    preview_url: string
    /** Full-size mp4 — what gets inserted into the comment markdown. */
    insert_url: string
    width: number
    height: number
    added_at: number
}

const STORAGE_KEY = 'ek_klipy_favs'
const MAX_FAVORITES = 100

function readFromStorage(): KlipyFavorite[] {
    if (!import.meta.client) return []
    try {
        const raw = localStorage.getItem(STORAGE_KEY)
        if (!raw) return []
        const parsed = JSON.parse(raw)
        return Array.isArray(parsed) ? parsed.filter((x) => x && typeof x.id === 'number') : []
    } catch {
        return []
    }
}

function writeToStorage(list: KlipyFavorite[]) {
    if (!import.meta.client) return
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(list))
    } catch {
        // quota exceeded — silently drop
    }
}

// Single shared reactive list across all consumers (mounted at module level
// so a favorite added in one picker instance is visible in another).
const favorites = ref<KlipyFavorite[]>([])
let _hydrated = false

function hydrate() {
    if (_hydrated || !import.meta.client) return
    favorites.value = readFromStorage()
    _hydrated = true
}

export function useKlipyFavorites() {
    hydrate()

    function isFavorite(id: number): boolean {
        return favorites.value.some((f) => f.id === id)
    }

    function add(fav: Omit<KlipyFavorite, 'added_at'>) {
        if (isFavorite(fav.id)) return
        const next = [{ ...fav, added_at: Date.now() }, ...favorites.value]
        if (next.length > MAX_FAVORITES) next.length = MAX_FAVORITES
        favorites.value = next
        writeToStorage(next)
    }

    function remove(id: number) {
        const next = favorites.value.filter((f) => f.id !== id)
        favorites.value = next
        writeToStorage(next)
    }

    function toggle(fav: Omit<KlipyFavorite, 'added_at'>) {
        if (isFavorite(fav.id)) remove(fav.id)
        else add(fav)
    }

    return {
        favorites: readonly(favorites),
        isFavorite,
        add,
        remove,
        toggle,
    }
}
