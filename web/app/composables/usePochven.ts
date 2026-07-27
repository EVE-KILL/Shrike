const POCHVEN_REGION_ID = 10000070

export function pochvenClass(regionId: number | null | undefined): string {
    return regionId === POCHVEN_REGION_ID ? 'font-triglavian' : ''
}
