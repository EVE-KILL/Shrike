import type { InjectionKey, Ref } from 'vue'

export const killNavigationTokenKey: InjectionKey<Ref<string>> = Symbol('kill-navigation-token')

export interface KillNavigationContext {
    version: 1
    createdAt: number
    returnTo: string
    label: string
    ids: number[]
}
const prefix = 'evekill-kill-navigation:'
const ttl = 2 * 60 * 60 * 1000

export function parseKillNavigation(raw: string | null, now = Date.now()): KillNavigationContext | null {
    try {
        if (!raw || raw.length > 24000) return null
        const context = JSON.parse(raw)
        if (context.version !== 1 || !Number.isFinite(context.createdAt) || context.createdAt > now || now - context.createdAt > ttl) return null
        if (typeof context.returnTo !== 'string' || !/^\/(?!\/)/.test(context.returnTo) || /[\\\r\n]/.test(context.returnTo)) return null
        // Also reject encoded protocol-relative paths and backslashes.
        const decoded = decodeURIComponent(context.returnTo.split('?')[0])
        if (!/^\/(?!\/)/.test(decoded) || /[\\\r\n]/.test(decoded)) return null
        if (typeof context.label !== 'string' || context.label.length > 160) return null
        if (!Array.isArray(context.ids) || context.ids.length < 1 || context.ids.length > 100 || context.ids.some((id: unknown) => !Number.isSafeInteger(id) || Number(id) <= 0)) return null
        if (new Set(context.ids).size !== context.ids.length) return null
        return context
    } catch { return null }
}

export function readKillNavigation(token: unknown): KillNavigationContext | null {
    if (typeof token !== 'string' || !/^[a-zA-Z0-9-]{1,80}$/.test(token)) return null
    try { return parseKillNavigation(sessionStorage.getItem(prefix + token)) } catch { return null }
}

export function writeKillNavigation(token: string, context: KillNavigationContext): boolean {
    try {
        const keys = Object.keys(sessionStorage).filter(key => key.startsWith(prefix))
        const fresh: { key: string; createdAt: number }[] = []
        for (const key of keys) {
            const existing = parseKillNavigation(sessionStorage.getItem(key))
            if (!existing) sessionStorage.removeItem(key)
            else if (key !== prefix + token) fresh.push({ key, createdAt: existing.createdAt })
        }
        fresh.sort((a, b) => b.createdAt - a.createdAt)
        for (const old of fresh.slice(9)) sessionStorage.removeItem(old.key)
        sessionStorage.setItem(prefix + token, JSON.stringify(context))
        return true
    } catch { return false }
}

export function adjacentKill(context: KillNavigationContext, current: number, direction: -1 | 1): number | null {
    const index = context.ids.indexOf(current)
    if (index < 0) return null
    return context.ids[index + direction] ?? null
}
