import {isRecord} from './json'
import {trackerDashboardWidgets,type DashboardSection,type DashboardWidgetType} from '../../utils/trackerDashboard'
import type {HomepageWidgets,WidgetConfig} from '../../models/domainConfig'

export type DashboardConfig={configured:boolean;widgets:HomepageWidgets}
export function isDashboardSection(value:unknown):value is DashboardSection{return value==='top'||value==='left'||value==='right'}
export function isDashboardWidgetType(value:unknown):value is DashboardWidgetType{return typeof value==='string'&&trackerDashboardWidgets.some(widget=>widget.type===value)}
function widget(value:unknown):WidgetConfig|null{
  if(!isRecord(value)||!isDashboardWidgetType(value.type)||typeof value.enabled!=='boolean')return null
  return {type:value.type,enabled:value.enabled,
    ...(typeof value.content==='string'?{content:value.content}:{}),
    ...(typeof value.killlistType==='string'?{killlistType:value.killlistType}:{}),
  }
}
export function readDashboardSeed(text:string|undefined):DashboardConfig|null{
  let value:unknown
  try{value=JSON.parse(text??'null')}catch{return null}
  if(!isRecord(value)||typeof value.configured!=='boolean'||!isRecord(value.widgets))return null
  const rows=(value:unknown)=>Array.isArray(value)?value.map(widget).filter((row):row is WidgetConfig=>row!==null):[]
  return {configured:value.configured,widgets:{top:rows(value.widgets.top),left:rows(value.widgets.left),right:rows(value.widgets.right),
    ...(typeof value.widgets.columnRatio==='string'?{columnRatio:value.widgets.columnRatio}:{}),
  }}
}
