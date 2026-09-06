import type { InjectionKey, Ref } from 'vue'

/** One clock per list, shared only with rows that have hydrated. */
export const killlistClockKey: InjectionKey<Ref<number>> = Symbol('killlistClock')
