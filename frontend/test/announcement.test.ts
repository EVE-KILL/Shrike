import { expect, test } from 'vitest'
import type { Announcement } from '../shared/api/index'
import type { AnnouncementEvent } from '../src/utils/announcement'
import { announcementLists, applyAnnouncementEvent, reconcileAnnouncementSnapshot, tickerFreshness } from '../src/utils/announcement'

const now = Date.parse('2026-10-04T12:00:00Z')
const row = (id: number, tier: 1|2|3 = 1, created = now + id): Announcement => ({ id,tier,title:'Fixture',body_md:'',body_html:'',color:'info',icon:null,link_url:null,link_label:null,starts_at:new Date(now-1000).toISOString(),expires_at:new Date(now+3600000).toISOString(),created_at:new Date(created).toISOString() })
test('announcement buckets cap each category independently and respect expiry, schedule and dismissals', () => {
    const items = [...Array.from({length:20},(_,i)=>row(-1000000000-i)),...Array.from({length:10},(_,i)=>row(-2000000000-i)),...Array.from({length:10},(_,i)=>row(-3000000000-i)),row(1,2),row(2,3),{...row(3,2),expires_at:new Date(now).toISOString()},{...row(4,3),starts_at:new Date(now+1).toISOString()}]
    const lists = announcementLists(items,new Set([-1000000000,1]),now)
    expect(lists.ticker).toHaveLength(17)
    expect(lists.ticker.filter(a=>a.id<=-3000000000)).toHaveLength(4)
    expect(lists.ticker.filter(a=>a.id<=-2000000000&&a.id>-3000000000)).toHaveLength(5)
    expect(lists.ticker.some(a=>a.id===-1000000000)).toBe(false)
    expect(lists.banners).toHaveLength(0)
    expect(lists.modals.map(a=>a.id)).toEqual([2])
})
test('live upserts preserve ordering and a slow snapshot cannot resurrect an archived row or lose a live insertion', () => {
    const before=[row(1)]
    const updated=applyAnnouncementEvent(before,'announcements',{event_type:'updated',announcement:{...row(1),title:'Updated'}})
    expect(updated[0]?.title).toBe('Updated')
    expect(before[0]?.title).toBe('Fixture')
    const events: [string,AnnouncementEvent][]=[['announcements',{event_type:'archived',announcement:row(1)}],['announcements',{event_type:'new',announcement:row(2)}],['announcements',{event_type:'new',announcement:row(2)}]]
    expect(reconcileAnnouncementSnapshot(before,events).map(a=>a.id)).toEqual([2])
    expect(applyAnnouncementEvent(before,'killlist',{event_type:'new',announcement:row(3)})).toBe(before)
    expect(applyAnnouncementEvent(before,'announcements',null)).toBe(before)
})
test('ticker freshness fades over ten minutes without affecting lifetime filtering', () => {
    expect(tickerFreshness({expires_at:new Date(now+600000).toISOString()},now)).toBe(1)
    expect(tickerFreshness({expires_at:new Date(now+300000).toISOString()},now)).toBeCloseTo(0.675)
    expect(tickerFreshness({expires_at:new Date(now).toISOString()},now)).toBe(0.35)
    expect(tickerFreshness({expires_at:'invalid'},now)).toBe(1)
})
