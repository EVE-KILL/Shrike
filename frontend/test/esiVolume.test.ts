import {expect,test} from 'vitest'
import {esiVolumeSlots} from '../src/utils/esiVolume'
import {esiRowOk,esiStatusDotClass} from '../src/utils/esiLog'

test('ESI chart retains quiet hours and parses PostgreSQL offsets in UTC',()=>{
    const hours=[{hour:'2026-10-01 00:00:00+00',total:1,errors:0,new_items:2},{hour:'2026-10-01T23:00:00Z',total:10,errors:2,new_items:7}]
    const slots=esiVolumeSlots(hours,true)
    expect(slots).toHaveLength(24)
    expect(slots.filter(s=>s.empty)).toHaveLength(22)
    expect(slots[0]?.at).toBe(Date.parse('2026-10-01T00:00:00Z'))
    expect(slots[23]?.errorHeight).toBeCloseTo(19.2)
    expect(slots[23]?.okHeight).toBeCloseTo(76.8)
    expect(slots[23]?.tooltip).toBe('23:00: 10 requests, 2 errors, 7 new items')
    expect(slots[1]?.tooltip).toBe('01:00 — no requests logged')
    expect(esiVolumeSlots([{hour:'invalid',total:1,errors:0}])).toEqual([])
    expect(esiVolumeSlots([])).toEqual([])
})

test('ESI conditional requests stay successful in the shared row vocabulary',()=>{
    expect(esiRowOk({success:false,status_code:304})).toBe(true)
    expect(esiStatusDotClass({success:false,status_code:304})).toBe('bg-green-400')
    expect(esiStatusDotClass({success:false,status_code:503})).toBe('bg-red-400')
    expect(esiStatusDotClass({success:false,status_code:403})).toBe('bg-yellow-400')
})
