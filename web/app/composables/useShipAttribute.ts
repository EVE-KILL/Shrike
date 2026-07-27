/**
 * Reactive lookup + formatting for a single named dogma attribute from
 * the current fit's calculated statistics. Ported from
 * @eveshipfit/react's `useAttribute` hook.
 *
 * Returns a `ComputedRef` of `{ value, change }`:
 *
 *   - `value` is the formatted number as a locale-aware string, ready to
 *     drop into a template (e.g. `"57,072"`, `"2.03"`, `"8.50"`).
 *   - `change` is one of `Increase`/`Decrease`/`Unchanged`/`Unknown`,
 *     computed by comparing the current preview-applied value against
 *     `currentStats` (the committed non-preview snapshot). This is what
 *     paints the green/red delta hint when the user hovers a module in
 *     the hardware listing.
 *
 * Formatting rules preserved from the reference:
 *
 *   - Resistance attributes (`isResistance: true`) flip orientation
 *     (`100 - v*100`) and force "high is bad" for change comparison.
 *   - `divideBy` is applied after the resistance flip (e.g. for `ms`
 *     → `s` conversion).
 *   - `roundDown` / `roundUp` force floor/ceil, otherwise default round.
 *   - `-0` is coerced to `0` so the display never shows `"-0"`.
 *   - Resistance values are clamped ≥ 0 (engine can return slightly
 *     negative at perfect 100% resist).
 *
 * Special case: the UI sometimes renders `capacityUsed` which is a
 * derived statistic the engine doesn't compute — the React lib
 * stores it alongside the stats. We don't have that yet, so we treat
 * it as a missing attribute ("?"). When we wire the cargo bay UI
 * we'll compute it in the Vue layer and plumb it through here.
 */

import type { MaybeRefOrGetter } from "vue";

export interface ShipAttributeProps {
    /** Dogma attribute name (e.g. "ehp", "shieldCapacity"). */
    name: string;
    /** Unit postfix, e.g. "hp", "m/s", "dps". */
    unit?: string;
    /** Decimal places for display rounding. */
    fixed: number;
    /** Resistance attributes flip the value and reverse the change direction. */
    isResistance?: boolean;
    /** Divide by this before formatting — used for ms→s, m→km, kg→t. */
    divideBy?: number;
    /** Force floor instead of round. */
    roundDown?: boolean;
    /** Force ceil instead of round. */
    roundUp?: boolean;
}

export enum AttributeChange {
    Increase = "Increase",
    Decrease = "Decrease",
    Unchanged = "Unchanged",
    Unknown = "Unknown",
}

export interface ShipAttributeResult {
    value: string;
    change: AttributeChange;
}

/**
 * Bind a named attribute to a reactive result. `props` can be a
 * static object, a ref, or a getter — the computed re-runs whenever
 * the stats or the props change.
 */
export function useShipAttribute(
    kind: "Ship" | "Char",
    props: MaybeRefOrGetter<ShipAttributeProps>,
): ComputedRef<ShipAttributeResult> {
    const { sde } = useEveData();
    const { stats, currentStats } = useFitStatistics();

    return computed<ShipAttributeResult>(() => {
        const p = toValue(props);
        const s = stats.value;
        const cs = currentStats.value;
        const sdeData = sde.value;

        // Still loading? Return an unknown placeholder — matches the React
        // lib's behavior of rendering "?" when stats are null.
        if (!sdeData || !s) {
            return { value: "?", change: AttributeChange.Unknown };
        }

        const attributeId = sdeData.attributeNameToId.get(p.name);
        if (attributeId === undefined) {
            return { value: "?", change: AttributeChange.Unknown };
        }

        const dogmaAttr = sdeData.dogmaAttributes.get(attributeId);

        const item = kind === "Ship" ? s.hull : s.char;
        const currentItem = cs ? (kind === "Ship" ? cs.hull : cs.char) : undefined;

        const rawValue = item.attributes.get(attributeId)?.value ?? dogmaAttr?.defaultValue ?? 0;
        const rawCurrent =
            currentItem?.attributes.get(attributeId)?.value ?? dogmaAttr?.defaultValue ?? 0;

        if (rawValue === undefined) {
            return { value: "?", change: AttributeChange.Unknown };
        }

        // Change direction: compare raw values (before unit conversions) and
        // honor the attribute's `highIsGood` flag. Resistance attributes and
        // `mass` are always "low is good" regardless of SDE setting.
        let change = AttributeChange.Unchanged;
        if (rawCurrent !== undefined && rawCurrent !== rawValue) {
            const highIsGood =
                p.isResistance || p.name === "mass" ? false : Boolean(dogmaAttr?.highIsGood);
            if (rawCurrent < rawValue) {
                change = highIsGood ? AttributeChange.Increase : AttributeChange.Decrease;
            } else {
                change = highIsGood ? AttributeChange.Decrease : AttributeChange.Increase;
            }
        }

        let v = rawValue;
        if (p.isResistance) v = 100 - v * 100;
        if (p.divideBy) v /= p.divideBy;

        const k = Math.pow(10, p.fixed);
        if (k > 0) {
            if (p.isResistance || p.roundUp) {
                // The -1/(k*10) nudge matches the reference: resistance values
                // sit just below an integer boundary and would otherwise ceil
                // up by one when displayed.
                v -= 1 / k / 10;
                v = Math.ceil(v * k) / k;
            } else if (p.roundDown) {
                v = Math.floor(v * k) / k;
            } else {
                v = Math.round(v * k) / k;
            }
        }

        if (Object.is(v, -0)) v = 0;
        if (p.isResistance) v = Math.max(v, 0);

        return {
            value: v.toLocaleString("en", {
                minimumFractionDigits: p.fixed,
                maximumFractionDigits: p.fixed,
            }),
            change,
        };
    });
}
