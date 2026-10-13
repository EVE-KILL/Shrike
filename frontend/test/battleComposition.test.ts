// @vitest-environment happy-dom
import { expect, test } from 'vitest'
import { selectBattleCompositionRows } from '../src/browser/battle-composition'

const pilot = (id: number, rank: number, lost: number, damage: number, name: string, group = 30) => ({character_id:id,rank,isk_lost:lost,damage_done:damage,damage_taken:0,character_name:name,corporation_name:'Fleet Corp',ship_name:group===30?'Titan':'Capsule',ship_group_id:group,deaths:lost>0?1:0})
const rows=[pilot(1,94,10,5,'Zulu'),pilot(2,94,20,5,'Beta'),pilot(3,94,20,6,'Alpha'),pilot(3,1,0,1,'Alpha',29)]
const filters={search:'',group:null,losses:false},sort={key:'default',dir:'desc'} as const
test('composition preserves multi-hull observations and the Nuxt rank/value/damage ties',()=>{
 expect(selectBattleCompositionRows({individuals:rows},'individuals',filters,sort).map(p=>p.character_id)).toEqual([3,2,1,3])
 expect(rows[0]?.character_id).toBe(1)
 expect(selectBattleCompositionRows({individuals:rows},'individuals',{...filters,search:'fleet corp',group:30,losses:true},sort)).toHaveLength(3)
 expect(selectBattleCompositionRows({individuals:rows},'individuals',{...filters,search:'capsule',losses:true},sort)).toHaveLength(0)
})
test('ship and group views use their own filtered aggregate totals and stable sort',()=>{
 const ships=[{rank:94,count:1,losses:1,isk_lost:20,name:'Avatar',ship_group_id:30},{rank:94,count:2,losses:0,isk_lost:0,name:'Erebus',ship_group_id:30}]
 expect(selectBattleCompositionRows({by_ship:ships},'by_ship',filters,sort).map(r=>r.name)).toEqual(['Erebus','Avatar'])
 expect(selectBattleCompositionRows({by_ship:ships},'by_ship',{...filters,losses:true},sort).map(r=>r.name)).toEqual(['Avatar'])
 expect(selectBattleCompositionRows({by_ship:ships},'by_ship',filters,{key:'name',dir:'asc'}).map(r=>r.name)).toEqual(['Avatar','Erebus'])
})
