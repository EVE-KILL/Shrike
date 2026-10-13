import type {FitMetric} from '../src/browser/shared/fit-filters';
import {expect,test} from 'vitest';
import {parseStatisticValue,sortForMetric,stateFromURL,searchParams,statisticLabel,setStatistic,statisticOptions,npcProfiles} from '../src/browser/fit-search-state';

test('fitting statistics shorthand and every metric preserve the search API contract',()=>{
  for(const [input,expected] of [['50k',50000],['1.5M',1500000],[' 2,000 ',2000],['.5',.5],['1b',1000000000],['-1',null],['1t',null],['',null]]) expect(parseStatisticValue(String(input))).toBe(expected);
  const state=stateFromURL(new URLSearchParams('ship=587&ship_name=Rifter'));
  for(const [metric] of statisticOptions){setStatistic(state,metric,0,50,metric==='npc_ehp'?'guristas':undefined);const q=searchParams(state);expect(q.get(`min_${metric}`)).toBe('0');expect(q.get(`max_${metric}`)).toBe('50');}
  for(const metric of ['shield_repair','armor_repair','hull_repair','passive_shield'] as FitMetric[])expect(sortForMetric(metric)).toBe('repair');
  expect(sortForMetric('align')).toBe('align');expect(statisticLabel({metric:'npc_ehp',min:50000,max:null,npcProfile:'guristas'})).toBe('EHP against NPC faction · Guristas Pirates ≥ 50,000');
  for(const [profile] of npcProfiles){setStatistic(state,'npc_ehp',null,null,profile);expect(searchParams(state).get('npc_profile')).toBe(profile);}
});

test('URL state keeps role/type predicates, zero counts, sorting and capacitor stability without pagination',()=>{
  const query=new URLSearchParams({ship:'29990',ship_name:'Loki',sort:'dps',cap_stable:'true',filters:JSON.stringify([{role_id:'autocannon',op:'>=',count:2},{type_id:2048,type_name:'Damage Control II',op:'=',count:0}]),min_dps:'50',max_dps:'100'});
  const state=stateFromURL(query),api=searchParams(state,{sort:true});
  expect(state.filters).toEqual([{kind:'role',role_id:'autocannon',op:'>=',count:2},{kind:'type',type_id:2048,type_name:'Damage Control II',op:'=',count:0}]);
  expect(api.get('sort')).toBe('dps');expect(api.get('ship_name')).toBeNull();expect(api.get('cap_stable')).toBe('true');expect(api.get('offset')).toBeNull();
  expect(stateFromURL(searchParams(state,{name:true}))).toEqual(state);
  expect(stateFromURL(new URLSearchParams({ship:'-1',filters:'not json',sort:'bad',min_ehp:'NaN'}))).toEqual({ship:null,filters:[],statistics:[],sort:'uses',capStable:false});
});

test('URL requirements reject malformed operators/counts/IDs and cap accepted requirements at eight',()=>{
 const bad = [{role_id:'gun',op:'!=',count:1},{role_id:'gun',op:'=',count:17},{role_id:'gun',op:'=',count:.5},{type_id:-1,op:'=',count:0},{type_id:'587',op:'=',count:0},null]
 const valid = Array.from({length:12},(_,index)=>({role_id:'role-'+index,op:'>=',count:0}))
 const state = stateFromURL(new URLSearchParams({filters:JSON.stringify([...bad,...valid])}))
 expect(state.filters).toHaveLength(8);expect(state.filters[0]).toEqual({kind:'role',role_id:'role-0',op:'>=',count:0});expect(state.filters.at(-1)).toEqual({kind:'role',role_id:'role-7',op:'>=',count:0})
})
test('invalid scalar statistics are discarded while zero and a named NPC profile survive URL round trip',()=>{
 const state = stateFromURL(new URLSearchParams('ship=Infinity&min_dps=-1&max_ehp=Infinity&min_alpha=0&min_speed=%20&npc_profile=guristas&cap_stable=true'))
 expect(state.ship).toBeNull();expect(state.statistics).toEqual([{metric:'alpha',min:0,max:null},{metric:'npc_ehp',min:null,max:null,npcProfile:'guristas'}]);expect(stateFromURL(searchParams(state,{name:true}))).toEqual(state)
})
