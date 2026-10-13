interface AttackerImageIdentity {
    character_id?: number | null
    corporation_id?: number | null
    ship_type_id?: number | null
    ship_category_id?: number | null
}

export function attackerImage(attacker: AttackerImageIdentity, size: number): string {
    if (!attacker.ship_type_id || attacker.ship_type_id <= 0) return ''
    const category = attacker.ship_category_id
    const icon = (!attacker.character_id && !attacker.corporation_id) ||
        (category != null && category !== 6 && category !== 65)
    return `/images/types/${attacker.ship_type_id}/${icon ? 'icon' : 'overlayrender'}?size=${size}`
}
