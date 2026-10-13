import {describe,expect,test} from 'vitest'
import {defaultDomainWidgets,snapshotEditableDomain,buildDomainPatch,rebaseDomainEditor} from '../src/utils/domainEditor'

describe('shared domain edits',()=>{
  test('minimal patches preserve managed images and pair private campaign audiences',()=>{
    const domain={entities:[{type:'character',id:42}],theme:{bannerUrl:'/api/domains/asset/1/banner',logoUrl:'/api/domains/asset/1/logo',brandPrimary:'#112233'},navbar_links:[],widgets:defaultDomainWidgets(),site_name:'Fixture',site_description:'',campaign_policy:0,campaigns:[{campaign_id:'Fixture01',public_on_domain:false}]}
    const before=snapshotEditableDomain(domain)
    expect(buildDomainPatch(domain,before)).toEqual({})
    domain.theme.bannerUrl='/api/domains/asset/1/banner?v=2'
    domain.campaigns[0]!.public_on_domain=true
    expect(buildDomainPatch(domain,before)).toEqual({campaign_ids:['Fixture01'],campaign_public_ids:['Fixture01']})
    delete (domain.theme as Record<string,unknown>).brandPrimary
    expect(buildDomainPatch(domain,before).theme).toEqual({brandPrimary:null})
    expect(before.theme.brandPrimary).toBe('#112233')
  })
  test('new editors do not share widget state',()=>{
    const first=defaultDomainWidgets(),second=defaultDomainWidgets()
    first.left[0]!.enabled=false
    expect(second.left[0]!.enabled).toBe(true)
    expect(second.columnRatio).toBe('250px_1fr')
  })
  test('concurrent edits stay dirty while approved images follow the server',()=>{
    const submitted={entities:[],theme:{brandPrimary:'#112233'},navbar_links:[],widgets:defaultDomainWidgets(),site_name:'Submitted name',campaigns:[]}
    const editor=structuredClone(submitted)
    editor.site_name='Newer draft'
    editor.theme.brandPrimary='#445566'
    const baseline=rebaseDomainEditor(editor,{...submitted,theme:{...submitted.theme,bannerUrl:'/api/domains/asset/1/banner'}},submitted)
    expect(editor.site_name).toBe('Newer draft')
    expect(editor.theme.brandPrimary).toBe('#445566')
    expect((editor.theme as Record<string,unknown>).bannerUrl).toBe('/api/domains/asset/1/banner')
    expect(buildDomainPatch(editor,baseline)).toEqual({site_name:'Newer draft',theme:{brandPrimary:'#445566'}})
  })
})
