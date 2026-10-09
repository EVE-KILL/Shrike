import type { HomepageWidgets, WidgetConfig } from '../models/domainConfig'

// One widget catalog for Nuxt, the Go templates, and the browser controller.
export const trackerDashboardWidgets = [
    { type: 'mostValuable', title: 'Most Valuable', icon: 'gem' },
    { type: 'killList', title: 'Killlist', icon: 'swords' },
    { type: 'topCharacters', title: 'Active Pilots', icon: 'user', dataType: 'characters' },
    { type: 'topCorporations', title: 'Active Corporations', icon: 'building-2', dataType: 'corporations' },
    { type: 'topAlliances', title: 'Active Alliances', icon: 'flag', dataType: 'alliances' },
    { type: 'topShips', title: 'Most Used Ships', icon: 'rocket', dataType: 'ships' },
    { type: 'topSystems', title: 'Top Systems', icon: 'map-pin', dataType: 'systems' },
    { type: 'topRegions', title: 'Top Regions', icon: 'map', dataType: 'regions' },
    { type: 'entityInfo', title: 'Tracked Scope', icon: 'radar' },
    { type: 'textBlock', title: 'Text Block', icon: 'file-text' },
] as const
export const trackerDashboardSections = ['top', 'left', 'right'] as const
export type DashboardSection = typeof trackerDashboardSections[number]
export type DashboardWidgetType = typeof trackerDashboardWidgets[number]['type']
export const trackerDashboardLabels = Object.fromEntries(trackerDashboardWidgets.map(w => [w.type, w.title])) as Record<WidgetConfig['type'], string>
export const trackerDashboardIcons = Object.fromEntries(trackerDashboardWidgets.map(w => [w.type, 'lucide:' + w.icon])) as Record<WidgetConfig['type'], string>
export const trackerDashboardTopBoxes = Object.fromEntries(trackerDashboardWidgets.filter(w => 'dataType' in w).map(w => [w.type, w])) as Record<string, { title: string; dataType: string }>

export function availableDashboardWidgets(draft: HomepageWidgets): DashboardWidgetType[] {
    const existing = new Set(trackerDashboardSections.flatMap(section => draft[section].map(w => w.type)))
    return trackerDashboardWidgets.map(w => w.type).filter(type => type === 'textBlock' || !existing.has(type))
}
export function addDashboardWidget(draft: HomepageWidgets, type: DashboardWidgetType, section: DashboardSection): boolean {
    if (!availableDashboardWidgets(draft).includes(type)) return false
    const widget: WidgetConfig = { type, enabled: true }
    if (type === 'killList') widget.killlistType = 'latest'
    if (type === 'textBlock') widget.content = ''
    draft[section].push(widget)
    return true
}
export function moveDashboardWidget(draft: HomepageWidgets, section: DashboardSection, index: number, direction: -1 | 1) {
    const list = draft[section], destination = index + direction
    if (index < 0 || index >= list.length || destination < 0 || destination >= list.length) return
    const item = list[index]!
    list[index] = list[destination]!
    list[destination] = item
}
export function moveDashboardWidgetToSection(draft: HomepageWidgets, source: DashboardSection, index: number, destination: DashboardSection) {
    if (source === destination || index < 0 || index >= draft[source].length) return
    const [widget] = draft[source].splice(index, 1)
    if (widget) draft[destination].push(widget)
}
