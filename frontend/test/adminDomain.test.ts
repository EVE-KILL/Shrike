import { expect,test } from 'vitest'
import { adminDomainCounts,filterAdminDomains,formatDomainAssetSize,domainAssetStatusColor } from '../src/utils/adminDomain'
test('admin domain filters combine status and case-insensitive hostname/site/owner search without modifying rows',()=>{
    const rows=[{id:1,subdomain:'one',site_name:'Fleet One',owner_name:'Pilot',custom_hostname:'fleet.example',active:true,pending_assets:2},{id:2,subdomain:'two',site_name:null,owner_name:null,active:false,pending_assets:0},{id:3,subdomain:'three',owner_name:'Pilot',active:false,pending_assets:1}]
    expect(adminDomainCounts(rows)).toEqual({all:3,pending:2,inactive:2})
    expect(filterAdminDomains(rows,' FLEET.EXAMPLE ','pending').map(d=>d.id)).toEqual([1])
    expect(filterAdminDomains(rows,'pilot','inactive').map(d=>d.id)).toEqual([3])
    expect(filterAdminDomains(rows,'Two','all').map(d=>d.id)).toEqual([2])
    expect(filterAdminDomains(rows,'missing','all')).toHaveLength(0)
    expect(rows).toHaveLength(3)
})
test('asset sizes and status labels retain the source boundaries',()=>{
    expect([0,1023,1024,1536,1048576].map(formatDomainAssetSize)).toEqual(['0 B','1023 B','1.0 KB','1.5 KB','1.0 MB'])
    expect(domainAssetStatusColor('approved')).toContain('green')
    expect(domainAssetStatusColor('rejected')).toContain('red')
    expect(domainAssetStatusColor('pending')).toContain('yellow')
})
