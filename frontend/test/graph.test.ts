import {describe, expect, test} from 'vitest'
import {formatCompact as original} from '../src/utils/formatters'
import {formatCompact as bundled} from '../src/browser/formatters'
import {renderGraph} from '../src/browser/graph-render'
import type {GraphMode} from '../src/browser/shared/graph-model'

describe('graph API rendering', () => {
  test('uses the original formatter at count and ISK boundaries', () => {
    for (const value of [0, 1, 999, 1000, 1499, 1e6, 1.234567e9, 1e12, NaN, -10]) expect(bundled(value)).toBe(original(value))
    const html=renderGraph('rivalries',{items:[{entity_a:{id:1,name:'A'},entity_b:{id:2,name:'B'},mutual_kills:1234,total_isk:1.234567e9}]})
    expect(html).toContain('1.2k');expect(html).toContain('1.23b')
  })
  test('escapes graph names and metadata in every data shape', () => {
    const unsafe=`"><img src=x onerror=alert(1)>`
    const entity={id:1,name:unsafe,corp_name:unsafe,alliance_name:unsafe}
    const cases: [GraphMode, unknown][]=[
      ['path_finder',{path:{hops:1,nodes:[{...entity,corporation:entity,alliance:entity},entity],edges:[{weight:1}]}}],
      ['entity_intel',{allies:[{...entity,shared_enemy_kills:unsafe,mutual_kills:unsafe}],enemies:[entity]}],
      ['hunting_grounds',{systems:[{...entity,active_characters:1}]}],
      ['hot_zones',{systems:[{...entity,alliances:1,characters:2}]}],
      ['census',{totals:{},corps:[{...entity,total:1}]}],
      ['spy_check',{suspects:[{...entity,connections:[entity]}]}],
      ['migration',{corp_name:unsafe,departed:[{...entity,current_corp:entity}],joined:[{...entity,previous_corp:entity}]}],
      ['coalitions',{coalitions:[{id:1,alliances:[entity]}]}],
      ['rivalries',{items:[{entity_a:entity,entity_b:entity}]}],
    ]
    for(const [mode,data] of cases){const html=renderGraph(mode,data);expect(html).not.toContain('<img');expect(html).toContain('&lt;img')}
  })
  test('keeps entity route kinds and all result rows', () => {
    const html=renderGraph('entity_intel',{allies:[{id:98598862,name:'Corp',shared_enemy_kills:1200,mutual_kills:20}],enemies:[{id:42,name:'Enemy',kills_given:200,kills_taken:100}]},'corporation')
    expect(html).toContain('href="/corporation/98598862"')
    expect(html).toContain('Shared enemy kills: 1200, Mutual kills: 20')
    expect(html).toContain('Kills given');expect(html).toContain('Kills taken')
    const zones=renderGraph('hot_zones',{systems:Array.from({length:30},(_,i)=>({id:30000000+i,name:'System '+i,alliances:i,characters:i*2}))})
    expect(zones.match(/<tr class=/g)?.length).toBe(31)
    expect(zones).toContain('/system/30000029')
    expect(renderGraph('path_finder',{path:null})).toContain('No connection found within 6 hops')
    expect(renderGraph('census',{corps:[]})).toContain('No military composition data found')
  })
})
