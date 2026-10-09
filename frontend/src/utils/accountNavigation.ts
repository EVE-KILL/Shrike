// Constants and badge formatting taken from the original Navbar.vue.
export const LOGIN_PREFS_DEFAULTS = {charKm:true,corpKm:true,delay:0}
export const ALLOWED_DELAYS = [0,1,3,6,12,24,72]
export function formatWalletBalance(value:number|null,loading=false):string {
    if (value === null) return loading ? '…' : '— ISK'

    const absolute = Math.abs(value)
    const units = [
        { value: 1e12, suffix: 't' },
        { value: 1e9, suffix: 'b' },
        { value: 1e6, suffix: 'm' },
        { value: 1e3, suffix: 'k' },
    ]
    const unit = units.find(candidate => absolute >= candidate.value)
    const amount = unit ? `${Math.round(value / unit.value)}${unit.suffix}` : Math.round(value).toLocaleString('en-US')
    return `${amount} ISK`
 }
