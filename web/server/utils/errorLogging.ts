export type RequestErrorLogSink = (...values: unknown[]) => void

export interface RequestErrorLike {
    message?: string
    fatal?: boolean
    unhandled?: boolean
}

export function debugLoggingEnabled(logLevel = process.env.LOG_LEVEL): boolean {
    const normalized = logLevel?.trim().toLowerCase()
    return normalized === 'debug' || normalized === 'trace'
}

export function logRequestError(
    error: RequestErrorLike,
    method: string,
    url: string,
    logLevel = process.env.LOG_LEVEL,
    sink: RequestErrorLogSink = console.error,
): void {
    const tags = [
        '[request error]',
        error.unhandled && '[unhandled]',
        error.fatal && '[fatal]',
        `[${method}]`,
        url,
    ].filter(Boolean).join(' ')

    if (debugLoggingEnabled(logLevel)) {
        sink(tags, error)
        return
    }

    sink(`${tags}: ${error.message || 'Server Error'}`)
}
