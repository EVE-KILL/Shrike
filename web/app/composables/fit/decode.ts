/**
 * ESF fit decoder — the flip side of encode.ts. Takes a
 * `v3:<base64-gzip>` string and returns a `Fit`.
 *
 * Ported from @eveshipfit/react's ImportEveShipFit + DecodeEsfFitV3.
 * Only v3 is supported — v1/v2 predate EVEShipFit's current format
 * and realistically nothing still points at them. EFT + killmail
 * imports live in their own modules.
 *
 * Dispatcher lets the same entry point handle `?fit=v3:xxx` from
 * URL query params (what eveship.fit does) as well as arbitrary
 * pasted strings.
 */

import type { Fit, FitSlotType, FitState } from "./types";
import { cleanFit } from "./clean";

/**
 * base64 → ungzip → utf-8 string. Mirror of encode.ts's compress
 * pipeline. Spaces in the base64 (some URL encoders replace `+`
 * with space) are normalized back before decoding.
 */
async function base64Gunzip(b64: string): Promise<string> {
    const binary = atob(b64.replace(/ /g, "+"));
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
    const compressed = new Blob([bytes]).stream();
    const decompressed = compressed.pipeThrough(new DecompressionStream("gzip"));
    return new Response(decompressed).text();
}

/** Parse the CSV body inside a v3 blob into a Fit. */
export async function decodeFitV3(compressed: string): Promise<Fit | null> {
    let text: string;
    try {
        text = await base64Gunzip(compressed);
    } catch {
        return null;
    }

    const lines = text.trim().split("\n");
    if (lines.length === 0) return null;

    const header = lines[0]!.split(",");
    if (header[0] !== "ship") return null;

    const fit: Fit = {
        shipTypeId: Number.parseInt(header[1] ?? "0", 10),
        name: header[2] ?? "Imported Fit",
        description: header[3] ?? "",
        modules: [],
        drones: [],
        cargo: [],
    };

    for (let i = 1; i < lines.length; i++) {
        const parts = lines[i]!.split(",");
        switch (parts[0]) {
            case "module": {
                const rawState = parts[4] as FitState | "Preview";
                fit.modules.push({
                    slot: {
                        type: parts[1] as FitSlotType,
                        index: Number.parseInt(parts[2] ?? "0", 10),
                    },
                    typeId: Number.parseInt(parts[3] ?? "0", 10),
                    state: rawState === "Preview" ? "Active" : rawState,
                    charge: parts[5] ? { typeId: Number.parseInt(parts[5], 10) } : undefined,
                });
                break;
            }
            case "drone":
                fit.drones.push({
                    typeId: Number.parseInt(parts[1] ?? "0", 10),
                    states: {
                        Active: Number.parseInt(parts[2] ?? "0", 10),
                        Passive: Number.parseInt(parts[3] ?? "0", 10),
                    },
                });
                break;
            case "cargo":
                fit.cargo.push({
                    typeId: Number.parseInt(parts[1] ?? "0", 10),
                    quantity: Number.parseInt(parts[2] ?? "0", 10),
                });
                break;
        }
    }

    return cleanFit(fit);
}

/**
 * Entry point that dispatches on the prefix. Accepts `v3:xxx`
 * directly; callers that pull from URL query params should strip
 * any full-link prefix (`https://.../?fit=`) before calling.
 */
export async function decodeEsfEncoded(encoded: string): Promise<Fit | null> {
    // Strip a full URL if present — be forgiving for pasted inputs.
    const fitParam = encoded.includes("?fit=") ? encoded.split("?fit=")[1]! : encoded;
    const colonIdx = fitParam.indexOf(":");
    if (colonIdx < 0) return null;
    const prefix = fitParam.slice(0, colonIdx);
    const blob = fitParam.slice(colonIdx + 1);

    switch (prefix) {
        case "v3":
            return decodeFitV3(blob);
        default:
            return null;
    }
}
