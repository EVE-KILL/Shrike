import { expect, test } from 'vitest'
import { announcementForm, announcementPayload, annStatusLabel, type AdminAnnouncement } from '../src/utils/adminAnnouncement'

const now = new Date('2026-10-04T12:34:56Z')
test('announcement schedules preserve EVE wall-clock values and ticker expiry is relative to submission', () => {
    const form = announcementForm(undefined, now)
    expect(form.starts_at).toBe('2026-10-04T12:34')
    expect(form.expires_at).toBe('2026-10-11T12:34')
    expect(announcementPayload(form, now)).toMatchObject({ starts_at: '2026-10-04T12:34:00.000Z', expires_at: '2026-10-11T12:34:00.000Z', icon: null, link_url: null })
    form.tier = 1
    form.duration = 30
    expect(announcementPayload(form, now)).toMatchObject({ starts_at: now.toISOString(), expires_at: '2026-10-04T13:04:56.000Z' })
    const edited = announcementForm({ tier: 1, title: 'Existing', expires_at: '2026-10-04T12:51:00Z', starts_at: now.toISOString() } as AdminAnnouncement, now)
    expect(edited.duration).toBe(30)
    form.tier = 2
    form.expires_at = ''
    expect(() => announcementPayload(form, now)).toThrow()
})
test('announcement status prioritizes archival and distinguishes scheduled, active and expired', () => {
    const row = { archived_at: null, starts_at: '2026-10-04T12:00:00Z', expires_at: '2026-10-04T13:00:00Z' }
    expect(annStatusLabel(row, now.getTime()).label).toBe('Active')
    expect(annStatusLabel(row, new Date('2026-10-04T11:00:00Z').getTime()).label).toBe('Scheduled')
    expect(annStatusLabel(row, new Date('2026-10-04T14:00:00Z').getTime()).label).toBe('Expired')
    expect(annStatusLabel({ ...row, archived_at: now.toISOString() }, 0).label).toBe('Archived')
})
