import { describe, expect, test } from 'bun:test'
import { debugLoggingEnabled, logRequestError } from '../server/utils/errorLogging'

describe('server request error logging', () => {
    test('info logging emits one concise line without the error object', () => {
        const calls: unknown[][] = []
        const error = Object.assign(new Error('Post not found'), { fatal: true })

        logRequestError(
            error,
            'GET',
            'https://localhost:4001/blog/missing',
            'info',
            (...values) => calls.push(values),
        )

        expect(calls).toEqual([[
            '[request error] [fatal] [GET] https://localhost:4001/blog/missing: Post not found',
        ]])
    })

    test('debug and trace logging retain the error object and stack', () => {
        for (const level of ['debug', 'trace']) {
            const calls: unknown[][] = []
            const error = Object.assign(new Error('Post not found'), { fatal: true })

            logRequestError(
                error,
                'GET',
                'https://localhost:4001/blog/missing',
                level,
                (...values) => calls.push(values),
            )

            expect(calls).toHaveLength(1)
            expect(calls[0]).toHaveLength(2)
            expect(calls[0]?.[0]).toBe(
                '[request error] [fatal] [GET] https://localhost:4001/blog/missing',
            )
            expect(calls[0]?.[1]).toBe(error)
            expect((calls[0]?.[1] as Error).stack).toContain('Post not found')
        }
    })

    test('only debug and trace enable detailed logging', () => {
        expect(debugLoggingEnabled('debug')).toBe(true)
        expect(debugLoggingEnabled('TRACE')).toBe(true)
        expect(debugLoggingEnabled('info')).toBe(false)
        expect(debugLoggingEnabled(undefined)).toBe(false)
    })
})
