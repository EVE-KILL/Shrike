import { expect,test } from 'vitest'
import {adminCampaignStates,adminCampaignStatus,formatCampaignRuntime,adminCampaignProcessed} from '../src/utils/adminCampaign'

test('campaign processing labels prioritize a pause independently of archival status',()=>{
    expect(adminCampaignStates.map(s=>s.value)).toEqual(['all','paused','failed','pending','active','archived'])
    expect(adminCampaignStatus({processing_paused:true,status:2}).label).toBe('Paused')
    expect([0,'1',2].map(status=>adminCampaignStatus({status}).label)).toEqual(['Pending','Active','Archived'])
    expect(adminCampaignStatus({status:0}).className).toContain('yellow')
})
test('runtime units retain the source thresholds and explicit zero processed counts',()=>{
    expect([null,0,999,1000,59999,60000,150000].map(formatCampaignRuntime)).toEqual(['—','—','999ms','1.0s','60.0s','1.0m','2.5m'])
    expect(adminCampaignProcessed({last_processing_killmails:0,totals:{killCount:10}})).toBe(0)
    expect(adminCampaignProcessed({last_processing_killmails:null,totals:{killCount:10}})).toBe(10)
    expect(adminCampaignProcessed({})).toBe(0)
})
