# SEO checkup

Run from the repository root with Node.js 22 or later:

```sh
node scripts/seo-checkup.mjs
node scripts/seo-checkup.mjs --end 2026-09-03 --out .data/seo/custom.json
node --test scripts/seo-checkup.test.mjs
```

The checker reads Google Search Console, Bing Webmaster and Yandex Webmaster,
then fetches the public robots file, homepage and up to 30 child sitemaps.
It never submits URLs, changes provider configuration or modifies the site.
Google OAuth token issuance and Search Console query/inspection POST requests
are used only to authenticate and read data.

Credentials live outside the repository in `~/Private/evekill/seo`, overridable
with `SEO_CREDENTIALS_DIR`:

| File | Contents |
| --- | --- |
| `google-service-account.json` | Google-issued service-account JSON with Search Console property access |
| `bing.json` | `{"api_key":"..."}` |
| `yandex.json` | `{"access_token":"..."}` |

Keep the directory owner-only (`700`) and files owner-readable/writable (`600`).
The checker doesn't need the Yandex client secret. Credentials are not copied
into reports or error messages. Default reports go into the ignored `.data/seo`
directory. Reports contain search queries and site diagnostics; keep them local.

The default reporting window is 28 days ending three days before today's UTC
date, compared with the preceding 28 days. Google dates follow Search Console's
Pacific-time reporting convention and use finalized web-search data. Bing rows
are selected by date, with the number of reported days shown; absent rows are
not silently counted as zero. Yandex popular-query data uses its own default
period and is not directly comparable with those 28-day totals.

Google query/page tables contain the top 1,000 returned rows, not all traffic.
Use the ungrouped totals for overall traffic: query tables omit anonymized
queries and page aggregation differs from property aggregation. Provider crawl
and index snapshots can predate the current deployment. A sitemap API's reported
`indexed` value is not used to infer the site's total Google index size.

The report stores each endpoint's success or failure separately so partial
outages remain visible. Exit status is 1 if a provider request fails or live
sitemap/homepage checks find a problem. A successful HTTP response alone does
not make a sitemap healthy: empty URL sets, invalid roots and embedded Nuxt
source errors are flagged. Empty categories can be legitimate and need review.

References: [Google Search Analytics](https://developers.google.com/webmaster-tools/v1/searchanalytics/query),
[Bing Webmaster API](https://learn.microsoft.com/en-us/bingwebmaster/),
[Yandex popular queries](https://yandex.com/dev/webmaster/doc/en/reference/host-search-queries-popular).
