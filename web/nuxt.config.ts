import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath } from 'node:url'
import { KILL_LIST_TYPES } from './shared/utils/killListTypes'

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  // nuxt-security was removed (module + dependency) — its CSP/nonce/XSS
  // features are redundant given our own comment sanitization pipeline
  // (DOMPurify + markdown renderer) and Cloudflare's WAF layer. Also
  // confirmed innocent in the April 2026 SSR context retention leak
  // bisect (identical ~30 MB/min heap growth with and without it).
  modules: ['@nuxtjs/seo', '@nuxt/icon', '@nuxt/image', '@vueuse/nuxt', 'nuxt-umami'],

  umami: {
      host: 'https://analytics.karbowiak.dk',
      id: 'ee10fd70-9e08-4512-9e36-2f2c4c8145ff',
      autoTrack: true,
      ignoreLocalhost: true,
      logErrors: true,
  },

  // ── SEO: Site identity ──────────────────────────────────────────────
  site: {
      url: 'https://eve-kill.com',
      name: 'EVE-KILL',
      defaultLocale: 'en',
  },

  // ── SEO: Sitemap ────────────────────────────────────────────────────
  // Multi-sitemap mode: generates /sitemap_index.xml listing each per-entity
  // sub-sitemap. Lets crawlers parallelize and gives each entity type its own
  // 50k URL budget instead of competing for one shared file.
  sitemap: {
      cacheMaxAgeSeconds: 3600,
      sitemaps: {
          // Static pages — Nuxt auto-includes app routes (about, faq, wars, battles, stats, etc.)
          pages: {
              includeAppSources: true,
              // /kills/[type] is a dynamic route, so includeAppSources can't
              // enumerate it and every kill-list browse page was missing from
              // the sitemap despite being priority-0.8 content. The list is
              // shared with the page itself so the two can't drift.
              urls: KILL_LIST_TYPES.map(t => `/kills/${t}`),
              // /status renders `noindex, nofollow`; advertising it in the
              // sitemap asks crawlers to index a page that then refuses.
              exclude: ['/character/**', '/corporation/**', '/alliance/**', '/system/**', '/region/**', '/item/**', '/kill/**', '/war/**', '/battle/**', '/status'],
          },
          kills:        { sources: ['/api/__sitemap__/kills'] },
          characters:   { sources: ['/api/__sitemap__/characters'] },
          corporations: { sources: ['/api/__sitemap__/corporations'] },
          alliances:    { sources: ['/api/__sitemap__/alliances'] },
          systems:      { sources: ['/api/__sitemap__/systems'] },
          regions:      { sources: ['/api/__sitemap__/regions'] },
          ships:        { sources: ['/api/__sitemap__/ships'] },
          items:        { sources: ['/api/__sitemap__/items'] },
          wars:         { sources: ['/api/__sitemap__/wars'] },
          battles:      { sources: ['/api/__sitemap__/battles'] },
      },
  },

  // ── SEO: Robots ─────────────────────────────────────────────────────
  robots: {
      blockAiBots: false,
      groups: [
          {
              userAgent: '*',
              allow: ['/'],
              disallow: ['/api/'],
          },
      ],
  },

  // ── SEO: OG Image defaults ──────────────────────────────────────────
  ogImage: {
      enabled: false, // We use EVE image CDN portraits/logos — no server-side OG generation needed
  },

  // ── SEO: Schema.org ─────────────────────────────────────────────────
  schemaOrg: {
      identity: {
          type: 'Organization',
          name: 'EVE-KILL',
          url: 'https://eve-kill.com',
          logo: 'https://eve-kill.com/icon.svg',
          description: 'Community-driven killboard for EVE Online — real-time combat data, killmail tracking, and battle reports for New Eden.',
          email: 'contact@eve-kill.com',
          sameAs: [
              'https://discord.gg/Bz5gMHd',
              'https://github.com/EVE-KILL',
          ],
          contactPoint: {
              '@type': 'ContactPoint',
              'contactType': 'customer support',
              'url': 'https://discord.gg/Bz5gMHd',
          },
      },
  },

  // ── Image optimization (IPX) ─────────────────────────────────────────
  image: {
      format: ['webp', 'png'],
      quality: 80,
  },

  // ── Icons ───────────────────────────────────────────────────────────
  // Bundle all referenced Lucide icons into the client JS at build time.
  // Without this, every <Icon name="lucide:foo" /> rendered after client
  // hydration triggers a runtime fetch to /api/_nuxt_icon/lucide.json —
  // which under bot-firehose traffic became ~18% of all request volume
  // (7969 hits / hour on a single icon) and was the dominant allocation
  // source behind the Bun mimalloc arena retention. Scan=true has the
  // module regex-scan source files for static `collection:name` strings
  // and embed the matching SVGs directly into the client chunk.
  //
  // Dynamic `<Icon :name="ref" />` usages that the scanner can't resolve
  // statically still fall through to /api/_nuxt_icon/ — the routeRule
  // below gives those a long cache-control so CF absorbs the load.
  icon: {
      clientBundle: {
          scan: true,
          // The embedded set is ~508 KB raw (84 KB gzip) — right at the old
          // 512 ceiling. Headroom so newly referenced icons keep embedding
          // instead of silently falling back to runtime fetches.
          sizeLimitKb: 768,
      },
  },

  // ── SEO: Link checker (dev only) ────────────────────────────────────
  linkChecker: {
      enabled: false, // Only enable in CI / dev audits
  },

  // ── SEO: Defaults ───────────────────────────────────────────────────
  seo: {
      automaticDefaults: true,
      fallbackTitle: false,
  },

  css: ['~/assets/main.css'],

  nitro: {
      // Bun's stock Nitro entrypoint only binds TCP. Our tiny entrypoint keeps
      // the Bun preset and its export conditions, but binds NITRO_UNIX_SOCKET
      // so Caddy never needs a second public or loopback TCP listener.
      preset: 'bun',
      entry: './runtime/bun-unix.mjs',
      minify: true,
      compressPublicAssets: true,
      sourceMap: false,

      // Esbuild minification
      esbuild: {
          options: {
              minifySyntax: true,
              minifyWhitespace: true,
              minifyIdentifiers: true,
              treeShaking: true,
              target: 'esnext',
          },
      },

      // Renderer-owned response headers. Go owns /api, /auth, /images, and
      // /health before requests can reach Nitro.
      routeRules: {
          // Static assets — immutable, 1 year CDN cache
          '/_nuxt/**': { headers: { 'Cache-Control': 'public, max-age=31536000, immutable' } },
          '/backgrounds/**': { headers: { 'Cache-Control': 'public, max-age=31536000, immutable' } },
          '/fonts/**': { headers: { 'Cache-Control': 'public, max-age=31536000, immutable' } },
          '/remotes/**': { headers: { 'Cache-Control': 'public, max-age=604800' } },
          '/favicon.*': { headers: { 'Cache-Control': 'public, max-age=31536000, immutable' } },
          '/icon.*': { headers: { 'Cache-Control': 'public, max-age=31536000, immutable' } },
          '/manifest.json': { headers: { 'Cache-Control': 'public, max-age=86400' } },

          // @nuxt/icon runtime fallback endpoint — icons the build-time
          // scanner couldn't resolve (dynamic `:name="foo"`) land here.
          // Long immutable cache so Cloudflare absorbs the volume instead
          // of bouncing it back to the pods.
          '/api/_nuxt_icon/**': { headers: { 'Cache-Control': 'public, max-age=31536000, immutable' } },

          // Pages — CF edge caching via Cache-Control headers only.
          // Nitro's SWR (defineCachedEventHandler) is deliberately NOT used
          // for page routes: its in-memory pending-promise dedup Map retains
          // handler closures (H3Event → NuxtApp → full component tree) under
          // concurrent bot traffic, causing ~30 MB/min heap growth. With
          // s-maxage headers instead, Cloudflare caches the rendered HTML at
          // the edge (via the "respect origin s-maxage" cache rule) while the
          // pods stay stateless — each request is a fresh SSR render whose
          // context is fully GC'd after the response. API-level caching via
          // cachedApiHandler is unaffected and still handles the DB cost.
          '/': { sitemap: { changefreq: 'always', priority: 1.0 }, headers: { 'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=30' } },
          '/kills/**': { sitemap: { changefreq: 'always', priority: 0.8 }, headers: { 'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=30' } },
          '/kill/**': { sitemap: { changefreq: 'monthly', priority: 0.6 }, headers: { 'Cache-Control': 'public, s-maxage=300, stale-while-revalidate=300' } },
          '/character/**': { sitemap: { changefreq: 'daily', priority: 0.7 }, headers: { 'Cache-Control': 'public, s-maxage=120, stale-while-revalidate=120' } },
          '/corporation/**': { sitemap: { changefreq: 'daily', priority: 0.7 }, headers: { 'Cache-Control': 'public, s-maxage=120, stale-while-revalidate=120' } },
          '/alliance/**': { sitemap: { changefreq: 'daily', priority: 0.7 }, headers: { 'Cache-Control': 'public, s-maxage=120, stale-while-revalidate=120' } },
          '/wars': { sitemap: { changefreq: 'hourly', priority: 0.6 }, headers: { 'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=60' } },
          '/war/**': { sitemap: { changefreq: 'daily', priority: 0.5 }, headers: { 'Cache-Control': 'public, s-maxage=120, stale-while-revalidate=120' } },
          // Legacy old-EVE-KILL URL redirects (/wars/{id} → /war/{id},
          // /killmail/{id} → /kill/{id}) are handled by an nginx server-snippet
          // on the frontend ingress — see helm/eve-kill/templates/frontend/ingress.yaml.
          // Nitro routeRules can't express "match /wars/{id} but not /wars"
          // because `/wars/**` also matches `/wars` itself.
          '/battles': { sitemap: { changefreq: 'hourly', priority: 0.6 }, headers: { 'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=60' } },
          '/battle/**': { sitemap: { changefreq: 'weekly', priority: 0.5 }, headers: { 'Cache-Control': 'public, s-maxage=120, stale-while-revalidate=120' } },
          // Without an explicit s-maxage CF's respect-origin rule falls back
          // to the zone DEFAULT edge TTL (hours) — keep these short so new
          // campaigns and stat refreshes show up promptly on hard loads.
          '/campaigns': { sitemap: { changefreq: 'hourly', priority: 0.6 }, headers: { 'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=30' } },
          '/campaign/**': { sitemap: { changefreq: 'daily', priority: 0.5 }, headers: { 'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=60' } },
          // Auth-gated form — SSR output differs between logged-in/out, so it
          // must never be shared through the edge cache.
          '/campaigncreator': { sitemap: false, headers: { 'Cache-Control': 'no-cache, no-store, must-revalidate' } },
          '/stats': { sitemap: { changefreq: 'hourly', priority: 0.5 }, headers: { 'Cache-Control': 'public, s-maxage=60, stale-while-revalidate=60' } },
          '/about': { sitemap: { changefreq: 'monthly', priority: 0.3 } },
          '/faq': { sitemap: { changefreq: 'monthly', priority: 0.4 } },
          '/donate': { sitemap: { changefreq: 'monthly', priority: 0.2 } },
          '/search': { sitemap: { changefreq: 'daily', priority: 0.5 }, headers: { 'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=30' } },
          '/comments': { sitemap: { changefreq: 'hourly', priority: 0.4 }, headers: { 'Cache-Control': 'public, s-maxage=30, stale-while-revalidate=30' } },
          '/settings': { robots: false, sitemap: false, cache: false, headers: { 'Cache-Control': 'no-cache, no-store, must-revalidate' } },
          '/settings/**': { robots: false, sitemap: false, cache: false, headers: { 'Cache-Control': 'no-cache, no-store, must-revalidate' } },
          '/admin': { robots: false, sitemap: false, cache: false, headers: { 'Cache-Control': 'no-cache, no-store, must-revalidate' } },
          '/admin/**': { robots: false, sitemap: false, cache: false, headers: { 'Cache-Control': 'no-cache, no-store, must-revalidate' } },
      },
  },

  // Runtime env vars: use NUXT_ prefix to override at runtime.
  runtimeConfig: {
      // During SSR Bun reaches Shrike directly over this private Unix socket.
      // The browser never receives this value and continues using relative
      // same-origin URLs through Caddy.
      apiSocket: '',
      // Nitro reaches the same Go/Caddy process over loopback for SSR API
      // calls only when Nuxt is intentionally run outside `shrike serve`.
      apiOrigin: 'http://127.0.0.1:4000',
      public: {
          // Same-origin WebSocket and API namespaces served by Shrike.
          wsUrl: '/ws',
          publicApiUrl: '/api',
          // Public MCP base URL — used by the /mcp page to load the live
          // tool list. Override with NUXT_PUBLIC_PUBLIC_MCP_URL=http://localhost:4010
          // when running the mcp server locally.
          publicMcpUrl: 'https://mcp.eve-kill.com',
      },
  },

  // Experimental performance features
  experimental: {
      renderJsonPayloads: true,   // Faster SSR JSON payloads via native JSON.parse
      writeEarlyHints: false,     // No-op on nitro's bun preset (node-only feature)
      viewTransition: true,       // Smooth page transitions via View Transitions API
      asyncContext: true,
      ssrStreaming: false,
      entryImportMap: true,
  },

  // Vite build optimizations
  vite: {
      // Custom-domain killboards are routed by Host header, so testing one
      // locally means sending e.g. `Host: void.eve-kill.com` at the dev
      // server. Vite rejects unknown hosts by default. Dev-only — the
      // production server never goes through Vite.
      server: {
          allowedHosts: ['.eve-kill.com', '.localhost'],
      },
      optimizeDeps: {
        include: [
            'isbot',
        ]
      },
      plugins: [tailwindcss()],
      resolve: {
          alias: {
              // protobufjs/minimal pulls in @protobufjs/inquire, which uses
              // `eval("quire".replace(/^/,"re"))(name)` as a bundler-opaque
              // require() to optionally load Node-only modules. The eval
              // CALL itself trips CSP `script-src` even though the failure
              // is caught and returns null in protobufjs's try/catch — the
              // violation report still fires twice on every dogma load.
              // Replace the whole module with a stub that returns null;
              // safe because we never need Buffer or Long in the browser
              // (binary blobs come in as ArrayBuffers and we pass
              // `longs: Number` to every toObject call). See the shim
              // file for the longer explanation.
              //
              // The .cjs format is deliberate — protobufjs/minimal is
              // pre-bundled as CJS by optimizeDeps and does
              //   util.inquire = require("@protobufjs/inquire");
              //   util.inquire(name);
              // so the shim needs to expose a function via
              // `module.exports = fn`, not an ESM `export default` which
              // interop wraps into `{ default: fn }` and breaks with
              // `util.inquire is not a function`.
              '@protobufjs/inquire': fileURLToPath(new URL('./shims/protobufjs-inquire.cjs', import.meta.url)),
          },
      },
      build: {
          // Consolidate CSS into fewer files
          cssCodeSplit: false,
          // Never inline vendored dogma assets. Several files in
          // `packages/dogma/dist/upstream/` (e.g. categories.pb2 at <1KB)
          // sit under Vite's default 4KB asset-inline threshold and would
          // otherwise be emitted as `data:application/octet-stream;base64,...`
          // URIs — which the browser then refuses to fetch under our
          // `connect-src` policy. Force every dogma upstream file to be a
          // real hashed asset so the immutable /_nuxt/** cache rule handles
          // them and CSP stays narrow.
          assetsInlineLimit(filePath) {
              if (filePath.includes('packages/dogma/dist/upstream/')) return false
              return undefined
          },
          // Drop console in production
          minify: 'terser',
          terserOptions: {
              compress: {
                  drop_console: process.env.NODE_ENV === 'production',
                  drop_debugger: true,
                  pure_funcs: ['console.log', 'console.info', 'console.debug'],
              },
          },
      },
  },

  app: {
      head: {
          htmlAttrs: { lang: 'en', class: 'dark' },
          title: '',
          meta: [
              { name: 'viewport', content: 'width=device-width, initial-scale=1' },
              { name: 'color-scheme', content: 'dark' },
              { name: 'theme-color', content: '#000000' },
              { name: 'author', content: 'EVE-KILL' },
              { property: 'og:locale', content: 'en_US' },
          ],
          script: [
              // Runs before first paint — reads cookies for auth hint and background
              {
                  innerHTML: [
                      // Auth hint: parse ek_auth cookie, set class + CSS variable on <html>
                      `(function(){try{var m=document.cookie.match(/ek_auth=([^;]+)/);if(m){var p=decodeURIComponent(m[1]).split(':');var id=+p[0];var name=p.slice(1).join(':');window.__AUTH_HINT__={characterId:id,characterName:name};document.documentElement.classList.add('is-authed');document.documentElement.style.setProperty('--auth-portrait','url(/images/characters/'+id+'/portrait?size=64)')}}catch(e){}})()`,
                      // Background: override html background-image from cookie
                      `(function(){try{var m=document.cookie.match(/siteBackground=([^;]+)/);if(m){document.documentElement.style.setProperty('--site-bg','url('+decodeURIComponent(m[1])+')')}}catch(e){}})()`,
                      // Theme: apply CSS variable overrides from ek_theme cookie
                      `(function(){try{var m=document.cookie.match(/ek_theme=([^;]+)/);if(m){var t=JSON.parse(decodeURIComponent(m[1]));var map={brandPrimary:'--color-brand-primary',brandPrimaryHover:'--color-brand-primary-hover',brandSecondary:'--color-brand-secondary',brandAccent:'--color-brand-accent',bgPrimary:'--color-bg-primary',bgSecondary:'--color-bg-secondary',bgTertiary:'--color-bg-tertiary',bgHover:'--color-bg-hover',textPrimary:'--color-text-primary',textSecondary:'--color-text-secondary',textTertiary:'--color-text-tertiary',borderLight:'--color-border-light',borderMedium:'--color-border-medium',borderFocus:'--color-border-focus',surfaceAlpha:'--color-surface-alpha',surfaceHover:'--color-surface-hover',colorSuccess:'--color-success',colorWarning:'--color-warning',colorError:'--color-error',colorInfo:'--color-info',lossBg:'--color-loss-bg',lossHover:'--color-loss-hover',lossBorder:'--color-loss-border',colorHighsec:'--color-highsec',colorLowsec:'--color-lowsec',colorNullsec:'--color-nullsec',scrollbarThumb:'--scrollbar-thumb-color',iskColor:'--color-isk-value',npcColor:'--color-npc-text',selectionBg:'--color-selection-bg',selectionText:'--color-selection-text'};var r=document.documentElement;for(var k in t){if(map[k])r.style.setProperty(map[k],t[k])}}}catch(e){}})()`,
                  ].join(';'),
                  tagPosition: 'head',
              },
          ],
          link: [
              // DNS prefetch for external domains we load images from
              { rel: 'dns-prefetch', href: 'https://evemaps.dotlan.net' },
              { rel: 'dns-prefetch', href: 'https://zkillboard.com' },
              // Preload critical font to avoid render-blocking chain
              { rel: 'preload', href: '/fonts/Exo2-Variable.woff2', as: 'font', type: 'font/woff2', crossorigin: '' },
              // AI-discoverable guides. rel="llms" + rel="mcp" are hints for
              // web-browsing LLMs / MCP-aware agents that land on the site.
              { rel: 'llms' as any, type: 'text/plain', href: '/llms.txt' },
              { rel: 'mcp' as any, type: 'text/plain', href: '/mcp.txt' },
          ],
      },
  },

})
