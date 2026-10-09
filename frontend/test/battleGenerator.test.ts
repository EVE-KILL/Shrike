import { describe, expect, test } from 'vitest'
import { generatorCanLoad, generatorEntityList, generatorKeepAssignments, generatorMove, generatorSeedTeams, generatorTimeError, generatorUndecided, eveInputToIso, type GeneratorEntity } from '../src/utils/battleGenerator'

const red: GeneratorEntity={id:1,name:'Red Alliance',type:'alliance'}
const blue: GeneratorEntity={id:2,name:'Blue Alliance',type:'alliance'}
const corp: GeneratorEntity={id:3,name:'Independent',type:'corporation',alliance_name:'Old Alliance'}
describe('shared battle generator state',()=>{
    test('UTC windows keep the entered EVE time and enforce the picker limits',()=>{
        expect(eveInputToIso('2026-09-24T12:00')).toBe('2026-09-24T12:00:00.000Z')
        expect(generatorCanLoad([{id:30000142}],'2026-09-24T12:00','2026-09-26T00:00')).toBe(true)
        expect(generatorTimeError('2026-09-24T12:00','2026-09-26T00:01')).toBe('Maximum 36 hour window')
        expect(generatorTimeError('2026-09-24T12:00','2026-09-24T11:00')).toBe('End must be after start')
        expect(generatorCanLoad([],'2026-09-24T12:00','2026-09-24T13:00')).toBe(false)
        expect(generatorCanLoad([{id:1}],'','2026-09-24T13:00')).toBe(false)
    })
    test('loaded alliance corporations stay grouped while window changes preserve matching assignments',()=>{
        const list=generatorEntityList({factions:[],alliances:[red,blue],corporations:[{id:4,name:'Red Fleet',type:'corporation',alliance_id:1},corp]})
        expect(list).toEqual([red,blue,corp])
        expect(generatorKeepAssignments([red,blue], [red,corp])).toEqual([red])
        expect(generatorUndecided(list,[red],[], 'old')).toEqual([corp])
        expect(generatorUndecided(list,[red],[blue],'')).toEqual([corp])
    })
    test('moving or removing an entity cannot leave it on both sides',()=>{
        const result=generatorMove([red],[blue],red,'b')
        expect(result.sideA).toEqual([]);expect(result.sideB).toEqual([blue,red])
        expect(generatorMove(result.sideA,result.sideB,red,'u').sideB).toEqual([blue])
    })
    test('editing seeds the first two teams and leaves extra teams visible as unassigned',()=>{
        const result=generatorSeedTeams([red,blue,corp],[{alliances:[{alliance_id:1}]},{alliances:[{alliance_id:null,corporations:[{corporation_id:3}]}]},{alliances:[{alliance_id:2}]}])
        expect(result.sideA).toEqual([red]);expect(result.sideB).toEqual([corp]);expect(result.droppedTeamCount).toBe(1)
        expect(generatorUndecided([red,blue,corp],result.sideA,result.sideB,'')).toEqual([blue])
    })
})
