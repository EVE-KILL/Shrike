import { expect, test } from 'vitest'
import { createReloadChannel } from '../src/browser/reload-channel'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'
import {transformSync} from 'esbuild'

function port() {
    return { messages: [] as string[], started: false, closed: false, onmessage: null as null | ((event: {data: string}) => void),
        postMessage(value: string) { this.messages.push(value) }, start() { this.started = true }, close() { this.closed = true } }
}

test('seven documents share one reload stream and the last departure closes it', () => {
    const streams: { close: () => void; addEventListener: (event: string, callback: (event: {data: string}) => void) => void; closed: boolean; build?: (event: {data: string}) => void }[] = []
    const channel = createReloadChannel(() => {
        const stream = { closed: false, build: undefined as undefined | ((event: {data: string}) => void), close() { this.closed = true }, addEventListener(_event: string, callback: (event: {data: string}) => void) { this.build = callback } }
        streams.push(stream); return stream
    })
    const clients = Array.from({length: 7}, port)
    clients.forEach(client => channel.connect(client))
    expect(streams).toHaveLength(1)
    streams[0]!.build!({data: 'next-process'})
    expect(clients.every(client => client.started && client.messages[0] === 'next-process')).toBe(true)
    clients.slice(0,6).forEach(client => client.onmessage!({data:'disconnect'}))
    expect(streams[0]!.closed).toBe(false)
    clients[6]!.onmessage!({data:'disconnect'})
    expect(streams[0]!.closed).toBe(true)
    expect(clients.every(client => client.closed)).toBe(true)
    channel.connect(port())
    expect(streams).toHaveLength(2)
})

test('a new document never receives a stale cached build from the shared worker', () => {
    let build: ((event: {data: string}) => void) | undefined
    const channel = createReloadChannel(() => ({close(){}, addEventListener(_name: string, callback: (event: {data: string}) => void){build=callback}}))
    const old = port(),fresh = port()
    channel.connect(old);build!({data:'old-process'});channel.connect(fresh)
    expect(fresh.messages).toEqual([])
    build!({data:'new-process'})
    expect(old.messages).toEqual(['old-process','new-process'])
    expect(fresh.messages).toEqual(['new-process'])
})

for (const unavailable of ['missing', 'blocked']) test(`reload falls back to finite reads when SharedWorker is ${unavailable}`, async () => {
    const timers=new Map<number,{callback:()=>void; delay:number}>(),events=new Map<string,(event:{persisted?:boolean})=>void>()
    let timerId=0,version='first-process',reloads=0,reads=0
    const window={SharedWorker:unavailable==='missing'?undefined:class{constructor(){throw new Error('Worker blocked')}},location:{reload(){reloads++}},addEventListener(name:string,callback:(event:{persisted?:boolean})=>void){events.set(name,callback)}}
    runInNewContext(transformSync(readFileSync(new URL('../src/browser/reload.ts',import.meta.url),'utf8'),{loader:'ts',format:'iife'}).code,{
        document:{querySelector(){return{content:'first-process'}}},window,SharedWorker:window.SharedWorker,AbortController,
        async fetch(path:string,options:{cache:string;signal:AbortSignal}){
            expect(path).toBe('/__dev/version');expect(options.cache).toBe('no-store');reads++
            return{ok:true,async text(){return version}}
        },
        setTimeout(callback:()=>void,delay:number){const id=++timerId;timers.set(id,{callback,delay});return id},clearTimeout(id:number){timers.delete(id)},
    })
    const settle=()=>new Promise<void>(resolve=>setImmediate(resolve))
    await settle()
    expect(reads).toBe(1);expect(reloads).toBe(0)
    expect([...timers.values()].map(t=>t.delay)).toEqual([1500])
    events.get('pagehide')!({})
    expect(timers.size).toBe(0)
    events.get('pageshow')!({persisted:true})
    await settle();expect(reads).toBe(2)
    version='next-process'
    const next=[...timers.values()].find(t=>t.delay===1500)!
    timers.clear();next.callback();await settle()
    expect(reads).toBe(3);expect(reloads).toBe(1);expect(timers.size).toBe(0)
})
