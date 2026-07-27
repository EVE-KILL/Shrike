/**
 * Hand-written type shim for ./esf.js — the pbjs static-module output.
 *
 * pbts (the protobufjs type generator) emits a 1300-line .d.ts that imports
 * `long` and the full `protobufjs` type tree. We don't need any of that —
 * core.ts only ever calls `decode(buffer)` followed by `toObject(message,
 * options)`, then casts the result through `Record<string, any>` and walks
 * it manually. So this shim describes just enough surface area to make those
 * two call sites type-check cleanly, and nothing more.
 *
 * If you bump the upstream proto schema and re-run `bun run gen:proto`, the
 * regenerated esf.js will still satisfy this shim as long as the four message
 * types (Types, TypeDogma, DogmaAttributes, DogmaEffects) stay map-shaped
 * with an `entries` field — which is structural to the SDE dump format and
 * unlikely to change.
 */

interface ConvertOptions {
    enums?: NumberConstructor | StringConstructor;
    longs?: NumberConstructor | StringConstructor;
    arrays?: boolean;
    objects?: boolean;
    defaults?: boolean;
}

interface MessageType {
    decode(buffer: Uint8Array | ArrayBuffer, length?: number): unknown;
    toObject(message: unknown, options?: ConvertOptions): { entries: Record<string, unknown> };
}

export const esf: {
    Types: MessageType;
    TypeDogma: MessageType;
    DogmaAttributes: MessageType;
    DogmaEffects: MessageType;
    Categories: MessageType;
    Groups: MessageType;
    MarketGroups: MessageType;
};
