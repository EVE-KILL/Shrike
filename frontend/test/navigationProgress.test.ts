import { expect, test } from 'vitest'
import { createLoadingState, isPageDestination, observePageNavigations } from '../src/browser/loading-state'

function indicator() {
  let next = 0
  const timers = new Map<number, () => void>()
  const frames: { visible: boolean; sources: unknown[] }[] = []
  const state = createLoadingState({
    setTimer(callback: () => void, delay: number) { expect(delay).toBe(150); timers.set(++next, callback); return next },
    clearTimer(id: number) { timers.delete(id) },
    render(visible: boolean, sources: unknown[]) { frames.push({ visible, sources }) },
  })
  return { ...state, timers, frames, delay() {
    for (const [id, callback] of [...timers]) { timers.delete(id); callback() }
  } }
}

test('fast or cancelled work does not flash a loading indicator', () => {
  const loading = indicator(), finish = loading.start('link')
  expect(loading.frames).toEqual([])
  finish(); loading.delay()
  expect(loading.frames).toEqual([])
  expect(loading.timers.size).toBe(0)
})

test('slow work shows its source and clears on completion', () => {
  const loading = indicator(), finish = loading.start('row')
  loading.delay()
  expect(loading.frames.at(-1)).toEqual({ visible: true, sources: ['row'] })
  finish()
  expect(loading.frames.at(-1)).toEqual({ visible: false, sources: [] })
  const count = loading.frames.length
  finish()
  expect(loading.frames).toHaveLength(count)
})

test('full document navigation paints before browser timers can be suspended', () => {
  const loading = indicator(), finish = loading.start('row', { immediate: true })
  expect(loading.frames.at(-1)).toEqual({ visible: true, sources: ['row'] })
  expect(loading.timers.size).toBe(0)
  finish()
  expect(loading.frames.at(-1)).toEqual({ visible: false, sources: [] })
})

test('overlapping fragment requests keep the shared bar until the last one settles', () => {
  const loading = indicator(), first = loading.start('tab'), second = loading.start('filter')
  loading.delay(); first()
  expect(loading.frames.at(-1)).toEqual({ visible: true, sources: ['filter'] })
  second()
  expect(loading.frames.at(-1)).toEqual({ visible: false, sources: [] })
})

test('a superseding navigation can preserve the visible indicator', () => {
  const loading = indicator(), old = loading.start('first')
  loading.delay()
  const replacement = loading.start('second'); old()
  expect(loading.frames.every(frame => frame.visible)).toBe(true)
  replacement()
  expect(loading.frames.at(-1)?.visible).toBe(false)
})

test('clearing a document cancels timers and late completions are harmless', () => {
  const loading = indicator(), old = loading.start()
  loading.clear(); loading.delay(); old()
  expect(loading.frames).toEqual([])
  const current = loading.start('current'); loading.delay(); old()
  expect(loading.frames.at(-1)).toEqual({ visible: true, sources: ['current'] })
  current()
})

test('clearing visible work restores a ready document', () => {
  const loading = indicator(); loading.start('row'); loading.delay(); loading.clear()
  expect(loading.frames.at(-1)).toEqual({ visible: false, sources: [] })
})

test('page destinations omit in-document anchors and non-page protocols', () => {
  const current = 'https://eve-kill.com/character/42'
  for (const path of ['#main', '/character/42#top', 'javascript:void(0)', 'mailto:a@example.com', 'data:text/plain,hi'])
    expect(isPageDestination(path, current)).toBe(false)
  for (const path of ['/character/43', '/character/42/kills', '/character/42?limit=100', current, 'https://example.com'])
    expect(isPageDestination(path, current)).toBe(true)
  expect(isPageDestination('/character/42#top', current+'#old')).toBe(false)
})

function pageNavigation(overrides = {}) {
  const controller = new AbortController()
  const event = Object.assign(new Event('navigate', { cancelable: true }), {
    destination: { sameDocument: false, url: 'https://eve-kill.com/kill/42' },
    signal: controller.signal, hashChange: false, downloadRequest: null, sourceElement: null,
    ...overrides,
  })
  return { event, controller }
}

test('programmatic row navigation without a source element starts feedback synchronously', () => {
  const events = new EventTarget(), loading = indicator()
  let finish: (() => void) | undefined
  observePageNavigations(events, () => finish = loading.start('JS row', { immediate: true }), () => finish?.())
  events.dispatchEvent(pageNavigation().event)
  expect(loading.frames.at(-1)).toEqual({ visible: true, sources: ['JS row'] })
  events.dispatchEvent(new Event('navigatesuccess'))
  expect(loading.frames.at(-1)?.visible).toBe(false)
})

test('cancelled navigation releases only its own work while a replacement stays visible', () => {
  const events = new EventTarget(), loading = indicator()
  observePageNavigations(events, () => loading.start('row', { immediate: true }), () => {})
  const first = pageNavigation(), second = pageNavigation()
  events.dispatchEvent(first.event); events.dispatchEvent(second.event)
  first.controller.abort()
  expect(loading.frames.at(-1)?.visible).toBe(true)
  second.controller.abort()
  expect(loading.frames.at(-1)?.visible).toBe(false)
})

test('hashes, same-document history, downloads and cancelled native events do not start a page loader', () => {
  const events = new EventTarget(), loading = indicator()
  const stop = observePageNavigations(events, () => loading.start(null, { immediate: true }), () => loading.clear())
  for (const overrides of [{ hashChange: true }, { downloadRequest: 'report.csv' }, { downloadRequest: '' }, { destination: { sameDocument: true, url: 'https://eve-kill.com/kill/42' } }])
    events.dispatchEvent(pageNavigation(overrides).event)
  const prevented = pageNavigation(); prevented.event.preventDefault(); events.dispatchEvent(prevented.event)
  expect(loading.frames).toEqual([])
  stop(); events.dispatchEvent(pageNavigation().event)
  expect(loading.frames).toEqual([])
})

test('a failed native navigation clears its feedback', () => {
  const events = new EventTarget(), loading = indicator()
  observePageNavigations(events, () => loading.start(null, { immediate: true }), () => loading.clear())
  events.dispatchEvent(pageNavigation().event); events.dispatchEvent(new Event('navigateerror'))
  expect(loading.frames.at(-1)?.visible).toBe(false)
})
