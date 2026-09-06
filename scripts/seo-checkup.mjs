#!/usr/bin/env node
// Read-only provider queries. No sitemap submission or indexing mutations.
import { readFileSync, mkdirSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join, resolve, dirname } from 'node:path';
import { createSign } from 'node:crypto';
import { pathToFileURL } from 'node:url';

export function shiftDay(day, offset) {
  return new Date(Date.parse(day + 'T00:00:00Z') + offset * 86400000).toISOString().slice(0, 10);
}

export function bingDate(value) {
  const match = String(value).match(/\/Date\((-?\d+)(?:[+-]\d+)?\)\//);
  const date = new Date(match ? Number(match[1]) : value);
  return Number.isNaN(date.getTime()) ? null : date.toISOString().slice(0, 10);
}

export function trafficWindow(rows, start, end) {
  const selected = rows.filter(r => { const d = bingDate(r.Date); return d && d >= start && d <= end; });
  const clicks = selected.reduce((n, r) => n + (r.Clicks ?? 0), 0);
  const impressions = selected.reduce((n, r) => n + (r.Impressions ?? 0), 0);
  return { start, end, reportedDays: new Set(selected.map(r => bingDate(r.Date))).size,
    clicks: selected.length ? clicks : null, impressions: selected.length ? impressions : null,
    ctr: impressions ? clicks / impressions : null };
}

export function sitemapDetails(body) {
  const urls = [...body.matchAll(/<loc>\s*([^<]+?)\s*<\/loc>/g)].map(m => m[1]);
  return { count: urls.length, samples: urls.slice(0, 3),
    sourceError: body.includes('errors=true') || body.includes('error_messages='),
    validRoot: /<urlset[\s>]/.test(body) };
}

async function request(url, options = {}) {
  let response;
  try { response = await fetch(url, { ...options, signal: AbortSignal.timeout(30000) }); }
  catch { throw new Error('Network failure or 30-second timeout'); }
  // Never include request URLs (Bing key), tokens, or upstream error bodies.
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  try { return await response.json(); } catch { throw new Error('Invalid JSON response'); }
}

async function collect(tasks) {
  return Object.fromEntries(await Promise.all(Object.entries(tasks).map(async ([name, task]) => {
    try { return [name, { ok: true, data: await task() }]; }
    catch (error) { return [name, { ok: false, error: error.message }]; }
  })));
}

function credential(dir, name) {
  try { return JSON.parse(readFileSync(join(dir, name), 'utf8')); }
  catch { throw new Error(`Missing or invalid credential file: ${name}`); }
}

async function google(dir, period, previous) {
  const sa = credential(dir, 'google-service-account.json');
  const enc = value => Buffer.from(JSON.stringify(value)).toString('base64url');
  const now = Math.floor(Date.now() / 1000);
  const unsigned = enc({ alg: 'RS256', typ: 'JWT' }) + '.' + enc({ iss: sa.client_email,
    scope: 'https://www.googleapis.com/auth/webmasters.readonly',
    aud: 'https://oauth2.googleapis.com/token', iat: now, exp: now + 3600 });
  let signature;
  try { signature = createSign('RSA-SHA256').update(unsigned).sign(sa.private_key, 'base64url'); }
  catch { throw new Error('Invalid Google signing key'); }
  const token = await request('https://oauth2.googleapis.com/token', { method: 'POST',
    body: new URLSearchParams({ grant_type: 'urn:ietf:params:oauth:grant-type:jwt-bearer', assertion: unsigned + '.' + signature }) });
  const headers = { Authorization: `Bearer ${token.access_token}`, 'Content-Type': 'application/json' };
  const base = 'https://www.googleapis.com/webmasters/v3/sites/' + encodeURIComponent('sc-domain:eve-kill.com');
  const query = (range, dimensions = [], rowLimit = 100) => request(base + '/searchAnalytics/query', {
    method: 'POST', headers, body: JSON.stringify({ startDate: range.start, endDate: range.end,
      dimensions, rowLimit, type: 'web', dataState: 'final' }) });
  return collect({ current: () => query(period), previous: () => query(previous),
    daily: () => query({ start: previous.start, end: period.end }, ['date']),
    queries: () => query(period, ['query'], 1000), pages: () => query(period, ['page'], 1000),
    devices: () => query(period, ['device']), countries: () => query(period, ['country']),
    sitemaps: () => request(base + '/sitemaps', { headers }),
    homepageIndex: () => request('https://searchconsole.googleapis.com/v1/urlInspection/index:inspect', {
      method: 'POST', headers, body: JSON.stringify({ inspectionUrl: 'https://eve-kill.com/', siteUrl: 'sc-domain:eve-kill.com', languageCode: 'en-US' }) }) });
}

async function bing(dir, period, previous) {
  const key = credential(dir, 'bing.json').api_key;
  if (!key) throw new Error('Missing Bing api_key');
  const call = async method => {
    const params = new URLSearchParams({ apikey: key, siteUrl: 'https://eve-kill.com' });
    const result = await request(`https://ssl.bing.com/webmaster/api.svc/json/${method}?${params}`);
    if (result.ErrorCode || !('d' in result)) throw new Error('Bing API error or unexpected response');
    return result.d;
  };
  const result = await collect(Object.fromEntries(['GetRankAndTrafficStats', 'GetCrawlStats', 'GetQueryStats', 'GetCrawlIssues', 'GetFeeds']
    .map(method => [method, () => call(method)])));
  if (result.GetRankAndTrafficStats.ok) {
    const rows = result.GetRankAndTrafficStats.data;
    result.comparison = { current: trafficWindow(rows, period.start, period.end),
      previous: trafficWindow(rows, previous.start, previous.end) };
  }
  return result;
}

async function yandex(dir) {
  const token = credential(dir, 'yandex.json').access_token;
  if (!token) throw new Error('Missing Yandex access_token');
  const headers = { Authorization: `OAuth ${token}` };
  const user = await request('https://api.webmaster.yandex.net/v4/user/', { headers });
  const base = `https://api.webmaster.yandex.net/v4/user/${user.user_id}/hosts/${encodeURIComponent('https:eve-kill.com:443')}`;
  return collect({ summary: () => request(base + '/summary', { headers }),
    diagnostics: () => request(base + '/diagnostics', { headers }),
    queries: () => request(base + '/search-queries/popular?order_by=TOTAL_SHOWS&query_indicator=TOTAL_SHOWS&query_indicator=TOTAL_CLICKS&limit=100', { headers }),
    sitemaps: () => request(base + '/sitemaps/', { headers }) });
}

async function site() {
  const get = async path => {
    const r = await fetch('https://eve-kill.com' + path, { signal: AbortSignal.timeout(30000) });
    return { status: r.status, url: r.url, body: await r.text() };
  };
  const robots = await get('/robots.txt');
  const sitemap = await get('/sitemap_index.xml');
  const locations = [...sitemap.body.matchAll(/<loc>([^<]+)<\/loc>/g)].map(m => m[1]);
  const children = [];
  // Sequential and same-origin only: bounded load on the public site.
  for (const location of locations.slice(0, 30)) {
    const url = new URL(location);
    if (url.origin !== 'https://eve-kill.com') continue;
    try {
      const child = await get(url.pathname + url.search);
      children.push({ url: location, status: child.status, ...sitemapDetails(child.body) });
    } catch { children.push({ url: location, error: 'Request failed' }); }
  }
  const issues = [];
  if (robots.status !== 200) issues.push(`robots.txt HTTP ${robots.status}`);
  if (sitemap.status !== 200 || !locations.length) issues.push('Sitemap index unavailable or empty');
  for (const child of children) {
    if (child.error || child.status !== 200 || !child.validRoot || child.sourceError || !child.count)
      issues.push(`Unhealthy sitemap: ${child.url} (URLs: ${child.count ?? 'unknown'}, source error: ${child.sourceError ?? 'unknown'})`);
  }
  const home = await get('/');
  const homepage = { status: home.status, title: home.body.match(/<title>(.*?)<\/title>/s)?.[1] ?? null,
    metadata: [...home.body.matchAll(/<(?:meta|link)\b[^>]*(?:canonical|name="robots"|name="description")[^>]*>/g)].map(m => m[0]) };
  if (home.status !== 200) issues.push(`Homepage HTTP ${home.status}`);
  return { robots, homepage, sitemap: { status: sitemap.status, locations, children }, issues };
}

function summary(report) {
  const lines = [`SEO check: ${report.generatedAt}`, `Google/Bing comparison: ${report.period.start} to ${report.period.end}; previous ${report.previous.start} to ${report.previous.end}`];
  for (const [name, result] of Object.entries(report.providers)) {
    lines.push(`\n${name}: ${result.ok ? 'queried' : result.error}`);
    if (!result.ok) continue;
    for (const [section, value] of Object.entries(result.data)) {
      if (value?.ok === false) lines.push(`  ${section}: ERROR ${value.error}`);
    }
  }
  const google = report.providers.google.data;
  for (const key of ['current', 'previous']) {
    if (google?.[key]?.ok) lines.push(`Google ${key}: ${JSON.stringify(google[key].data.rows?.[0] ?? { note: 'No rows returned' })}`);
  }
  const comparison = report.providers.bing.data?.comparison;
  if (comparison) lines.push(`Bing: ${JSON.stringify(comparison)}`);
  const yandex = report.providers.yandex.data?.summary;
  if (yandex?.ok) lines.push(`Yandex: ${JSON.stringify(yandex.data)}`);
  const diagnostics = report.providers.yandex.data?.diagnostics;
  if (diagnostics?.ok) for (const [name, problem] of Object.entries(diagnostics.data.problems ?? {})) {
    if (problem.state === 'PRESENT') lines.push(`Yandex diagnostic: ${name} (${problem.severity})`);
  }
  const sitemaps = google?.sitemaps;
  if (sitemaps?.ok) for (const sitemap of sitemaps.data.sitemap ?? []) {
    lines.push(`Google sitemap: ${sitemap.path} errors=${sitemap.errors} warnings=${sitemap.warnings} last read=${sitemap.lastDownloaded}`);
  }
  const site = report.providers.site.data;
  if (site) {
    lines.push(`Live homepage title: ${site.homepage.title}`);
    lines.push(`Live sitemaps: ${site.sitemap.children.length}, URLs: ${site.sitemap.children.reduce((n, s) => n + (s.count ?? 0), 0)}`);
    for (const issue of site.issues) lines.push(`SITE ISSUE: ${issue}`);
  }
  return lines.join('\n');
}

export async function main(args = process.argv.slice(2)) {
  if (args.includes('--help')) {
    console.log('Usage: node scripts/seo-checkup.mjs [--end YYYY-MM-DD] [--out report.json]\nCredentials: SEO_CREDENTIALS_DIR or ~/Private/evekill/seo. Queries only; no search engine mutations.');
    return;
  }
  const options = {};
  for (let i = 0; i < args.length; i += 2) {
    if (!['--end', '--out'].includes(args[i]) || !args[i + 1]) throw new Error('Expected --end DATE or --out PATH');
    options[args[i]] = args[i + 1];
  }
  const end = options['--end'] ?? shiftDay(new Date().toISOString().slice(0, 10), -3);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(end) || !Number.isFinite(Date.parse(end)) || shiftDay(end, 0) !== end) throw new Error('Invalid end date');
  const period = { start: shiftDay(end, -27), end };
  const previous = { start: shiftDay(end, -55), end: shiftDay(end, -28) };
  const dir = process.env.SEO_CREDENTIALS_DIR ?? join(homedir(), 'Private/evekill/seo');
  const report = { generatedAt: new Date().toISOString(), period, previous,
    notes: ['Google uses finalized web-search data; query/page tables are top 1000, not exhaustive.',
      'Bing periods are filtered by actual dates; missing days are not assumed to be zero.',
      'Yandex popular queries use the provider default period; do not compare directly with the 28-day totals.',
      'Crawl diagnostics and URL inspection reflect provider snapshots, not necessarily the current deployment.'],
    providers: await collect({ google: () => google(dir, period, previous), bing: () => bing(dir, period, previous),
      yandex: () => yandex(dir), site }) };
  const output = resolve(options['--out'] ?? join('.data', 'seo', `${end}.json`));
  mkdirSync(dirname(output), { recursive: true });
  writeFileSync(output, JSON.stringify(report, null, 2) + '\n', { mode: 0o600 });
  console.log(summary(report));
  console.log(`\nReport: ${output}`);
  const failed = Object.values(report.providers).some(p => !p.ok || Object.values(p.data).some(v => v?.ok === false));
  if (failed || report.providers.site.data?.issues.length) process.exitCode = 1;
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch(() => { console.error('SEO check failed; check arguments and local configuration. No credentials were logged.'); process.exitCode = 1; });
}
