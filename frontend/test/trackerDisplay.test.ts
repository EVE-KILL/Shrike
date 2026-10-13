import {expect,test} from 'vitest'
import {trackerStreamTopics,trackerTargetIcon,trackerTargetImage} from '../src/utils/trackerDisplay'

test('recorded-activity streams include active watches with alerts off',()=>{
    const watch=(type:string,id:number,enabled=true)=>({target_type:type,target_id:id,enabled,notifications_enabled:false}) as any
    expect(trackerStreamTopics([watch('character',42),watch('corporation',42),watch('region',1),watch('constellation',2),watch('system',3),watch('alliance',4,false)])).toEqual(['attacker.42','constellation.2','region.1','system.3','victim.42'])
    expect(trackerTargetImage(watch('corporation',42))).toBe('https://images.evetech.net/corporations/42/logo?size=64')
    expect(trackerTargetImage(watch('region',1))).toBeNull()
    expect(trackerTargetIcon('constellation')).toBe('lucide:orbit')
})

test('browser-alert state honors denied/unsupported permissions and never requests them on load',async()=>{
    const globals=globalThis as any,saved=Object.fromEntries(['window','Notification','localStorage'].map(key=>[key,Object.getOwnPropertyDescriptor(globals,key)]));let requests=0
    class BrowserNotification {static permission='denied';static requestPermission(){requests++;return Promise.resolve(this.permission)}}
    const values=new Map<string,string>()
    try {
        Object.assign(globals,{window:{addEventListener(){},Notification:BrowserNotification},Notification:BrowserNotification,localStorage:{getItem:(key:string)=>values.get(key)||null,setItem:(key:string,value:string)=>values.set(key,value)}})
        const {trackerState,setBrowserAlerts}=await import('../src/browser/tracker-state')
        expect(requests).toBe(0)
        expect(await setBrowserAlerts(true)).toBe(false)
        expect(trackerState.browserPermission).toBe('denied')
        expect(requests).toBe(0)
        BrowserNotification.permission='default'
        expect(await setBrowserAlerts(true)).toBe(false)
        expect(requests).toBe(1)
        BrowserNotification.permission='granted'
        expect(await setBrowserAlerts(true)).toBe(true)
        expect(await setBrowserAlerts(false)).toBe(false)
        delete globals.window.Notification
        expect(await setBrowserAlerts(true)).toBe(false)
        expect(trackerState.browserPermission).toBe('unsupported')
    }finally{for(const [key,descriptor]of Object.entries(saved)){if(descriptor)Object.defineProperty(globals,key,descriptor);else delete globals[key]}}
})
