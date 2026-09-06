import { filterBattleLosses, type BattleSide } from '~/utils/battleExplorer'
import type { ReplayKill } from '~/utils/map/replay'
export function useBattleExplorer(kills: Ref<ReplayKill[]>, teams: () => BattleSide[]) {
    const route = useRoute()
    const router = useRouter()
    const number = (key: string) => {
        const value = route.query[key]
        if (typeof value !== 'string' || !value.trim()) return null
        const parsed = Number(value)
        return Number.isFinite(parsed) ? parsed : null
    }
    const filters = computed(() => ({ side: number('side'), group: number('group'), minIsk: Math.max(0, number('minIsk') ?? 0), from: number('from'), to: number('to') }))
    const filtered = computed(() => filterBattleLosses(kills.value, teams(), filters.value))
    const setFilter = (key: string, value: string | number | null) => router.replace({ query: { ...route.query, [key]: value == null || value === '' ? undefined : String(value) } })
    const clear = () => router.replace({ query: { ...route.query, side: undefined, group: undefined, minIsk: undefined, from: undefined, to: undefined } })
    const location = (tab: string, extra: Record<string, string | number | undefined> = {}) => ({ path: `/battle/${route.params.id}/${tab}`, query: { ...route.query, ...extra } })
    const replayLocation = (kill: ReplayKill) => location('replay', { at: Date.parse(kill.killmail_time), kill: kill.killmail_id })
    return { filters, filtered, setFilter, clear, location, replayLocation }
}
