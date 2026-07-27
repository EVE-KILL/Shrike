/**
 * @evekill/dogma — node/Bun entry.
 *
 * Thin adapter that wires fs-based IO into the platform-agnostic factory in
 * ./core. Consumer-facing API is identical to the browser entry at
 * ./index.browser — any code that imports from "@evekill/dogma" gets the same
 * functions regardless of runtime.
 *
 * Package.json `exports` uses the "browser" condition to route Vite client
 * builds at src/index.browser.ts, and the "default" condition back here for
 * Bun / Node / Nuxt SSR.
 */

import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createDogma, type RawSdeBytes } from "./core";

const HERE = dirname(fileURLToPath(import.meta.url));
const UPSTREAM_DIR = resolve(HERE, "../dist/upstream");
const DATA_DIST = resolve(UPSTREAM_DIR, "data/dist");
const ENGINE_DIR = resolve(UPSTREAM_DIR, "dogma-engine");

const dogma = createDogma({
    async loadSdeBytes(): Promise<RawSdeBytes> {
        const [types, typeDogma, dogmaAttributes, dogmaEffects, categories, groups, marketGroups] =
            await Promise.all([
                readFile(resolve(DATA_DIST, "sde/types.pb2")),
                readFile(resolve(DATA_DIST, "sde/typeDogma.pb2")),
                readFile(resolve(DATA_DIST, "sde/dogmaAttributes.pb2")),
                readFile(resolve(DATA_DIST, "sde/dogmaEffects.pb2")),
                readFile(resolve(DATA_DIST, "sde/categories.pb2")),
                readFile(resolve(DATA_DIST, "sde/groups.pb2")),
                readFile(resolve(DATA_DIST, "sde/marketGroups.pb2")),
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
    async loadEngineBytes() {
        return readFile(resolve(ENGINE_DIR, "esf_dogma_engine_bg.wasm"));
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
