// Fires `auth.login_complete` once after the EVE SSO callback redirects
// back to the app. server/routes/auth/callback.ts appends `?ek_login=ok`
// to the destination; we strip it so it never lingers in shared URLs.

export default defineNuxtPlugin(() => {
    if (import.meta.server) return

    const route = useRoute()
    const router = useRouter()
    const { track } = useAnalytics()

    watch(() => route.query.ek_login, (val) => {
        if (val !== 'ok') return
        track('auth.login_complete')
        const { ek_login, ...rest } = route.query
        router.replace({ query: rest })
    }, { immediate: true })
})
