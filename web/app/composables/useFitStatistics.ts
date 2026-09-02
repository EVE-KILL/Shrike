/**
 * Watches the current fit and keeps a `stats` ref in sync by running
 * @evekill/dogma's `calculateFit` whenever the fit changes.
 *
 * Two refs are exposed:
 *
 *   - `stats`  — the "preview-applied" result, bound to `fit` (which is
 *                itself preview-applied in useCurrentFit). This is what
 *                the ring, stats panel, and widgets render.
 *   - `currentStats` — the "last non-preview" result, only updated when
 *                the committed fit changes. This is what ShipAttribute
 *                compares `stats` against to paint green/red delta
 *                indicators during hover.
 *
 * ## Singleton watcher
 *
 * This composable is called by ~90 components on the fitting page
 * (31 slots + ~60 stat widgets). A naive implementation would create
 * a fresh `watch(fit, …)` per caller, which means 90 concurrent
 * calculateFit runs per mutation — catastrophic for perf.
 *
 * Instead we install the watcher exactly once, in a detached
 * `effectScope` that outlives every component. Subsequent callers
 * just read the shared `useState`-backed refs. The detached scope is
 * never disposed — in practice the fitting page is a persistent app
 * and the watcher should live for the session. (In dev HMR may leak
 * a scope per reload; acceptable.)
 *
 * Concurrency: rapid hover events kick off new calculateFit calls
 * before the previous ones resolve. A monotonic generation counter
 * inside the singleton scope discards any stale result so the latest
 * preview always wins. No debouncing — the WASM calculation is fast
 * enough (<50 ms per fit after warmup) that debouncing adds more
 * latency than it saves.
 *
 * Skills come from useFitSkillProfile. Its current presets generate a full
 * all-III/all-IV/all-V map; ESI and custom maps can be added there later
 * without changing any statistics consumers.
 */

import { effectScope } from "vue";
import { calculateFit, type FitStats } from "@evekill/dogma";
import { fitToEngine } from "./fit/toEngine";

// Module-scoped flag: true once the singleton watcher is live. Guarded
// by `import.meta.client` so SSR never touches it. Don't use useState
// for this — a useState key is per-request on SSR and that's the
// opposite of what we want (we want the flag to be single-process and
// reset cleanly between page navigations via the detached scope).
let watcherInstalled = false;

export function useFitStatistics() {
    const stats = useState<FitStats | null>("fit:stats", () => null);
    const currentStats = useState<FitStats | null>("fit:stats:current", () => null);
    const isLoading = useState<boolean>("fit:stats:loading", () => false);
    const error = useState<Error | null>("fit:stats:error", () => null);

    if (import.meta.client && !watcherInstalled) {
        watcherInstalled = true;

        // Detached scope: outlives every component that mounts on the
        // page. The watch inside only runs once regardless of how many
        // components call useFitStatistics().
        const scope = effectScope(true);
        scope.run(() => {
            const { fit, currentFit, isPreview } = useCurrentFit();
            const { skills } = useFitSkillProfile();
            let generation = 0;

            watch(
                [fit, skills],
                async ([next, skillMap]) => {
                    if (next === null) {
                        stats.value = null;
                        return;
                    }

                    const gen = ++generation;
                    isLoading.value = true;
                    error.value = null;

                    try {
                        const result = await calculateFit(fitToEngine(next), skillMap);
                        if (gen !== generation) return; // a newer request superseded us
                        // CRITICAL: markRaw to keep Vue from reactively
                        // wrapping the result. FitStats contains a
                        // `hull.attributes` Map plus one Map per fitted
                        // item; without markRaw every lookup goes through
                        // a Proxy and the UI pegs at 100% CPU.
                        const raw = markRaw(result);
                        stats.value = raw;
                        // Mirror to currentStats whenever the committed
                        // fit is what we just calculated (not a preview).
                        if (!isPreview.value) {
                            currentStats.value = raw;
                        }
                    } catch (err) {
                        if (gen !== generation) return;
                        error.value =
                            err instanceof Error ? err : new Error(String(err));
                        // eslint-disable-next-line no-console
                        console.error("calculateFit failed:", err);
                    } finally {
                        if (gen === generation) {
                            isLoading.value = false;
                        }
                    }
                },
                // No `deep: true` — mutators always replace the fit with
                // a fresh clone, so identity changes trigger the watch,
                // and deep-tracking the markRaw'd fit would be a no-op
                // anyway.
                { immediate: true },
            );

            // When the committed fit changes without a preview, also
            // clear any stale currentStats that belonged to the previous
            // hull.
            watch(currentFit, (next) => {
                if (next === null) currentStats.value = null;
            });
        });
    }

    return { stats, currentStats, isLoading, error };
}
