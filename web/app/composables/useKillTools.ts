/**
 * Catalogue of external destinations for a killmail — DOTLAN, EVEEye, EVERef
 * and friends, each with the sub-links that make sense for this kill.
 *
 * Lifted out of the old KillNavbar so the definitions outlive the bar that
 * used to sit across the top of the page.
 */
export interface KillToolContext {
    killmailId: number
    killmailHash: string
    systemName: string | null
    systemId: number
    constellationId: number | null
    constellationName: string | null
    regionName: string | null
    regionId: number | null
    shipTypeId: number | null
    shipGroupId: number | null
    shipGroupName: string | null
    shipName: string | null
    characterId: number | null
    characterName: string | null
    corporationId: number | null
    corporationName: string | null
    allianceId: number | null
    allianceName: string | null
    finalBlowShipTypeId: number | null
    finalBlowShipName: string | null
    topDamageShipTypeId: number | null
    topDamageShipName: string | null
    siblings: {
        killmail_id: number
        ship_type_id: number | null
        ship_group_id: number | null
        ship_name: string | null
        total_value: number
    }[]
}

interface ToolLink {
    name: string
    icon: string
    items: { label: string; desc?: string | null; url: string; external?: boolean; disabled?: boolean }[]
}

function buildToolLinks(k: KillToolContext): ToolLink[] {
    const links: ToolLink[] = []

    // DOTLAN
    links.push({
        name: 'DOTLAN', icon: '/remotes/dotlan.png',
        items: [
            { label: 'System', desc: k.systemName, url: `https://evemaps.dotlan.net/system/${encodeURIComponent(k.systemName || '')}`, external: true, disabled: !k.systemName },
            { label: 'Constellation', desc: k.constellationName, url: `https://evemaps.dotlan.net/map/${encodeURIComponent(k.regionName || '')}/${encodeURIComponent(k.constellationName || '')}`, external: true, disabled: !k.constellationName },
            { label: 'Region', desc: k.regionName, url: `https://evemaps.dotlan.net/region/${encodeURIComponent(k.regionName || '')}`, external: true, disabled: !k.regionName },
        ],
    })

    // EVEEye
    links.push({
        name: 'EVEEye', icon: '/remotes/eveeye.svg',
        items: [
            { label: 'Region', desc: k.regionName, url: `https://eveeye.com/?m=${encodeURIComponent(k.regionName || '')}`, external: true, disabled: !k.regionName },
        ],
    })

    // EVE Missioneer
    links.push({
        name: 'Missioneer', icon: '/remotes/evemissioneer.png',
        items: [
            { label: 'System', desc: k.systemName, url: `https://evemissioneer.com/s/${k.systemId}`, external: true },
            { label: 'Region', desc: k.regionName, url: `https://evemissioneer.com/r/${k.regionId}`, external: true, disabled: !k.regionId },
        ],
    })

    // EveShip.fit
    links.push({
        name: 'EveShip.fit', icon: '/remotes/eveship-fit.png',
        items: [
            { label: 'Fitting', desc: k.shipName, url: `https://eveship.fit/fit/${k.killmailId}`, external: true, disabled: !k.shipTypeId },
        ],
    })

    // EVERef
    links.push({
        name: 'EVERef', icon: '/remotes/everef.png',
        items: [
            ...(k.shipGroupId ? [{ label: 'Ship Group', desc: k.shipGroupName, url: `https://everef.net/group/${k.shipGroupId}`, external: true }] : []),
            ...(k.shipTypeId ? [{ label: 'Ship', desc: k.shipName, url: `https://everef.net/type/${k.shipTypeId}`, external: true }] : []),
        ],
    })

    // Jita.Space
    links.push({
        name: 'Jita.Space', icon: '/remotes/jita-space.png',
        items: [
            { label: 'System', desc: k.systemName, url: `https://www.jita.space/system/${k.systemId}`, external: true },
            { label: 'Constellation', desc: k.constellationName, url: `https://www.jita.space/constellation/${k.constellationId}`, external: true, disabled: !k.constellationId },
            { label: 'Region', desc: k.regionName, url: `https://www.jita.space/region/${k.regionId}`, external: true, disabled: !k.regionId },
        ],
    })

    // EVEWho
    links.push({
        name: 'EVEWho', icon: '/remotes/evewho.png',
        items: [
            { label: 'Character', desc: k.characterName, url: `https://evewho.com/character/${k.characterId}`, external: true, disabled: !k.characterId },
            { label: 'Corporation', desc: k.corporationName, url: `https://evewho.com/corporation/${k.corporationId}`, external: true, disabled: !k.corporationId },
            { label: 'Alliance', desc: k.allianceName, url: `https://evewho.com/alliance/${k.allianceId}`, external: true, disabled: !k.allianceId },
        ],
    })

    // zKillboard
    links.push({
        name: 'zKillboard', icon: '/remotes/zkillboard.png',
        items: [
            { label: 'Killmail', desc: `#${k.killmailId}`, url: `https://zkillboard.com/kill/${k.killmailId}/`, external: true },
            { label: 'System', desc: k.systemName, url: `https://zkillboard.com/system/${k.systemId}/`, external: true },
            { label: 'Constellation', desc: k.constellationName, url: `https://zkillboard.com/constellation/${k.constellationId}/`, external: true, disabled: !k.constellationId },
            { label: 'Region', desc: k.regionName, url: `https://zkillboard.com/region/${k.regionId}/`, external: true, disabled: !k.regionId },
            { label: 'Ship Group', desc: k.shipGroupName, url: `https://zkillboard.com/group/${k.shipGroupId}/`, external: true, disabled: !k.shipGroupId },
            { label: 'Ship', desc: k.shipName, url: `https://zkillboard.com/ship/${k.shipTypeId}/`, external: true, disabled: !k.shipTypeId },
            { label: 'Character', desc: k.characterName, url: `https://zkillboard.com/character/${k.characterId}/`, external: true, disabled: !k.characterId },
            { label: 'Corporation', desc: k.corporationName, url: `https://zkillboard.com/corporation/${k.corporationId}/`, external: true, disabled: !k.corporationId },
            { label: 'Alliance', desc: k.allianceName, url: `https://zkillboard.com/alliance/${k.allianceId}/`, external: true, disabled: !k.allianceId },
        ],
    })

    // kb.evetools.org
    links.push({
        name: 'kb.evetools', icon: '/remotes/evetools.png',
        items: [
            { label: 'Killmail', desc: `#${k.killmailId}`, url: `https://kb.evetools.org/kill/${k.killmailId}`, external: true },
            { label: 'Victim Ship', desc: k.shipName, url: `https://kb.evetools.org/ship/${k.shipTypeId}`, external: true, disabled: !k.shipTypeId },
            { label: 'Final Blow Ship', desc: k.finalBlowShipName, url: `https://kb.evetools.org/ship/${k.finalBlowShipTypeId}`, external: true, disabled: !k.finalBlowShipTypeId },
            { label: 'Top Damage Ship', desc: k.topDamageShipName, url: `https://kb.evetools.org/ship/${k.topDamageShipTypeId}`, external: true, disabled: !k.topDamageShipTypeId },
            { label: 'Character', desc: k.characterName, url: `https://kb.evetools.org/character/${k.characterId}`, external: true, disabled: !k.characterId },
            { label: 'Corporation', desc: k.corporationName, url: `https://kb.evetools.org/corporation/${k.corporationId}`, external: true, disabled: !k.corporationId },
            { label: 'Alliance', desc: k.allianceName, url: `https://kb.evetools.org/alliance/${k.allianceId}`, external: true, disabled: !k.allianceId },
            { label: 'System', desc: k.systemName, url: `https://kb.evetools.org/system/${k.systemId}`, external: true, disabled: !k.systemId },
            { label: 'Region', desc: k.regionName, url: `https://kb.evetools.org/region/${k.regionId}`, external: true, disabled: !k.regionId },
        ],
    })

    // SocketKill
    links.push({
        name: 'SocketKill', icon: '/remotes/socketkill.png',
        items: [
            { label: 'Killmail', desc: `#${k.killmailId}`, url: `https://socketkill.com/kill/${k.killmailId}`, external: true },
        ],
    })

    // ESI
    links.push({
        name: 'ESI', icon: '',
        items: [
            { label: 'Raw Killmail', desc: 'JSON from CCP', url: `https://esi.evetech.net/latest/killmails/${k.killmailId}/${k.killmailHash}/`, external: true },
        ],
    })

    return links
        .map(t => ({ ...t, items: t.items.filter(i => !i.disabled) }))
        .filter(t => t.items.length > 0)
}

export function useKillTools(ctx: MaybeRefOrGetter<KillToolContext>) {
    return computed<ToolLink[]>(() => {
        const k = toValue(ctx)
        return buildToolLinks(k)
    })
}
