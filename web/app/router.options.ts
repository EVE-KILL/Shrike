import type { RouterConfig } from '@nuxt/schema'

// Entity routes that use the [[tab]] pattern
const tabRoutePattern = /^\/(character|corporation|alliance|system|region|constellation|item|faction|battle)\/(\d+)/

export default <RouterConfig>{
    scrollBehavior(to, from, savedPosition) {
        // Back/forward navigation: restore saved position
        if (savedPosition) return savedPosition

        // Hash navigation: scroll to the element
        if (to.hash) return { el: to.hash, behavior: 'smooth' }

        // Tab switch detection: same entity type + same ID, only tab segment differs
        const toMatch = to.path.match(tabRoutePattern)
        const fromMatch = from.path.match(tabRoutePattern)
        if (toMatch && fromMatch && toMatch[1] === fromMatch[1] && toMatch[2] === fromMatch[2]) {
            return false // preserve current scroll position
        }

        // Default: scroll to top
        return { top: 0 }
    },
}
