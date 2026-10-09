// Security status → Three.js hex color
export function secColorHex(sec: number | null): number {
    if (sec == null) return 0x6b7280
    if (sec >= 1.0) return 0x22d3ee
    if (sec >= 0.9) return 0x34d399
    if (sec >= 0.8) return 0x4ade80
    if (sec >= 0.7) return 0xa3e635
    if (sec >= 0.6) return 0xfacc15
    if (sec >= 0.5) return 0xf59e0b
    if (sec >= 0.4) return 0xf97316
    if (sec >= 0.3) return 0xef4444
    if (sec >= 0.2) return 0xdc2626
    if (sec >= 0.1) return 0xb91c1c
    return 0x991b1b
}

// Security status → CSS color string
export function secColorStr(sec: number | null): string {
    if (sec == null) return '#6b7280'
    if (sec >= 1.0) return '#22d3ee'
    if (sec >= 0.9) return '#34d399'
    if (sec >= 0.8) return '#4ade80'
    if (sec >= 0.7) return '#a3e635'
    if (sec >= 0.6) return '#facc15'
    if (sec >= 0.5) return '#f59e0b'
    if (sec >= 0.4) return '#f97316'
    if (sec >= 0.3) return '#ef4444'
    if (sec >= 0.2) return '#dc2626'
    if (sec >= 0.1) return '#b91c1c'
    return '#991b1b'
}

export function secLabel(sec: number | null): string {
    if (sec == null) return '?'
    return sec.toFixed(1)
}

// Celestial group_id → Three.js hex color
export function celestialColorHex(groupId: number): number {
    if (groupId === 6) return 0xfbbf24  // star
    if (groupId === 7) return 0x60a5fa  // planet
    if (groupId === 8) return 0x9ca3af  // moon
    if (groupId === 9) return 0xa78bfa  // belt
    if (groupId === 10) return 0x34d399 // stargate
    if (groupId === 15) return 0xf472b6 // station
    return 0x6b7280
}

export function celestialCategory(groupId: number): string {
    if (groupId === 6) return 'Star'
    if (groupId === 7) return 'Planet'
    if (groupId === 8) return 'Moon'
    if (groupId === 9) return 'Asteroid Belt'
    if (groupId === 10) return 'Stargate'
    if (groupId === 15) return 'Station'
    return 'Celestial'
}

export function celestialPointSize(groupId: number): number {
    if (groupId === 6) return 12
    if (groupId === 7) return 7
    if (groupId === 8) return 4
    if (groupId === 9) return 3
    if (groupId === 10) return 5
    if (groupId === 15) return 5
    return 3
}
