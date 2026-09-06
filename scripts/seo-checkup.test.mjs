import test from 'node:test';
import assert from 'node:assert/strict';
import { shiftDay, bingDate, trafficWindow, sitemapDetails } from './seo-checkup.mjs';

test('28-day comparison windows are inclusive and adjacent across leap day', () => {
  const end = '2024-03-03';
  assert.equal(shiftDay(end, -27), '2024-02-05');
  assert.equal(shiftDay(end, -28), '2024-02-04');
  assert.equal(shiftDay(end, -55), '2024-01-08');
});

test('Bing uses dates, not row order, and distinguishes absent data from zero traffic', () => {
  const ms = Date.parse('2026-09-03T07:00:00Z');
  assert.equal(bingDate(`/Date(${ms}-0700)/`), '2026-09-03');
  const rows = [{ Date: '2026-09-03', Clicks: 3, Impressions: 20 },
    { Date: '2026-08-01', Clicks: 99, Impressions: 100 },
    { Date: '2026-09-01', Clicks: 0, Impressions: 10 }];
  assert.deepEqual(trafficWindow(rows, '2026-09-01', '2026-09-03'), {
    start: '2026-09-01', end: '2026-09-03', reportedDays: 2, clicks: 3, impressions: 30, ctr: 0.1 });
  assert.equal(trafficWindow([], '2026-09-01', '2026-09-03').clicks, null);
  assert.equal(trafficWindow([rows[2]], '2026-09-01', '2026-09-03').clicks, 0);
});

test('HTTP-success sitemap can contain a source failure or an HTML error page', () => {
  assert.equal(sitemapDetails('<?xml-stylesheet href="style.xsl?errors=true"?><urlset></urlset>').sourceError, true);
  assert.equal(sitemapDetails('<html>Not found</html>').validRoot, false);
  assert.deepEqual(sitemapDetails('<urlset><url><loc> https://eve-kill.com/item/587 </loc></url></urlset>'), {
    count: 1, samples: ['https://eve-kill.com/item/587'], sourceError: false, validRoot: true });
});
