import { expect, test } from 'vitest'
import { attackerImage } from '../src/utils/attackerImage'

test('corporation-owned structure weapons and other modules use icons', () => {
    expect(attackerImage({ ship_type_id: 35928, ship_category_id: 66, corporation_id: 1639878825 }, 512))
        .toBe('/images/types/35928/icon?size=512')
    expect(attackerImage({ ship_type_id: 10, ship_category_id: 7, character_id: 42 }, 128))
        .toBe('/images/types/10/icon?size=128')
})

test('ships and structures retain renders, environment attackers use icons, and missing types stay empty', () => {
    for (const ship_category_id of [6, 65, null])
        expect(attackerImage({ ship_type_id: 23919, ship_category_id, corporation_id: 77 }, 512))
            .toBe('/images/types/23919/overlayrender?size=512')
    expect(attackerImage({ ship_type_id: 100 }, 512)).toBe('/images/types/100/icon?size=512')
    expect(attackerImage({}, 512)).toBe('')
    expect(attackerImage({ ship_type_id: -1 }, 512)).toBe('')
})
