import { expect, test } from 'vitest'
import { FIT_TABLE_COLUMN_DEFAULTS, fitTableColumns, fitTableMinimumWidth } from '../src/utils/fitTable'

test('table preserves a readable Name column and saved resize widths', () => {
    expect(fitTableMinimumWidth(FIT_TABLE_COLUMN_DEFAULTS)).toBe(525)
    const resized = [160, 70, 80, 50, 50, 70, 120]
    expect(fitTableColumns(resized)).toEqual(resized)
    expect(fitTableMinimumWidth(resized)).toBe(600)
    expect(fitTableColumns(resized)).not.toBe(resized)
})

test('stale or invalid column preferences fall back safely', () => {
    for (const value of [null, [100], ['bad',55,58,46,46,60,120], [null,null,58,46,46,60,120], [null,55,58,46,46,60,-1], [Infinity,55,58,46,46,60,120]]) {
        expect(fitTableColumns(value)).toEqual(FIT_TABLE_COLUMN_DEFAULTS)
    }
})
