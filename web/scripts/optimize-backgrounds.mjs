import { mkdir, readFile, writeFile } from 'node:fs/promises'
import sharp from 'sharp'

// Keep originals for export/viewing. Bump this directory when changing source
// art or encoding settings: background assets are cached as immutable.
const root = new URL('../public/backgrounds/', import.meta.url)
const output = new URL('optimized-v1/', root)
await mkdir(output, { recursive: true })
for (let id = 1; id <= 8; id++) {
    const name = `bg${id}.webp`
    const original = await readFile(new URL(name, root))
    const encoded = await sharp(original).webp({ quality: 80, effort: 6 }).toBuffer()
    const result = encoded.length < original.length ? encoded : original
    await writeFile(new URL(name, output), result)
    const avif = await sharp(original).avif({ quality: 55, effort: 6 }).toBuffer()
    await writeFile(new URL(`bg${id}.avif`, output), avif)
    console.log(`${name}: ${original.length} -> WebP ${result.length}, AVIF ${avif.length} bytes`)
}
