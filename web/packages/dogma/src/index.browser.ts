/**
 * @evekill/dogma — browser entry (Vite only).
 *
 * Mirror of ./index.ts but uses fetch() against Vite-resolved asset URLs
 * instead of fs/promises. The ?url imports are recognized by Vite at build
 * time: each is rewritten to the URL of the emitted asset, and the
 * referenced file is copied into the output bundle with a content-hashed
 * name. At runtime the browser just fetches the asset URLs from the same
 * origin.
 *
 * This file is only reachable via the "browser" condition in
 * packages/dogma/package.json — Bun, Node, and Nuxt SSR resolve to
 * src/index.ts instead (which uses fs-based IO).
 *
 * On first calculateFit():
 *   ~600ms  — fetch + compile WASM (154 KB) + fetch SDE blobs (10 MB)
 *             all in parallel, decode protobufs, instantiate engine
 *   subsequent calls: ~ms for the calculate() itself, the caches stay warm
 *
 * The SDE blobs dominate cold start; a service worker or HTTP cache (which
 * Vite emits with hashed filenames) means that cost is paid exactly once
 * per content version per browser.
 */

import { createDogma, type RawSdeBytes } from "./core";

// Vite asset URL imports. The ?url suffix tells Vite to emit each file as
// a static asset and resolve the import to its runtime URL. Without it,
// Vite would try to treat these files as source modules (and crash on the
// binary ones). The esf.proto schema is no longer fetched at runtime — its
// message classes were pre-generated into ./generated/esf.js, which removes
// both the network round-trip and the codegen path that needed `unsafe-eval`.
import wasmUrl from "../dist/upstream/dogma-engine/esf_dogma_engine_bg.wasm?url";
import typesUrl from "../dist/upstream/data/dist/sde/types.pb2?url";
import typeDogmaUrl from "../dist/upstream/data/dist/sde/typeDogma.pb2?url";
import dogmaAttributesUrl from "../dist/upstream/data/dist/sde/dogmaAttributes.pb2?url";
import dogmaEffectsUrl from "../dist/upstream/data/dist/sde/dogmaEffects.pb2?url";
import categoriesUrl from "../dist/upstream/data/dist/sde/categories.pb2?url";
import groupsUrl from "../dist/upstream/data/dist/sde/groups.pb2?url";
import marketGroupsUrl from "../dist/upstream/data/dist/sde/marketGroups.pb2?url";

const dogma = createDogma({
    async loadSdeBytes(): Promise<RawSdeBytes> {
        // Seven blobs fetched in parallel — HTTP/2 multiplexing handles it
        // fine, and HTTP/1.1 falls back to the browser's default concurrency
        // window (typically 6 per origin). The last three (categories,
        // groups, marketGroups) back the HardwareListing sidebar; the first
        // four are what the engine actually calculates against.
        const [
            types,
            typeDogma,
            dogmaAttributes,
            dogmaEffects,
            categories,
            groups,
            marketGroups,
        ] = await Promise.all([
            fetch(typesUrl).then((r) => r.arrayBuffer()),
            fetch(typeDogmaUrl).then((r) => r.arrayBuffer()),
            fetch(dogmaAttributesUrl).then((r) => r.arrayBuffer()),
            fetch(dogmaEffectsUrl).then((r) => r.arrayBuffer()),
            fetch(categoriesUrl).then((r) => r.arrayBuffer()),
            fetch(groupsUrl).then((r) => r.arrayBuffer()),
            fetch(marketGroupsUrl).then((r) => r.arrayBuffer()),
        ]);
        return {
            types,
            typeDogma,
            dogmaAttributes,
            dogmaEffects,
            categories,
            groups,
            marketGroups,
        };
    },
    async loadEngineBytes(): Promise<ArrayBuffer> {
        const r = await fetch(wasmUrl);
        return r.arrayBuffer();
    },
});

export const {
    calculateFit,
    loadSde,
    loadEngine,
    getEngine,
    getItemStat,
    getHullStat,
    getHullStats,
} = dogma;

export * from "./types";
export { fittingToEsfFit, type FittingItemRow, type FittingToEsfOptions } from "./converter";
export type { SdeData, LoadedEngine, RawSdeBytes, DogmaIO, DogmaApi } from "./core";
