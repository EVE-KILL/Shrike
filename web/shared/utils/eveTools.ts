export interface EveTool {
    name: string
    label?: string
    url: string
    icon: string
    title: string
}

export const eveTools: EveTool[] = [
    { name: 'DOTLAN', url: 'https://evemaps.dotlan.net/', icon: '/remotes/dotlan.png', title: 'EVE Maps & Intel' },
    { name: 'EVEEye', url: 'https://eveeye.com/', icon: '/remotes/eveeye.svg', title: 'Interactive Maps' },
    { name: 'Missioneer', url: 'https://evemissioneer.com/', icon: '/remotes/evemissioneer.png', title: 'Missions & PvE' },
    { name: 'EveShip.fit', url: 'https://eveship.fit/', icon: '/remotes/eveship-fit.png', title: 'Ship Fitting Tool' },
    { name: 'EVERef', url: 'https://everef.net/', icon: '/remotes/everef.png', title: 'Game Reference' },
    { name: 'Jita.Space', url: 'https://jita.space/', icon: '/remotes/jita-space.png', title: 'Market Analysis' },
    { name: 'EVEWho', url: 'https://evewho.com/', icon: '/remotes/evewho.png', title: 'Character Intel' },
    { name: 'Capsuleers.app', label: 'Capsuleers', url: 'https://capsuleers.app/', icon: '/remotes/capsuleers.png', title: 'EVE Online Community Hub' },
    { name: 'zKillboard', url: 'https://zkillboard.com/', icon: '/remotes/zkillboard.png', title: 'Killmail Database' },
    { name: 'Socket.Kill', url: 'https://socketkill.com/', icon: '/remotes/socketkill.png', title: 'Live Killmail Stream' },
    { name: 'RIFT Intel Fusion', label: 'RIFT', url: 'https://riftforeve.online/', icon: '/images/rift-intel-fusion-tool-256.png', title: 'Intel Fusion Tool' },
    { name: 'Eve Monthly', url: 'https://www.evemonthly.com/', icon: '/remotes/evemonthly.svg', title: 'Coalition Intelligence' },
    { name: 'EVE LKM', url: 'https://eve-lkm.capsuleer.life/', icon: '/remotes/eve-lkm.png', title: 'Last Known Mail' },
    { name: 'Evetools.org', url: 'https://br.evetools.org/', icon: '/remotes/evetools.png', title: 'Battle Reports' },
]
