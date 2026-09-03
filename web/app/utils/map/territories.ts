import { contours } from 'd3-contour'

export interface TerritoryNode {
    id: number
    x: number
    y: number
}

export interface TerritoryLink {
    from: number
    to: number
}

export interface TerritoryBounds {
    x: number
    y: number
    width: number
    height: number
}

export interface TerritoryPolygon {
    rings: number[][]
    area: number
}

export interface AllianceTerritory {
    allianceId: number
    polygons: TerritoryPolygon[]
}

export interface SovereigntyTerritoryMap {
    width: number
    height: number
    bounds: TerritoryBounds
    owners: Int32Array
    territories: AllianceTerritory[]
}

function segmentDistanceSquared(px: number, py: number, ax: number, ay: number, bx: number, by: number) {
    const dx = bx - ax, dy = by - ay
    const lengthSquared = dx * dx + dy * dy
    const t = lengthSquared === 0 ? 0 : Math.max(0, Math.min(1, ((px - ax) * dx + (py - ay) * dy) / lengthSquared))
    const x = ax + t * dx, y = ay + t * dy
    return (px - x) ** 2 + (py - y) ** 2
}

function ringArea(points: number[][]) {
    let area = 0
    for (let index = 0; index < points.length - 1; index++) {
        const point = points[index]!, next = points[index + 1]!
        area += point[0]! * next[1]! - next[0]! * point[1]!
    }
    return Math.abs(area / 2)
}

function smoothRing(points: number[][], passes = 2) {
    if (points.length < 5) return points
    let current = points.slice(0, -1)
    for (let pass = 0; pass < passes; pass++) {
        const next: number[][] = []
        for (let index = 0; index < current.length; index++) {
            const point = current[index]!, following = current[(index + 1) % current.length]!
            next.push([
                point[0]! * 0.75 + following[0]! * 0.25,
                point[1]! * 0.75 + following[1]! * 0.25,
            ], [
                point[0]! * 0.25 + following[0]! * 0.75,
                point[1]! * 0.25 + following[1]! * 0.75,
            ])
        }
        current = next
    }
    return [...current, current[0]!]
}

export function createSovereigntyTerritories(
    nodes: TerritoryNode[],
    links: TerritoryLink[],
    bounds: TerritoryBounds,
    ownerBySystem: Map<number, number>,
    gridWidth = 560,
): SovereigntyTerritoryMap {
    const width = Math.max(64, Math.round(gridWidth))
    const height = Math.max(64, Math.round(width * bounds.height / Math.max(bounds.width, 1)))
    const owners = new Int32Array(width * height)
    const distances = new Float32Array(width * height)
    distances.fill(Number.POSITIVE_INFINITY)
    const nodeById = new Map(nodes.map(node => [node.id, node]))
    const cellWidth = bounds.width / width, cellHeight = bounds.height / height
    const radius = 22

    const paint = (allianceId: number, minX: number, maxX: number, minY: number, maxY: number, distanceAt: (x: number, y: number) => number) => {
        const fromX = Math.max(0, Math.floor((minX - bounds.x) / cellWidth))
        const toX = Math.min(width - 1, Math.ceil((maxX - bounds.x) / cellWidth))
        const fromY = Math.max(0, Math.floor((minY - bounds.y) / cellHeight))
        const toY = Math.min(height - 1, Math.ceil((maxY - bounds.y) / cellHeight))
        for (let y = fromY; y <= toY; y++) {
            const worldY = bounds.y + (y + 0.5) * cellHeight
            for (let x = fromX; x <= toX; x++) {
                const worldX = bounds.x + (x + 0.5) * cellWidth
                const distance = distanceAt(worldX, worldY)
                if (distance > radius * radius) continue
                const offset = y * width + x
                if (distance < distances[offset]! || (distance === distances[offset] && allianceId < owners[offset]!)) {
                    distances[offset] = distance
                    owners[offset] = allianceId
                }
            }
        }
    }

    for (const link of links) {
        const from = nodeById.get(link.from), to = nodeById.get(link.to)
        const allianceId = ownerBySystem.get(link.from)
        if (!from || !to || !allianceId || ownerBySystem.get(link.to) !== allianceId) continue
        if (Math.hypot(from.x - to.x, from.y - to.y) > 45) continue
        paint(
            allianceId,
            Math.min(from.x, to.x) - radius, Math.max(from.x, to.x) + radius,
            Math.min(from.y, to.y) - radius, Math.max(from.y, to.y) + radius,
            (x, y) => segmentDistanceSquared(x, y, from.x, from.y, to.x, to.y),
        )
    }
    for (const node of nodes) {
        const allianceId = ownerBySystem.get(node.id)
        if (!allianceId) continue
        paint(allianceId, node.x - radius, node.x + radius, node.y - radius, node.y + radius, (x, y) => (x - node.x) ** 2 + (y - node.y) ** 2)
    }

    // Make the actual system locations authoritative after overlapping areas
    // have been resolved, and carve a small void around NPC/unclaimed systems.
    const carveRadius = 9
    for (const node of nodes) {
        const allianceId = ownerBySystem.get(node.id) ?? 0
        const localRadius = allianceId ? 7 : carveRadius
        const fromX = Math.max(0, Math.floor((node.x - localRadius - bounds.x) / cellWidth))
        const toX = Math.min(width - 1, Math.ceil((node.x + localRadius - bounds.x) / cellWidth))
        const fromY = Math.max(0, Math.floor((node.y - localRadius - bounds.y) / cellHeight))
        const toY = Math.min(height - 1, Math.ceil((node.y + localRadius - bounds.y) / cellHeight))
        for (let y = fromY; y <= toY; y++) {
            const worldY = bounds.y + (y + 0.5) * cellHeight
            for (let x = fromX; x <= toX; x++) {
                const worldX = bounds.x + (x + 0.5) * cellWidth
                if ((worldX - node.x) ** 2 + (worldY - node.y) ** 2 > localRadius ** 2) continue
                const offset = y * width + x
                owners[offset] = allianceId
                distances[offset] = 0
            }
        }
    }

    const extents = new Map<number, { minX: number; maxX: number; minY: number; maxY: number }>()
    for (let y = 0; y < height; y++) {
        for (let x = 0; x < width; x++) {
            const allianceId = owners[y * width + x]!
            if (!allianceId) continue
            const extent = extents.get(allianceId) ?? { minX: x, maxX: x, minY: y, maxY: y }
            extent.minX = Math.min(extent.minX, x); extent.maxX = Math.max(extent.maxX, x)
            extent.minY = Math.min(extent.minY, y); extent.maxY = Math.max(extent.maxY, y)
            extents.set(allianceId, extent)
        }
    }

    const territories: AllianceTerritory[] = []
    for (const [allianceId, extent] of extents) {
        const originX = Math.max(0, extent.minX - 2), originY = Math.max(0, extent.minY - 2)
        const localWidth = Math.min(width - 1, extent.maxX + 2) - originX + 1
        const localHeight = Math.min(height - 1, extent.maxY + 2) - originY + 1
        const mask = new Array<number>(localWidth * localHeight).fill(0)
        for (let y = 0; y < localHeight; y++) {
            for (let x = 0; x < localWidth; x++) {
                mask[y * localWidth + x] = owners[(originY + y) * width + originX + x] === allianceId ? 1 : 0
            }
        }
        const contour = contours().size([localWidth, localHeight]).thresholds([0.5]).smooth(true)(mask)[0]
        if (!contour) continue
        const polygons = contour.coordinates.map(polygon => {
            const rings = polygon.map(ring => smoothRing(ring.map(point => [
                bounds.x + (originX + point[0]!) * cellWidth,
                bounds.y + (originY + point[1]!) * cellHeight,
            ])))
            return { rings: rings.map(ring => ring.flat()), area: ringArea(rings[0] ?? []) }
        }).filter(polygon => (polygon.rings[0]?.length ?? 0) >= 8)
        if (polygons.length) territories.push({ allianceId, polygons })
    }
    return { width, height, bounds, owners, territories }
}

export function sovereigntyOwnerAt(map: SovereigntyTerritoryMap, x: number, y: number) {
    const gridX = Math.floor((x - map.bounds.x) / map.bounds.width * map.width)
    const gridY = Math.floor((y - map.bounds.y) / map.bounds.height * map.height)
    if (gridX < 0 || gridX >= map.width || gridY < 0 || gridY >= map.height) return null
    return map.owners[gridY * map.width + gridX] || null
}
