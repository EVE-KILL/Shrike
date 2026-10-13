import {expect,test} from 'vitest'
import {trackerDashboardWidgets,availableDashboardWidgets,addDashboardWidget,moveDashboardWidget,moveDashboardWidgetToSection} from '../src/utils/trackerDashboard'
import type {HomepageWidgets} from '../src/models/domainConfig'

test('dashboard catalog and placement preserve every source widget and enabled state',()=>{
 const draft:HomepageWidgets={top:[{type:'mostValuable',enabled:true}],left:[{type:'topCharacters',enabled:false}],right:[{type:'killList',enabled:true,killlistType:'npc'}],columnRatio:'250px_1fr'}
 expect(trackerDashboardWidgets.map(w=>w.type)).toHaveLength(10)
 expect(availableDashboardWidgets(draft)).not.toContain('topCharacters')
 expect(addDashboardWidget(draft,'killList','left')).toBe(false)
 expect(addDashboardWidget(draft,'textBlock','top')).toBe(true)
 expect(addDashboardWidget(draft,'textBlock','top')).toBe(true)
 draft.top[1]!.content='<script>note</script>\nSecond line'
 moveDashboardWidget(draft,'top',0,-1)
 expect(draft.top[0]!.type).toBe('mostValuable')
 moveDashboardWidget(draft,'top',1,-1)
 expect(draft.top[0]!.content).toContain('Second line')
 moveDashboardWidgetToSection(draft,'top',0,'right')
 expect(draft.right[1]!.content).toContain('<script>')
 expect(draft.right[0]!.killlistType).toBe('npc')
 expect(draft.left[0]!.enabled).toBe(false)
 expect(draft.top).toHaveLength(2)
 moveDashboardWidgetToSection(draft,'left',10,'top')
 expect(draft.top).toHaveLength(2)
})

import {matchesStreamKillType} from '../shared/utils/killlistRow'
test('private feeds use the source relay thresholds and strict solo and NPC flags',()=>{
 expect(matchesStreamKillType({total_value:999999999},'big')).toBe(false)
 expect(matchesStreamKillType({total_value:1000000000},'big')).toBe(true)
 expect(matchesStreamKillType({total_value:5000000000},'5b')).toBe(true)
 expect(matchesStreamKillType({total_value:9999999999},'10b')).toBe(false)
 expect(matchesStreamKillType({is_npc:true},'solo')).toBe(false)
 expect(matchesStreamKillType({is_solo:true},'solo')).toBe(true)
 expect(matchesStreamKillType({is_solo:true},'npc')).toBe(false)
 expect(matchesStreamKillType({is_npc:true},'npc')).toBe(true)
})
