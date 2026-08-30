import { describe, expect, test } from 'bun:test'

describe('authentication routes', () => {
    test('does not use the removed API-prefixed namespace', async () => {
        const legacyReferences: string[] = []
        const sourceFiles = new Bun.Glob('**/*.{ts,vue}')

        for await (const path of sourceFiles.scan('app')) {
            const source = await Bun.file(`app/${path}`).text()
            if (source.includes('/api/auth/')) legacyReferences.push(path)
        }

        expect(legacyReferences).toEqual([])
    })
})
