/**
 * Derive a usable UI accent from a corporation's ESI palette.
 *
 * Palettes come from the in-game logo designer, so they're frequently
 * near-black (logo colors on a dark backdrop). We pick the most vivid of the
 * up-to-three colors by OKLCH chroma, then clamp lightness into a band that
 * reads on the site's dark surfaces. Palettes that are effectively grayscale
 * produce no accent at all — callers fall back to the theme's default accent.
 */

export interface EntityPalette {
    main_color?: string
    secondary_color?: string
    tertiary_color?: string
}

export interface EntityAccent {
    /** Opaque accent color (hex), lightness-clamped for dark surfaces */
    accent: string
    /** Darker variant for hover states (hex) */
    hover: string
    /** Low-alpha version for background washes */
    soft: string
    /** Mid-alpha version for borders/underlines */
    border: string
    /** OKLCH hue in degrees — for distance checks between two accents */
    hue: number
}

/** Minimum OKLCH chroma before a palette color counts as "vivid" */
const MIN_CHROMA = 0.04
/** Lightness band that stays visible on dark backgrounds without glowing */
const L_MIN = 0.62
const L_MAX = 0.78

const parseHex = (hex: string): [number, number, number] | null => {
    const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim())
    if (!m) return null
    const n = Number.parseInt(m[1]!, 16)
    return [((n >> 16) & 0xff) / 255, ((n >> 8) & 0xff) / 255, (n & 0xff) / 255]
}

const linearize = (c: number) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)
const delinearize = (c: number) => (c <= 0.0031308 ? c * 12.92 : 1.055 * c ** (1 / 2.4) - 0.055)

const srgbToOklch = ([r, g, b]: [number, number, number]): [number, number, number] => {
    const [lr, lg, lb] = [linearize(r), linearize(g), linearize(b)]
    const l = Math.cbrt(0.4122214708 * lr + 0.5363325363 * lg + 0.0514459929 * lb)
    const m = Math.cbrt(0.2119034982 * lr + 0.6806995451 * lg + 0.1073969566 * lb)
    const s = Math.cbrt(0.0883024619 * lr + 0.2817188376 * lg + 0.6299787005 * lb)
    const L = 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s
    const a = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s
    const bb = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s
    return [L, Math.hypot(a, bb), Math.atan2(bb, a)]
}

const oklchToSrgb = ([L, C, h]: [number, number, number]): [number, number, number] => {
    const a = C * Math.cos(h)
    const b = C * Math.sin(h)
    const l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3
    const m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3
    const s = (L - 0.0894841775 * a - 1.291485548 * b) ** 3
    return [
        delinearize(4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s),
        delinearize(-1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s),
        delinearize(-0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s),
    ]
}

const inGamut = (rgb: [number, number, number]) => rgb.every(c => c >= -0.0001 && c <= 1.0001)

/** Reduce chroma until the color fits in sRGB */
const gamutMap = (L: number, C: number, h: number): [number, number, number] => {
    let rgb = oklchToSrgb([L, C, h])
    if (inGamut(rgb)) return rgb
    let lo = 0
    let hi = C
    for (let i = 0; i < 20; i++) {
        const mid = (lo + hi) / 2
        rgb = oklchToSrgb([L, mid, h])
        if (inGamut(rgb)) lo = mid
        else hi = mid
    }
    return oklchToSrgb([L, lo, h])
}

const toHex = (rgb: [number, number, number]) =>
    `#${rgb.map(c => Math.round(Math.min(1, Math.max(0, c)) * 255).toString(16).padStart(2, "0")).join("")}`

export const entityAccent = (palette: EntityPalette | null | undefined): EntityAccent | null => {
    if (!palette) return null

    let best: [number, number, number] | null = null
    for (const hex of [palette.main_color, palette.secondary_color, palette.tertiary_color]) {
        if (!hex) continue
        const rgb = parseHex(hex)
        if (!rgb) continue
        const lch = srgbToOklch(rgb)
        if (lch[1] >= MIN_CHROMA && (!best || lch[1] > best[1])) best = lch
    }
    if (!best) return null

    const [, C, h] = best
    const L = Math.min(Math.max(best[0], L_MIN), L_MAX)
    const rgb = gamutMap(L, C, h)
    const [r, g, b] = rgb.map(c => Math.round(Math.min(1, Math.max(0, c)) * 255))
    return {
        accent: toHex(rgb),
        hover: toHex(gamutMap(Math.max(L - 0.08, 0.4), C, h)),
        soft: `rgba(${r}, ${g}, ${b}, 0.08)`,
        border: `rgba(${r}, ${g}, ${b}, 0.4)`,
        hue: Math.round(((h * 180 / Math.PI) + 360) % 360),
    }
}

/**
 * Accents for two opposing battle teams. All-or-nothing: both teams must
 * resolve to an accent AND their hues must be far enough apart to read as
 * different sides — otherwise fall back to the default red/blue scheme.
 */
export const battleTeamAccents = (palettes: (EntityPalette | null | undefined)[]): (EntityAccent | null)[] => {
    const accents = palettes.map(p => entityAccent(p))
    if (accents.length !== 2 || !accents[0] || !accents[1]) return palettes.map(() => null)
    const dh = Math.abs(accents[0].hue - accents[1].hue)
    if (Math.min(dh, 360 - dh) < 60) return palettes.map(() => null)
    return accents
}

/**
 * Accents for N campaign sides (1–4). Same all-or-nothing contract as
 * battleTeamAccents: every side must resolve to an accent AND all pairwise
 * hues must be ≥60° apart — otherwise everyone falls back to the fixed
 * side palette (red/blue/purple/amber). A single-side campaign only needs
 * its own accent to resolve.
 */
export const campaignSideAccents = (palettes: (EntityPalette | null | undefined)[]): (EntityAccent | null)[] => {
    const accents = palettes.map(p => entityAccent(p))
    if (accents.length === 0 || accents.some(a => !a)) return palettes.map(() => null)
    for (let i = 0; i < accents.length; i++) {
        for (let j = i + 1; j < accents.length; j++) {
            const dh = Math.abs(accents[i]!.hue - accents[j]!.hue)
            if (Math.min(dh, 360 - dh) < 60) return palettes.map(() => null)
        }
    }
    return accents
}
