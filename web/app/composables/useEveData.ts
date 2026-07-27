/**
 * Exposes the loaded SDE as a reactive ref for components that need to
 * resolve attribute/type names to IDs at runtime.
 *
 * Most components won't need this — they read their values off the stats
 * ref returned by `useFitStatistics`, whose keys are already resolved IDs
 * from the engine callbacks. This composable exists for the cases where
 * the UI itself needs a lookup, e.g.:
 *
 *   - `ShipAttribute` doing `sde.attributeNameToId.get("hiSlots")` to fish
 *     a specific stat out of `hull.attributes`
 *   - `HullListing` / `HardwareListing` walking `sde.marketGroups` to build
 *     the left-sidebar tree
 *
 * `loadSde()` caches internally in @evekill/dogma — calling it multiple
 * times is cheap after the first. The first call is what triggers the
 * SDE protobuf fetch, so this composable is a no-op until something
 * actually mounts on the page that uses it.
 */

import { loadSde, type SdeData } from "@evekill/dogma";

export function useEveData() {
    const sde = useState<SdeData | null>("fit:sde", () => null);
    const error = useState<Error | null>("fit:sde:error", () => null);

    // Kick off the load on first call per page. `loadSde` itself is
    // idempotent, so repeated calls from multiple components are free.
    //
    // CRITICAL: wrap the SDE with markRaw before storing. It contains
    // four Maps with ~30k entries each plus nested type/attribute
    // objects; if Vue wraps it reactively, every `.get()` goes through
    // a Proxy and the first render stalls for 10+ seconds. We only
    // need identity-level reactivity (null → loaded) which the ref
    // itself provides.
    if (import.meta.client && sde.value === null && error.value === null) {
        loadSde()
            .then((data) => {
                sde.value = markRaw(data);
            })
            .catch((err) => {
                error.value = err instanceof Error ? err : new Error(String(err));
            });
    }

    const isReady = computed(() => sde.value !== null);

    return { sde, error, isReady };
}
