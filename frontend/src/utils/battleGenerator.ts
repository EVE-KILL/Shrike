import { eveInputToDate, eveInputToIso, isoToEveInput } from './eveTime'
export { eveInputToDate, eveInputToIso, isoToEveInput }
export { formatNumber, formatIsk } from './formatters'

export interface GeneratorEntity {
    id: number
    name: string
    type: 'alliance' | 'corporation' | 'faction'
    alliance_id?: number | null
    alliance_name?: string | null
}
export interface GeneratorEntities {
    factions: GeneratorEntity[]
    alliances: GeneratorEntity[]
    corporations: GeneratorEntity[]
}
export const generatorKey = (entity: GeneratorEntity) => `${entity.type}-${entity.id}`
export function generatorWindowHours(start: string, end: string): number {
    if (!start || !end) return 0
    return (eveInputToDate(end).getTime() - eveInputToDate(start).getTime()) / 3_600_000
}
export function generatorTimeError(start: string, end: string): string | null {
    if (start && end && eveInputToDate(end) <= eveInputToDate(start)) return 'End must be after start'
    if (generatorWindowHours(start, end) > 36) return 'Maximum 36 hour window'
    return null
}
export function generatorCanLoad(systems: {id: number}[], start: string, end: string): boolean {
    return systems.length > 0 && systems.length <= 5 && !!start && !!end
        && eveInputToDate(end) > eveInputToDate(start) && !generatorTimeError(start, end)
}
// The exact alliance grouping, reload preservation and seeding used by the Vue wizard.
export function generatorEntityList(result: GeneratorEntities): GeneratorEntity[] {
    const alliances = new Set(result.alliances.map(a => a.id))
    return [...result.factions, ...result.alliances, ...result.corporations.filter(c => !c.alliance_id || !alliances.has(c.alliance_id))]
}
export function generatorKeepAssignments(side: GeneratorEntity[], listed: GeneratorEntity[]): GeneratorEntity[] {
    return side.filter(picked => listed.some(e => e.type === picked.type && e.id === picked.id))
}
export function generatorUndecided(all: GeneratorEntity[], sideA: GeneratorEntity[], sideB: GeneratorEntity[], filter: string): GeneratorEntity[] {
    const assigned = new Set([...sideA, ...sideB].map(generatorKey))
    let list = all.filter(e => !assigned.has(generatorKey(e)))
    const q = filter.toLowerCase().trim()
    if (q) list = list.filter(e => e.name.toLowerCase().includes(q) || e.alliance_name?.toLowerCase().includes(q))
    return list
}
export function generatorMove(sideA: GeneratorEntity[], sideB: GeneratorEntity[], entity: GeneratorEntity, target: 'a' | 'b' | 'u') {
    const remaining = (side: GeneratorEntity[]) => side.filter(e => generatorKey(e) !== generatorKey(entity))
    const a = remaining(sideA), b = remaining(sideB)
    if (target === 'a') a.push(entity)
    if (target === 'b') b.push(entity)
    return { sideA: a, sideB: b }
}
export interface GeneratorSeedTeam {alliances?:Array<{alliance_id?:number|null;corporations?:Array<{corporation_id:number}>}>}
export function generatorSeedTeams(all: GeneratorEntity[], teams: GeneratorSeedTeam[]) {
    const find = (type: 'alliance' | 'corporation', id: number) => all.find(e => e.type === type && e.id === id)
    const seed = (team: GeneratorSeedTeam|undefined) => {
        const picked: GeneratorEntity[] = []
        for (const group of team?.alliances ?? []) {
            if (group.alliance_id) {
                const hit = find('alliance', group.alliance_id)
                if (hit) picked.push(hit)
                continue
            }
            for (const corp of group.corporations ?? []) {
                const hit = find('corporation', corp.corporation_id)
                if (hit) picked.push(hit)
            }
        }
        return picked
    }
    return { sideA: seed(teams[0]), sideB: seed(teams[1]), droppedTeamCount: Math.max(0, teams.length - 2) }
}
