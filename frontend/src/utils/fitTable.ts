export const FIT_TABLE_COLUMN_DEFAULTS: Array<number | null> = [null, 55, 58, 46, 46, 60, 120]
export const FIT_TABLE_COLUMN_MINIMUMS = [140, 42, 48, 40, 40, 54, 80]

export function fitTableColumns(value: unknown): Array<number | null> {
    if (Array.isArray(value) && value.length === FIT_TABLE_COLUMN_DEFAULTS.length
        && value.every((width, index) => width === null && index === 0
            || typeof width === 'number' && Number.isFinite(width) && width >= FIT_TABLE_COLUMN_MINIMUMS[index]!)) return [...value]
    return [...FIT_TABLE_COLUMN_DEFAULTS]
}

export function fitTableMinimumWidth(widths: Array<number | null>): number {
    return widths.reduce<number>((total, width, index) => total + (width ?? FIT_TABLE_COLUMN_MINIMUMS[index]!), 0)
}
