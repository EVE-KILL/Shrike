export const riverStates = ['available', 'running', 'scheduled', 'retryable', 'discarded', 'cancelled', 'completed'] as const
export type RiverState = typeof riverStates[number]
export type RiverDepth = Record<RiverState, number>
export type RiverQueue = {
  name: string; depth: RiverDepth; cron: boolean; concurrency?: number; description?: string
  paused_at?: string | null; worker_updated_at?: string | null; worker_active?: boolean
}
export type RiverJob = {
  id: number; queue: string; kind: string; state: string; attempt: number; max_attempts: number; priority: number
  created_at: string; scheduled_at: string; attempted_at?: string | null; finalized_at?: string | null
  attempted_by: string[]; args: unknown; errors: Array<{ error?: string; at?: string; attempt?: number }>
  output?: unknown; metadata: unknown
}
const stateClasses: Record<string, string> = {
  completed: 'text-green-400', running: 'text-blue-400', discarded: 'text-red-400', retryable: 'text-yellow-400',
  cancelled: 'text-gray-500', available: 'text-cyan-400', scheduled: 'text-purple-400',
}
export const riverStateClass = (state: string) => stateClasses[state] ?? 'text-gray-400'
export const riverPending = (queue: RiverQueue) => ['available', 'running', 'scheduled', 'retryable'].reduce((sum, state) => sum + (queue.depth[state as RiverState] ?? 0), 0)
export const riverStamp = (value?: string | null) => value ? new Date(value).toLocaleString() : '—'
export const riverJSON = (value: unknown) => JSON.stringify(value, null, 2) ?? ''
export const riverCanCancel = (state: string) => ['running', 'available', 'scheduled', 'retryable'].includes(state)
