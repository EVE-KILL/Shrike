/**
 * ESF v3 fit encoder — converts a UI `Fit` into the `v3:...` share
 * string format used by eveship.fit. Ported from
 * @eveshipfit/react's ExportEveShipFit hook with the same line
 * format so our links are interoperable with eveship.fit's.
 *
 * Format inside the gzipped blob:
 *
 *   ship,{shipTypeId},{name},{description}\n
 *   module,{slotType},{slotIndex},{typeId},{state},{chargeTypeId?}\n
 *   drone,{typeId},{activeCount},{passiveCount}\n
 *   cargo,{typeId},{quantity}\n
 *
 * Gzipped, then base64-encoded, then prefixed with `v3:`. The whole
 * thing fits in a URL query param for links up to a few hundred
 * characters of fit data.
 *
 * Pure function — no Vue composable, no reactive state. Callers
 * pass a snapshot `Fit` and get back a Promise<string>.
 */

import type { Fit } from "./types";

/**
 * Gzip → base64. Uses CompressionStream which is available in all
 * modern browsers and Bun. The React reference streams the output
 * manually via a ReadableStream reader; using `new Response()` as a
 * sink lets us collect the bytes with a single `arrayBuffer()` call.
 */
async function gzipBase64(str: string): Promise<string> {
    const input = new Blob([str]).stream();
    const compressed = input.pipeThrough(new CompressionStream("gzip"));
    const buffer = await new Response(compressed).arrayBuffer();
    const bytes = new Uint8Array(buffer);
    let binary = "";
    // Chunked conversion to avoid "Maximum call stack size" when
    // String.fromCharCode is called with thousands of args at once.
    const CHUNK = 0x8000;
    for (let i = 0; i < bytes.length; i += CHUNK) {
        binary += String.fromCharCode.apply(null, Array.from(bytes.subarray(i, i + CHUNK)));
    }
    return btoa(binary);
}

/**
 * Encode a fit into the `v3:<base64-gzip-csv>` string used by the
 * share link and clipboard copy flow.
 */
export async function encodeFitV3(fit: Fit): Promise<string> {
    let result = `ship,${fit.shipTypeId},${fit.name},${fit.description}\n`;

    for (const m of fit.modules) {
        // Preview state is a UI overlay — never persist it. Collapse
        // back to Active so round-tripped fits match the committed
        // snapshot.
        const state = m.state === "Preview" ? "Active" : m.state;
        result += `module,${m.slot.type},${m.slot.index},${m.typeId},${state},${m.charge?.typeId ?? ""}\n`;
    }
    for (const d of fit.drones) {
        result += `drone,${d.typeId},${d.states.Active},${d.states.Passive}\n`;
    }
    for (const c of fit.cargo) {
        result += `cargo,${c.typeId},${c.quantity}\n`;
    }

    return "v3:" + (await gzipBase64(result));
}

/**
 * Build a shareable URL containing the current fit. `base` defaults
 * to the current origin — pass a full URL if you need one for a
 * different environment. Recipients land on /fits/create which auto-
 * loads the encoded fit from the `?fit=` query param on mount.
 */
export async function buildShareLink(fit: Fit, base?: string): Promise<string> {
    const encoded = await encodeFitV3(fit);
    const origin = base ?? (typeof window !== "undefined" ? window.location.origin : "");
    return `${origin}/fits/create?fit=${encoded}`;
}
