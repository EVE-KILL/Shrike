import { describe, expect, test } from 'bun:test'
import {
    eveImageSizeFromURL,
    eveImageSrcset,
    eveImageURL,
    eveImageWidthSrcset,
} from '../shared/utils/eveImage'

describe('EVE image URLs', () => {
    test('fluid candidates retain format and extra parameters without exceeding backend sizes', () => {
        const candidates = eveImageWidthSrcset('/images/oldcharacters/7?size=64&foo=bar', 'webp')!
        expect(candidates).toContain('/images/oldcharacters/7?size=128&foo=bar&format=webp 128w')
        expect(candidates).toContain('/images/oldcharacters/7?size=256&foo=bar&format=webp 256w')
        expect(candidates).not.toContain('512w')
        expect(candidates).not.toContain(' 2x')
        expect(eveImageWidthSrcset('https://example.com/photo.png')).toBeUndefined()
    })

    test('sets a supported size without losing existing query parameters', () => {
        expect(eveImageURL(
            '/images/characters/7/portrait?foo=bar',
            { size: 64 },
        )).toBe('/images/characters/7/portrait?foo=bar&size=64')
    })

    test('builds a retina source set backed by Go variants', () => {
        expect(eveImageSrcset(
            '/images/types/670/icon?size=64',
            64,
        )).toBe(
            '/images/types/670/icon?size=64 1x, ' +
            '/images/types/670/icon?size=128 2x',
        )
    })

    test('uses the next generated map size for responsive images', () => {
        expect(eveImageSrcset('/images/systems/30000142', 128)).toBe(
            '/images/systems/30000142?size=128 1x, ' +
            '/images/systems/30000142?size=256 2x',
        )
    })

    test('supports full-size generated maps', () => {
        expect(eveImageURL('/images/regions/10000042', { size: 1024 })).toBe(
            '/images/regions/10000042?size=1024',
        )
    })

    test('sizes and converts legacy portraits through Go', () => {
        expect(eveImageSrcset(
            '/images/oldcharacters/7',
            64,
            'webp',
        )).toBe(
            '/images/oldcharacters/7?size=64&format=webp 1x, ' +
            '/images/oldcharacters/7?size=128&format=webp 2x',
        )
    })

    test('leaves non-image-service URLs untouched', () => {
        expect(eveImageURL('https://example.com/image.png', {
            size: 64,
            format: 'webp',
        })).toBe('https://example.com/image.png')
    })

    test('reads an existing supported size', () => {
        expect(eveImageSizeFromURL(
            '/images/alliances/99000001/logo?size=256',
        )).toBe(256)
    })
})
