import type { ImageCharacterData } from '#shared/api'

export const EVE_IMAGE_SIZES = [
    8,
    16,
    32,
    64,
    128,
    256,
    512,
    1024,
] as const satisfies readonly EveImageSize[]

type ImageQuery = NonNullable<ImageCharacterData['query']>

export type EveImageSize = NonNullable<ImageQuery['size']>
export type EveImageFormat = NonNullable<ImageQuery['format']>

const MAP_IMAGE_SIZES: readonly EveImageSize[] = [32, 64, 128, 512, 1024]
const OLD_CHARACTER_IMAGE_SIZES: readonly EveImageSize[] = [
    8,
    16,
    32,
    64,
    128,
    256,
]
const IMAGE_ORIGIN = 'https://eve-kill.invalid'

function parsedImageURL(src: string): URL | null {
    try {
        return new URL(src, IMAGE_ORIGIN)
    } catch {
        return null
    }
}

function imageCategory(src: string): string | null {
    const url = parsedImageURL(src)
    if (!url || url.origin !== IMAGE_ORIGIN) return null
    const match = /^\/images\/([^/]+)\//.exec(url.pathname)
    return match?.[1] ?? null
}

function allowedSizes(src: string): readonly EveImageSize[] | null {
    switch (imageCategory(src)) {
        case 'characters':
        case 'corporations':
        case 'alliances':
        case 'types':
            return EVE_IMAGE_SIZES
        case 'oldcharacters':
            return OLD_CHARACTER_IMAGE_SIZES
        case 'regions':
        case 'systems':
        case 'constellations':
        case 'ui':
            return MAP_IMAGE_SIZES
        default:
            return null
    }
}

function serializedImageURL(source: string, url: URL): string {
    if (/^https?:\/\//i.test(source)) return url.toString()
    return `${url.pathname}${url.search}${url.hash}`
}

export function eveImageSizeFromURL(src: string): EveImageSize | undefined {
    const url = parsedImageURL(src)
    const size = Number(url?.searchParams.get('size'))
    const sizes = allowedSizes(src)
    return sizes?.find(candidate => candidate === size)
}

export function eveImageURL(
    src: string,
    options: {
        size?: EveImageSize
        format?: EveImageFormat
    } = {},
): string {
    const url = parsedImageURL(src)
    if (!url) return src

    if (options.size && allowedSizes(src)?.includes(options.size)) {
        url.searchParams.set('size', String(options.size))
    }
    if (
        options.format &&
        options.format !== 'auto' &&
        allowedSizes(src)
    ) {
        url.searchParams.set('format', options.format)
        url.searchParams.delete('imagetype')
    }
    return serializedImageURL(src, url)
}

export function eveImageSrcset(
    src: string,
    size: EveImageSize | undefined,
    format: EveImageFormat = 'auto',
): string | undefined {
    if (!size) return undefined
    const sizes = allowedSizes(src)
    if (!sizes?.includes(size)) return undefined

    const doubled = size * 2
    const retina = sizes.find(candidate => candidate >= doubled)
    const base = eveImageURL(src, { size, format })
    if (!retina) return base
    return `${base} 1x, ${eveImageURL(src, { size: retina, format })} 2x`
}
