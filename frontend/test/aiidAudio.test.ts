import { expect, test } from 'vitest'
import { createAIIDAudio } from '../src/utils/map/aiidAudioCore'

test('alarms use distinct pulses, independent rate limits and allow the first immediate alert', () => {
    let clock = 10
    const pulses: {type: string;frequency?: number;start?: number;stop?: number;gain?: number}[] = []
    const current = {
        state: 'running', currentTime: 2, destination: {},
        createOscillator() {
            const pulse = {type: ''} as typeof pulses[number];pulses.push(pulse)
            return {set type(t:string){pulse.type=t},frequency:{setValueAtTime(n:number){pulse.frequency=n},exponentialRampToValueAtTime(){}},connect(){},start(n:number){pulse.start=n},stop(n:number){pulse.stop=n}}
        },
        createGain(){return{gain:{setValueAtTime(){},exponentialRampToValueAtTime(n:number){if(n>.001)pulses.at(-1)!.gain=n}},connect(){}}},
    }
    const audio = createAIIDAudio(() => current as unknown as AudioContext, () => clock)
    audio.playAIIDAlarm('near')
    expect(pulses.map(p=>p.frequency)).toEqual([920,690,920])
    expect(pulses.map(p=>p.type)).toEqual(['sawtooth','sawtooth','sawtooth'])
    expect(pulses[0]!.start).toBeCloseTo(2.015)
    expect(pulses[2]!.stop).toBeCloseTo(2.605)
    expect(pulses[0]!.gain).toBe(.055)
    audio.playAIIDAlarm('near');expect(pulses).toHaveLength(3)
    audio.playAIIDAlarm('outer');expect(pulses.slice(3).map(p=>[p.frequency,p.type,p.gain])).toEqual([[560,'sine',.035],[740,'triangle',.035]])
    clock=759;audio.playAIIDAlarm('near');expect(pulses).toHaveLength(5)
    clock=760;audio.playAIIDAlarm('near');expect(pulses).toHaveLength(8)
})

test('suspended audio is silent until primed, with one context and no server-side constructor', async () => {
    let creates=0,resumes=0,oscillators=0
    const current={state:'suspended',currentTime:0,destination:{},async resume(){resumes++;this.state='running'},createOscillator(){oscillators++;return{frequency:{setValueAtTime(){},exponentialRampToValueAtTime(){}},connect(){},start(){},stop(){}}},createGain(){return{gain:{setValueAtTime(){},exponentialRampToValueAtTime(){}},connect(){}}}}
    const audio=createAIIDAudio(()=>{creates++;return current as unknown as AudioContext})
    audio.playAIIDAlarm('near');expect(oscillators).toBe(0)
    await audio.primeAIIDAudio();await audio.primeAIIDAudio()
    expect([creates,resumes]).toEqual([1,1])
    audio.playAIIDAlarm('outer');expect(oscillators).toBe(2)
    const server=createAIIDAudio(()=>null)
    await server.primeAIIDAudio();expect(()=>server.playAIIDAlarm('near')).not.toThrow()
})
