import {expect,test} from 'vitest'
import {preferredDefaultTab} from '../src/browser/default-tab'

test('saved default tabs affect only clean base URLs and supported routes',()=>{
 const cookie='ek_prefs='+encodeURIComponent(JSON.stringify({defaultTabs:{character:'kills',system:'kills',item:'market'}}))
 expect(preferredDefaultTab('character','/character/42','https://eve-kill.com/character/42',['/character/42/kills'],cookie)?.pathname).toBe('/character/42/kills')
 expect(preferredDefaultTab('character','/character/42','https://eve-kill.com/character/42/losses',['/character/42/kills'],cookie)).toBeNull()
 expect(preferredDefaultTab('system','/system/123','https://eve-kill.com/system/123',['/system/123/kills'],cookie)?.pathname).toBe('/system/123/kills')
 expect(preferredDefaultTab('group','/group/25','https://eve-kill.com/group/25',['/group/25/kills'],cookie)).toBeNull()
 expect(preferredDefaultTab('item','/item/34','https://eve-kill.com/item/34',['/item/34/market'],cookie)?.pathname).toBe('/item/34/market')
 expect(preferredDefaultTab('character','/character/42','https://eve-kill.com/character/42',['/character/42/kills'],'ek_prefs=%broken')).toBeNull()
})
