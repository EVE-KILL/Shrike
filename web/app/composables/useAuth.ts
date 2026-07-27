export interface AuthUser {
    characterId: number;
    characterName: string;
    characterOwnerHash: string;
    isAdmin: boolean;
    settings: Record<string, unknown>;
    corporationId: number | null;
    corporationName: string | null;
    allianceId: number | null;
    allianceName: string | null;
    /**
     * Highest comment id this user has acknowledged in their notifications bell.
     * Synced via POST /api/notifications/mark-read so unread state stays consistent
     * across devices. Mirrored from the server's eveAuth.ts AuthUser shape.
     */
    lastSeenNotificationId: number;
}

let _fullAuthFetched = false;

export function useAuth() {
    const user = useState<AuthUser | null>("auth-user", () => null);

    const isAuthenticated = computed(() => !!user.value);
    const isAdmin = computed(() => user.value?.isAdmin ?? false);

    // Auth hint from ek_auth cookie (set by inline script before hydration)
    const authHint = import.meta.client
        ? (window as any).__AUTH_HINT__ as { characterId: number; characterName: string } | undefined
        : undefined;

    if (import.meta.client && authHint && !_fullAuthFetched) {
        _fullAuthFetched = true;

        // Seed with hint immediately — this triggers after hydration
        // so ClientOnly will pick it up on its first client render. The
        // background /api/auth/me fetch below replaces this with the full
        // server-authoritative shape (incl. lastSeenNotificationId).
        if (!user.value) {
            user.value = {
                characterId: authHint.characterId,
                characterName: authHint.characterName,
                characterOwnerHash: "",
                isAdmin: false,
                settings: {},
                corporationId: null,
                corporationName: null,
                allianceId: null,
                allianceName: null,
                lastSeenNotificationId: 0,
            };
        }

        // Fetch full auth data in background
        apiFetch<{ user: AuthUser | null }>("/api/auth/me")
            .then((data) => {
                user.value = data.user ?? null;
            })
            .catch(() => {
                user.value = null;
            });
    }

    async function login(opts: { charKm?: boolean; corpKm?: boolean; delay?: number; redirect?: string } = {}) {
        const currentPath = opts.redirect || useRoute().fullPath;
        useAnalytics().track('auth.login_start', {
            charKm: opts.charKm !== false,
            corpKm: opts.corpKm !== false,
        });
        const data = await apiFetch<{ url: string }>("/api/auth/login", {
            params: {
                redirect: currentPath,
                charKm: opts.charKm === false ? "0" : "1",
                corpKm: opts.corpKm === false ? "0" : "1",
                delay: String(opts.delay ?? 0),
            },
        });
        navigateTo(data.url, { external: true });
    }

    async function logout() {
        await apiFetch("/api/auth/logout", { method: "POST" });
        user.value = null;
        _fullAuthFetched = false;
        navigateTo("/");
    }

    return {
        user: readonly(user),
        isAuthenticated,
        isAdmin,
        authHint: authHint ?? null,
        login,
        logout,
    };
}
