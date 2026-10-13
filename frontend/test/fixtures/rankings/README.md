Captured through the normal Go handlers on 2026-10-08 using the existing
testRankingsReader. The full page was reduced to its data-rankings-page root;
character/corporation/hourly result fragments retain exact Go markup and markers.
The temporary capture test was removed. DOM tests cover retained SSR content,
filters/history/retry, stale bodies and hourly polling cleanup/visibility. Live
desktop/phone validation remains pending; no production data was changed.
