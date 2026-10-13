// Preserve the complete Lucide collection without enlarging the common
// sprite downloaded by every page. Uncommon configurable icons are tiny,
// independently cached SVG assets generated from the same Iconify source.
import { getIconData, iconToSVG } from '@iconify/utils'
import { resolve } from 'node:path'
import {readFile} from 'node:fs/promises'
import {writeChanged} from './io'

const root = resolve(import.meta.dirname, '../..')
const collection = JSON.parse(await readFile(resolve(root, 'frontend/node_modules/@iconify-json/lucide/icons.json'),'utf8'))
const catalog: Record<string, { viewBox: string; body: string }> = {}
for (const name of [...Object.keys(collection.icons), ...Object.keys(collection.aliases ?? {})].sort()) {
    const data = getIconData(collection, name)
    if (!data) continue
    const svg = iconToSVG(data)
    catalog[name] = { viewBox: svg.attributes.viewBox!, body: svg.body }
}
const sprite = await readFile(resolve(root, 'internal/frontend/static/icons.svg'),'utf8')
const known = [...sprite.matchAll(/<symbol id="([^"]+)"/g)].map(m => m[1]).sort()
await writeChanged(resolve(root,'internal/frontend/static/lucide-icons.json'),JSON.stringify(catalog)+'\n')
await writeChanged(resolve(root,'frontend/data/icon-names.json'),JSON.stringify(known,null,2)+'\n')
console.log(`Go Lucide collection: ${Object.keys(catalog).length} individually cached icons`)
