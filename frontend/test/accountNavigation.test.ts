import {expect,test} from 'vitest'
import {ALLOWED_DELAYS,formatWalletBalance} from '../src/utils/accountNavigation'
import {targetFor,trackerTopics} from '../src/utils/notificationDisplay'

test('account badge handles loading, unavailable, reserved-balance input and ISK units',()=>{
    expect(formatWalletBalance(null,true)).toBe('…')
    expect(formatWalletBalance(null)).toBe('— ISK')
    for(const [balance,label] of [[0,'0 ISK'],[96000.5,'96k ISK'],[1200000,'1m ISK'],[2e9,'2b ISK'],[3e12,'3t ISK'],[-4000,'-4k ISK']] as const){
        expect(formatWalletBalance(balance)).toBe(label)
    }
    expect(ALLOWED_DELAYS).toEqual([0,1,3,6,12,24,72])
})

test('reply links reach every supported comment target and preserve encoded slugs',()=>{
    const cases:Array<[number,number,string|null,string]>=[
        [1,100,null,'/kill/100#comment-42'],[2,100,null,'/character/100#comment-42'],
        [3,100,null,'/corporation/100#comment-42'],[4,100,null,'/alliance/100#comment-42'],
        [5,100,null,'/system/100#comment-42'],[6,0,'/about','/about#comment-42'],
        [7,100,'report ID','/battle/report%20ID/comments#comment-42'],
        [8,0,'fit?hash','/fit/fit%3Fhash#comment-42'],
        [9,91,'article#one','/blog/article%23one#comment-42'],
        [10,0,'campaign ID','/campaign/campaign%20ID/comments#comment-42'],
    ]
    for(const [type,id,slug,href] of cases)expect(targetFor(type).pathFn(id,42,slug)).toBe(href)
    expect(targetFor(6).pathFn(0,42,'//example.com/path')).toBe('/example.com/path#comment-42')
    expect(targetFor(8).pathFn(0,42,null)).toBe('#')
})

test('tracker relay subscribes only to enabled opt-ins and deduplicates entity topics',()=>{
    const tracker=(target_type:string,target_id:number,enabled=true,notifications_enabled=true)=>({target_type,target_id,enabled,notifications_enabled})
    expect(trackerTopics([tracker('character',42),tracker('corporation',42),tracker('system',30000142),tracker('region',1,false),tracker('alliance',2,true,false)])).toEqual(['attacker.42','system.30000142','victim.42'])
    expect(trackerTopics([])).toEqual([])
})
