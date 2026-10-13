import { describe, expect, test } from 'vitest'
import { selectRoamCompositionRows, readRoamComposition } from '../src/browser/shared/roam-composition'

const pilot = (id:number,name:string,corporation='Fleet Corp') => ({ character_id:id,name,corporation_name:corporation })
const ship = (id:number,group:number,kills:number,losses=0) => ({ship_type_id:id,ship_name:id===587?'Rifter':'Thrasher',ship_group_id:group,ship_group_name:group===25?'Frigate':'Destroyer',killmails:kills,losses,damage_done:kills*1000})
const rows = [
  {pilot:pilot(1,'Alpha'),ship:ship(587,25,2,1),iskLost:25e6},
  {pilot:pilot(1,'Alpha'),ship:ship(16242,420,1),iskLost:0},
  {pilot:pilot(2,'Beta'),ship:ship(587,25,3),iskLost:0},
  {pilot:pilot(3,'Unobserved'),ship:null,iskLost:0},
]

describe('Roam Report composition parity',()=>{
  test('keeps all roster pilots and supports name/corporation/class/loss filters',()=>{
    expect(selectRoamCompositionRows(rows).map(row=>row.pilot.name)).toEqual(['Beta','Alpha','Alpha','Unobserved'])
    expect(selectRoamCompositionRows(rows,{search:'unobserved'})).toHaveLength(1)
    expect(selectRoamCompositionRows(rows,{search:'fleet corp'})).toHaveLength(4)
    expect(selectRoamCompositionRows(rows,{group:25})).toHaveLength(2)
    expect(selectRoamCompositionRows(rows,{losses:true})).toHaveLength(1)
  })
  test('aggregates only the filtered observations and counts pilots once',()=>{
    const ships=selectRoamCompositionRows(rows,{view:'ships'})
    expect(ships[0].name).toBe('Rifter')
    expect(ships[0].pilotIds.size).toBe(2)
    expect(ships[0].observations).toBe(5)
    expect(ships[0].iskLost).toBe(25e6)
    const filtered=selectRoamCompositionRows(rows,{view:'groups',search:'Alpha'})
    expect(filtered).toHaveLength(2)
    expect(filtered.every(row=>row.pilotIds.size===1)).toBe(true)
    expect(selectRoamCompositionRows(rows,{view:'ships',search:'Unobserved'})).toEqual([])
  })
  test('toggles numeric/name sort direction without mutating source observations',()=>{
    expect(selectRoamCompositionRows(rows,{sort:'observations',descending:false})[0].pilot.name).toBe('Unobserved')
    expect(selectRoamCompositionRows(rows,{sort:'pilot',descending:false})[0].pilot.name).toBe('Alpha')
    expect(selectRoamCompositionRows(rows,{view:'ships',sort:'observations',descending:false})[0].name).toBe('Thrasher')
    expect(rows[0].pilot.name).toBe('Alpha')
    expect(rows[0].ship?.killmails).toBe(2)
  })
})


test('roam composition transport retains unobserved pilots and normalizes unsafe counters',()=>{
 const data=readRoamComposition({pilots:2,rows:[{pilot:pilot(1,'Alpha'),ship:null,iskLost:NaN},{pilot:pilot(2,'Beta'),ship:{...ship(587,25,2),damage_done:Infinity},iskLost:25e6},null]});expect(data.rows).toHaveLength(2);expect(data.rows[0]).toMatchObject({pilot:{name:'Alpha'},ship:null,iskLost:0});expect(data.rows[1].ship?.damage_done).toBe(0);expect(selectRoamCompositionRows(data.rows,{search:'Alpha'})).toHaveLength(1)
})
test('roam composition rejects malformed transport instead of silently discarding the roster',()=>{
 expect(()=>readRoamComposition(null)).toThrow('unavailable');expect(()=>readRoamComposition({rows:'bad'})).toThrow('unavailable')
})
