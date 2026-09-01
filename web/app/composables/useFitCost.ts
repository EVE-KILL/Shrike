/**
 * Reactive ISK cost of the current fit (hull + modules + loaded
 * charges + drones × quantity).
 *
 * The composable watches the committed fit (not the hover preview —
 * hovering shouldn't make the cost flicker). When the type set
 * changes, it fetches /api/prices/bulk for any uncached type_ids and
 * recomputes the total. Already-priced types stay cached in a
 * module-level Map so module-swap-heavy editing sessions stop hitting
 * the network entirely.
 *
 * Cargo is intentionally excluded — that's loot/extras, not the cost
 * of the fitted ship.
 */

import type { Fit } from './fit/types'

const priceCache = new Map<number, number>()
const inflight = new Map<number, Promise<void>>()

async function ensurePrices(typeIds: number[]): Promise<void> {
    const missing = typeIds.filter((t) => !priceCache.has(t) && !inflight.has(t))
    if (missing.length === 0) {
        // Wait on any in-flight fetches that cover the requested ids.
        const waits = typeIds
            .map((t) => inflight.get(t))
            .filter((p): p is Promise<void> => p !== undefined)
        if (waits.length > 0) await Promise.all(waits)
        return
    }

    const promise = apiFetch<{ prices: Record<number, number> }>('/api/prices/bulk', {
        params: { types: missing.join(',') },
    })
        .then((res) => {
            for (const t of missing) {
                // Store 0 for absent keys too — that locks "we asked, no
                // price exists yet" into the cache so we don't keep
                // re-fetching on every render.
                priceCache.set(t, Number(res.prices[t] ?? 0))
            }
        })
        .catch(() => {
            // Network blip — clear inflight markers so a later call retries.
            for (const t of missing) inflight.delete(t)
        })
        .finally(() => {
            for (const t of missing) inflight.delete(t)
        })

    for (const t of missing) inflight.set(t, promise)
    await promise
}

function collectTypeIds(fit: Fit | null): number[] {
    if (!fit) return []
    const set = new Set<number>()
    if (fit.shipTypeId > 0) set.add(fit.shipTypeId)
    for (const m of fit.modules) {
        if (m.typeId > 0) set.add(m.typeId)
        if (m.charge?.typeId && m.charge.typeId > 0) set.add(m.charge.typeId)
    }
    for (const d of fit.drones) {
        if (d.typeId > 0) set.add(d.typeId)
    }
    return Array.from(set)
}

function computeTotal(fit: Fit | null): number {
    if (!fit) return 0
    let total = 0
    if (fit.shipTypeId > 0) total += priceCache.get(fit.shipTypeId) ?? 0
    for (const m of fit.modules) {
        if (m.typeId > 0) total += priceCache.get(m.typeId) ?? 0
        if (m.charge?.typeId && m.charge.typeId > 0) {
            total += priceCache.get(m.charge.typeId) ?? 0
        }
    }
    for (const d of fit.drones) {
        if (d.typeId <= 0) continue
        const qty = (d.states.Passive ?? 0) + (d.states.Active ?? 0)
        total += (priceCache.get(d.typeId) ?? 0) * qty
    }
    return total
}

export async function calculateFitCost(fit: Fit | null): Promise<number> {
    await ensurePrices(collectTypeIds(fit))
    return computeTotal(fit)
}

// Stub refs returned from the server. Sharing one set of readonly refs
// across SSR avoids per-request allocations and guarantees the server
// never tries to fetch prices or mutate the shared `priceCache` map.
// The client side instantiates its own refs on hydration.
const SSR_STUB_TOTAL = readonly(ref(0))
const SSR_STUB_LOADING = readonly(ref(false))
const SSR_STUB_HAS_PRICE = readonly(ref(false))

export function useFitCost() {
    if (import.meta.server) {
        return {
            total: SSR_STUB_TOTAL,
            isLoading: SSR_STUB_LOADING,
            hasPrice: SSR_STUB_HAS_PRICE,
            refresh: async () => {},
        }
    }

    const { currentFit } = useCurrentFit()

    const total = ref(0)
    const isLoading = ref(false)
    const hasPrice = ref(false)

    let fetchVersion = 0

    async function refresh() {
        const fit = currentFit.value
        const typeIds = collectTypeIds(fit)
        if (typeIds.length === 0) {
            total.value = 0
            hasPrice.value = false
            return
        }
        const myVersion = ++fetchVersion
        const needsFetch = typeIds.some((t) => !priceCache.has(t))
        if (needsFetch) {
            isLoading.value = true
            await ensurePrices(typeIds)
            // Bail if a newer refresh started while we were waiting.
            if (myVersion !== fetchVersion) return
            isLoading.value = false
        }
        total.value = computeTotal(fit)
        hasPrice.value = total.value > 0
    }

    watch(
        () => currentFit.value,
        () => {
            refresh()
        },
        { immediate: true },
    )

    return { total, isLoading, hasPrice, refresh }
}
