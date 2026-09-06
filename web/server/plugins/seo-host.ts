import { isCanonicalSeoHost } from '../utils/seoHost'

export default defineNitroPlugin((nitroApp) => {
    nitroApp.hooks.hook('beforeResponse', (event) => {
        if (!isCanonicalSeoHost(getRequestHost(event))) {
            // Set after nuxt-robots' response handling, including cached pages,
            // so the header agrees with tenant HTML. The custom error handler
            // applies the same rule to responses sent directly by h3.
            setResponseHeader(event, 'x-robots-tag', 'noindex, nofollow')
        }
    })
})
