export const mapScopes=['new-eden','wormhole','zarzakh','abyssal','proving'] as const
export type MapScope=typeof mapScopes[number]
export const isMapScope=(value:unknown):value is MapScope=>mapScopes.some(scope=>scope===value)
