<script setup lang="ts">
import type { HomepageWidgets, WidgetConfig } from '~/composables/useDomainConfig'
import type { EntityTracker, TrackerTargetType } from '~/composables/useTrackerNotifications'

interface PickedTarget {
    type: TrackerTargetType
    id: number
    name: string
    ticker: string | null
}

const props = defineProps<{
    widgets: HomepageWidgets
    trackers: EntityTracker[]
    trackerLimit: number
    firstVisit?: boolean
    saving?: boolean
}>()

const emit = defineEmits<{
    (event: 'save', value: HomepageWidgets): void
    (event: 'cancel'): void
}>()

const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value))
const draft = ref<HomepageWidgets>(clone(props.widgets))
watch(() => props.widgets, value => { draft.value = clone(value) }, { deep: true })

const creatingTracker = ref(false)
const trackerError = ref('')
const { refreshTrackers } = useTrackerNotifications()

type DashboardWidgetType = Exclude<WidgetConfig['type'], 'campaigns'>

const labels: Record<WidgetConfig['type'], string> = {
    mostValuable: 'Most Valuable',
    killList: 'Killlist',
    topCharacters: 'Active Pilots',
    topCorporations: 'Active Corporations',
    topAlliances: 'Active Alliances',
    topShips: 'Most Used Ships',
    topSystems: 'Top Systems',
    topRegions: 'Top Regions',
    entityInfo: 'Tracked Scope',
    textBlock: 'Text Block',
    campaigns: 'Campaigns',
}

const icons: Record<WidgetConfig['type'], string> = {
    mostValuable: 'lucide:gem',
    killList: 'lucide:swords',
    topCharacters: 'lucide:user',
    topCorporations: 'lucide:building-2',
    topAlliances: 'lucide:flag',
    topShips: 'lucide:rocket',
    topSystems: 'lucide:map-pin',
    topRegions: 'lucide:map',
    entityInfo: 'lucide:radar',
    textBlock: 'lucide:file-text',
    campaigns: 'lucide:flag',
}

const widgetTypes = (Object.keys(labels) as WidgetConfig['type'][])
    .filter((type): type is DashboardWidgetType => type !== 'campaigns')
const sections = ['top', 'left', 'right'] as const
type Section = typeof sections[number]

const existingTypes = computed(() => new Set(sections.flatMap(section =>
    draft.value[section].map(widget => widget.type))))
const availableTypes = computed(() => widgetTypes.filter(type => type === 'textBlock' || !existingTypes.value.has(type)))
const addType = ref<DashboardWidgetType | ''>('')
const addSection = ref<Section>('left')

function addWidget() {
    if (!addType.value) return
    const widget: WidgetConfig = { type: addType.value, enabled: true }
    if (addType.value === 'killList') widget.killlistType = 'latest'
    if (addType.value === 'textBlock') widget.content = ''
    draft.value[addSection.value].push(widget)
    addType.value = ''
}

function removeWidget(section: Section, index: number) {
    draft.value[section].splice(index, 1)
}

function moveWidget(section: Section, index: number, direction: -1 | 1) {
    const list = draft.value[section]
    const destination = index + direction
    if (destination < 0 || destination >= list.length) return
    const item = list[index]!
    list[index] = list[destination]!
    list[destination] = item
}

function moveToSection(source: Section, index: number, destination: Section) {
    if (source === destination) return
    const [widget] = draft.value[source].splice(index, 1)
    if (widget) draft.value[destination].push(widget)
}

const isPicked = (type: TrackerTargetType, id: number) =>
    props.trackers.some(tracker => tracker.target_type === type && tracker.target_id === id)

async function createTracker(target: PickedTarget) {
    if (creatingTracker.value || isPicked(target.type, target.id)) return
    creatingTracker.value = true
    trackerError.value = ''
    try {
        await apiFetch('/api/me/trackers', {
            method: 'POST',
            body: {
                target_type: target.type,
                target_id: target.id,
                notifications_enabled: false,
            },
        })
        await refreshTrackers()
    } catch (error: any) {
        trackerError.value = error?.data?.error || error?.message || 'Could not add tracker.'
    } finally {
        creatingTracker.value = false
    }
}

function save() {
    emit('save', clone(draft.value))
}
</script>

<template>
    <div class="space-y-5">
        <div class="glass-panel p-5">
            <div class="mb-4 flex items-start gap-3">
                <span class="flex h-8 w-8 items-center justify-center rounded-full bg-blue-500/10 text-sm font-bold text-blue-300">1</span>
                <div>
                    <h2 class="font-semibold text-white">Choose what this killboard tracks</h2>
                    <p class="mt-1 text-xs text-gray-500">Add entities or locations here. Alert settings remain separate and default to off.</p>
                </div>
            </div>
            <SearchPicker
                :types="['alliance', 'corporation', 'character', 'region', 'constellation', 'system']"
                placeholder="Add an entity or location..."
                :disabled="creatingTracker || trackers.length >= trackerLimit"
                :is-picked="isPicked"
                @select="createTracker"
            />
            <p v-if="trackerError" class="mt-2 text-xs text-red-300">{{ trackerError }}</p>
            <div v-if="trackers.length" class="mt-3 flex flex-wrap gap-2">
                <span
                    v-for="tracker in trackers"
                    :key="tracker.id"
                    class="inline-flex items-center gap-1.5 rounded-full border border-white/[0.08] bg-white/[0.03] px-2.5 py-1 text-xs"
                    :class="tracker.enabled ? 'text-gray-300' : 'text-gray-600'"
                >
                    <span class="capitalize text-gray-600">{{ tracker.target_type }}</span>
                    {{ tracker.target_name }}
                    <span v-if="!tracker.enabled" class="text-fine uppercase">paused</span>
                </span>
            </div>
            <div class="mt-3 flex items-center justify-between text-fine text-gray-600">
                <span>{{ trackers.length }}/{{ trackerLimit }} trackers</span>
                <NuxtLink to="/trackers" class="text-blue-400 hover:text-blue-300">Advanced tracker and alert settings →</NuxtLink>
            </div>
        </div>

        <div class="glass-panel p-5">
            <div class="mb-4 flex items-start gap-3">
                <span class="flex h-8 w-8 items-center justify-center rounded-full bg-blue-500/10 text-sm font-bold text-blue-300">2</span>
                <div>
                    <h2 class="font-semibold text-white">Arrange your dashboard</h2>
                    <p class="mt-1 text-xs text-gray-500">Top spans the page; left and right form the two-column area below it.</p>
                </div>
            </div>

            <div class="mb-4 flex flex-wrap items-end gap-3 rounded-lg border border-white/[0.06] bg-black/20 p-3">
                <label class="min-w-44 flex-1">
                    <span class="mb-1 block text-fine font-bold uppercase tracking-wider text-gray-600">Widget</span>
                    <select v-model="addType" class="w-full rounded-md border border-white/[0.08] bg-gray-950 px-3 py-2 text-xs text-gray-300">
                        <option value="">Choose a widget…</option>
                        <option v-for="type in availableTypes" :key="type" :value="type">{{ labels[type] }}</option>
                    </select>
                </label>
                <label>
                    <span class="mb-1 block text-fine font-bold uppercase tracking-wider text-gray-600">Place in</span>
                    <select v-model="addSection" class="rounded-md border border-white/[0.08] bg-gray-950 px-3 py-2 text-xs text-gray-300">
                        <option value="top">Top</option>
                        <option value="left">Left column</option>
                        <option value="right">Right column</option>
                    </select>
                </label>
                <button type="button" class="rounded-md bg-blue-500/15 px-3 py-2 text-xs font-semibold text-blue-300 hover:bg-blue-500/25 disabled:opacity-40" :disabled="!addType" @click="addWidget">
                    Add widget
                </button>
                <label>
                    <span class="mb-1 block text-fine font-bold uppercase tracking-wider text-gray-600">Columns</span>
                    <select v-model="draft.columnRatio" class="rounded-md border border-white/[0.08] bg-gray-950 px-3 py-2 text-xs text-gray-300">
                        <option value="250px_1fr">250px / flexible</option>
                        <option value="300px_1fr">300px / flexible</option>
                        <option value="1fr_1fr">50% / 50%</option>
                        <option value="1fr_2fr">33% / 66%</option>
                        <option value="1fr_3fr">25% / 75%</option>
                    </select>
                </label>
            </div>

            <div class="grid gap-4 lg:grid-cols-3">
                <section v-for="section in sections" :key="section" class="rounded-lg border border-white/[0.06] bg-black/20 p-3">
                    <h3 class="mb-2 text-fine font-bold uppercase tracking-[0.15em] text-blue-300">
                        {{ section === 'top' ? 'Top · full width' : `${section} column` }}
                    </h3>
                    <div v-if="draft[section].length === 0" class="rounded border border-dashed border-white/[0.08] px-3 py-6 text-center text-xs text-gray-700">
                        No widgets
                    </div>
                    <div v-else class="space-y-2">
                        <div v-for="(widget, index) in draft[section]" :key="`${widget.type}-${index}`" class="rounded-md border border-white/[0.06] bg-white/[0.025] p-2.5" :class="widget.enabled ? '' : 'opacity-50'">
                            <div class="flex items-center gap-2">
                                <Icon :name="icons[widget.type] || 'lucide:box'" class="text-gray-500" />
                                <span class="min-w-0 flex-1 truncate text-xs font-semibold text-gray-300">{{ labels[widget.type] || widget.type }}</span>
                                <button type="button" class="text-gray-600 hover:text-white disabled:opacity-20" :disabled="index === 0" aria-label="Move widget up" @click="moveWidget(section, index, -1)"><Icon name="lucide:chevron-up" /></button>
                                <button type="button" class="text-gray-600 hover:text-white disabled:opacity-20" :disabled="index === draft[section].length - 1" aria-label="Move widget down" @click="moveWidget(section, index, 1)"><Icon name="lucide:chevron-down" /></button>
                                <label class="flex cursor-pointer items-center" v-tooltip="widget.enabled ? 'Enabled' : 'Disabled'">
                                    <input v-model="widget.enabled" type="checkbox" class="rounded border-white/10 bg-black" />
                                </label>
                                <button type="button" class="text-gray-600 hover:text-red-400" aria-label="Remove widget" @click="removeWidget(section, index)"><Icon name="lucide:x" /></button>
                            </div>
                            <div class="mt-2 flex gap-2">
                                <select :value="section" class="min-w-0 flex-1 rounded border border-white/[0.06] bg-gray-950 px-2 py-1 text-fine text-gray-500" @change="moveToSection(section, index, ($event.target as HTMLSelectElement).value as Section)">
                                    <option value="top">Top</option>
                                    <option value="left">Left</option>
                                    <option value="right">Right</option>
                                </select>
                                <select v-if="widget.type === 'killList'" v-model="widget.killlistType" class="min-w-0 flex-1 rounded border border-white/[0.06] bg-gray-950 px-2 py-1 text-fine text-gray-500">
                                    <option value="latest">Latest</option>
                                    <option value="big">Big kills</option>
                                    <option value="solo">Solo</option>
                                    <option value="npc">NPC</option>
                                </select>
                            </div>
                            <textarea v-if="widget.type === 'textBlock'" v-model="widget.content" maxlength="2000" rows="3" placeholder="Write a note for your dashboard…" class="mt-2 w-full resize-y rounded border border-white/[0.06] bg-black/30 px-2 py-1.5 text-xs text-gray-300 outline-none focus:border-blue-400/30" />
                        </div>
                    </div>
                </section>
            </div>
        </div>

        <div class="flex justify-end gap-2">
            <button v-if="!firstVisit" type="button" class="rounded-md border border-white/10 px-4 py-2 text-sm text-gray-400 hover:text-white" @click="emit('cancel')">Cancel</button>
            <button type="button" class="rounded-md bg-blue-500 px-5 py-2 text-sm font-semibold text-white hover:bg-blue-400 disabled:opacity-50" :disabled="saving" @click="save">
                <Icon v-if="saving" name="lucide:loader-2" class="mr-1 animate-spin" />
                {{ firstVisit ? 'Create my killboard' : 'Save dashboard' }}
            </button>
        </div>
    </div>
</template>
