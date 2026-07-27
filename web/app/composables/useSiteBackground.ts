export function useSiteBackground() {
    const cookie = useCookie('siteBackground', { maxAge: 365 * 86400 })
    const { domainConfig } = useDomainConfig()

    // SSR always renders with the domain default background (or the base
    // bg2.webp on the main site) — never the user's cookie. This keeps the
    // rendered HTML identical for all visitors of a given host, which is
    // required for Cloudflare edge caching (one cached version per URL+host).
    //
    // The user's personal background is applied in two steps:
    //   1. Inline <head> script (nuxt.config.ts) reads the siteBackground
    //      cookie and sets backgroundImage on <html> BEFORE the body is
    //      parsed — no flash.
    //   2. Client hydration below syncs Vue state from the cookie so the
    //      background picker and viewer mode work correctly.
    const domainDefault = domainConfig.value?.backgrounds?.[0] || '/backgrounds/bg2.webp'

    const currentBackground = useState<string>(
        'currentBackground',
        () => domainDefault,
    )

    // Client: sync from cookie so Vue state matches the inline <head> script.
    if (import.meta.client) {
        const fromCookie = cookie.value as string | undefined
        if (fromCookie) {
            currentBackground.value = fromCookie
        }
    }

    const isRedditBackground = useState<boolean>('isRedditBg', () => false)
    const redditMeta = useState<{ title: string, source: string, subreddit: string } | null>('redditBgMeta', () => null)
    const viewerMode = useState<boolean>('bgViewerMode', () => false)

    // When background changes, update the cookie and the <html> inline style.
    // The inline <script> in <head> handles initial paint from cookie;
    // this watch handles subsequent changes during the session.
    //
    // CRITICAL: only register on the client. `useSiteBackground()` is called
    // from `layouts/default.vue` on every SSR render, so an unguarded
    // `watch()` here subscribes a fresh Vue effect to `currentBackground`'s
    // Dep per request. Vue 3 SSR does not aggressively dispose the
    // per-request effect scope, so those subscribers accumulate inside the
    // ref's Dep and retain the whole request's component graph. The side
    // effects (cookie + DOM style) are client-only anyway.
    if (import.meta.client) {
        watch(currentBackground, (v) => {
            cookie.value = v
            document.documentElement.style.setProperty('--site-bg', `url(${v})`)
        })
    }

    const setBackground = (path: string) => {
        currentBackground.value = path
        isRedditBackground.value = false
        redditMeta.value = null
    }

    const preloadImage = (url: string): Promise<void> => {
        return new Promise((resolve, reject) => {
            if (import.meta.server) { resolve(); return }
            const img = new Image()
            img.onload = () => resolve()
            img.onerror = () => reject()
            img.src = url
        })
    }

    const REDDIT_CACHE_KEY = 'ek_reddit_bg_cache'
    const REDDIT_CACHE_TTL = 30 * 60 * 1000 // 30 minutes

    const fetchRedditImages = async (): Promise<{ url: string, title: string, source: string, subreddit: string }[]> => {
        // Check localStorage cache first
        try {
            const cached = localStorage.getItem(REDDIT_CACHE_KEY)
            if (cached) {
                const { images, ts } = JSON.parse(cached)
                if (Date.now() - ts < REDDIT_CACHE_TTL && images?.length) return images
            }
        } catch {}

        // Server-side proxy of the r/eveporn Atom feed. Reddit 403s the
        // public .json endpoints (browser or not) since mid-2026, so direct
        // client fetching no longer works — see server/api/backgrounds/reddit.
        try {
            const response = await apiFetch<{ images: { url: string, title: string, source: string, subreddit: string }[] }>('/api/backgrounds/reddit')
            const images = response?.images || []

            // Cache in localStorage
            if (images.length) {
                try { localStorage.setItem(REDDIT_CACHE_KEY, JSON.stringify({ images, ts: Date.now() })) } catch {}
            }

            return images
        } catch {
            return []
        }
    }

    const setRedditBackground = async () => {
        if (import.meta.server) return
        const images = await fetchRedditImages()
        if (images.length) {
            const img = images[Math.floor(Math.random() * images.length)]
            if (!img) return
            await preloadImage(img.url).catch(() => {})
            currentBackground.value = img.url
            isRedditBackground.value = true
            redditMeta.value = { title: img.title, source: img.source, subreddit: img.subreddit }
        }
    }

    const toggleViewer = () => { viewerMode.value = !viewerMode.value }

    return {
        currentBackground: readonly(currentBackground),
        isRedditBackground: readonly(isRedditBackground),
        redditMeta: readonly(redditMeta),
        viewerMode,
        setBackground,
        setRedditBackground,
        toggleViewer,
    }
}
