<script setup lang="ts">
defineOptions({ name: "SettingsSessions" });

interface UserSession {
	id: number;
	current: boolean;
	ipAddress: string | null;
	countryCode: string | null;
	browser: string;
	operatingSystem: string;
	device: string;
	createdAt: string;
	lastSeenAt: string;
	expiresAt: string;
}

const { logout } = useAuth();
const { data, pending, error, refresh } = await useApiFetch<{
	sessions: UserSession[];
}>("/api/user/sessions");
const revokingId = ref<number | null>(null);
const revokingOthers = ref(false);
const confirmOthers = ref(false);
const actionError = ref<string | null>(null);

const sessions = computed(() => data.value?.sessions ?? []);
const otherSessionCount = computed(
	() => sessions.value.filter((session) => !session.current).length,
);

const regionNames =
	typeof Intl.DisplayNames !== "undefined"
		? new Intl.DisplayNames(["en"], { type: "region" })
		: null;

const locationLabel = (session: UserSession) => {
	const country = session.countryCode
		? (regionNames?.of(session.countryCode) ?? session.countryCode)
		: null;
	return (
		[country, session.ipAddress].filter(Boolean).join(" · ") ||
		"Location unavailable"
	);
};

const timeAgo = (iso: string) => {
	const seconds = Math.round((new Date(iso).getTime() - Date.now()) / 1000);
	const formatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
	if (Math.abs(seconds) < 60) return formatter.format(seconds, "second");
	const minutes = Math.round(seconds / 60);
	if (Math.abs(minutes) < 60) return formatter.format(minutes, "minute");
	const hours = Math.round(minutes / 60);
	if (Math.abs(hours) < 24) return formatter.format(hours, "hour");
	const days = Math.round(hours / 24);
	if (Math.abs(days) < 30) return formatter.format(days, "day");
	return formatDateTime(iso);
};

const revokeSession = async (session: UserSession) => {
	actionError.value = null;
	revokingId.value = session.id;
	try {
		if (session.current) {
			await logout();
			return;
		}
		await apiFetch(`/api/user/sessions/${session.id}/revoke`, { method: "POST" });
		await refresh();
	} catch (e: any) {
		actionError.value = extractFetchError(e, "Could not revoke that session");
	} finally {
		revokingId.value = null;
	}
};

let confirmTimer: ReturnType<typeof setTimeout> | null = null;
const revokeOtherSessions = async () => {
	if (!confirmOthers.value) {
		confirmOthers.value = true;
		if (confirmTimer) clearTimeout(confirmTimer);
		confirmTimer = setTimeout(() => {
			confirmOthers.value = false;
		}, 5000);
		return;
	}

	actionError.value = null;
	revokingOthers.value = true;
	try {
		await apiFetch("/api/user/sessions/revoke-others", { method: "POST" });
		confirmOthers.value = false;
		await refresh();
	} catch (e: any) {
		actionError.value = extractFetchError(e, "Could not revoke the other sessions");
	} finally {
		revokingOthers.value = false;
	}
};

onUnmounted(() => {
	if (confirmTimer) clearTimeout(confirmTimer);
});
</script>

<template>
    <div class="space-y-5">
        <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
            <div>
                <h2 class="text-fine font-bold uppercase tracking-[0.15em] text-gray-500">Logged-in sessions</h2>
                <p class="text-xs text-gray-500 mt-1">Each browser stays signed in independently. Revoke anything you do not recognize.</p>
            </div>
            <button
                v-if="otherSessionCount > 0"
                class="px-3 py-2 rounded-lg border text-xs font-medium transition-colors cursor-pointer disabled:opacity-50"
                :class="confirmOthers
                    ? 'border-red-500/40 bg-red-500/10 text-red-300'
                    : 'border-white/[0.08] bg-white/[0.03] text-gray-400 hover:text-red-300 hover:border-red-500/30'"
                :disabled="revokingOthers"
                @click="revokeOtherSessions"
            >
                <Icon :name="revokingOthers ? 'lucide:loader-2' : 'lucide:log-out'" :class="['mr-1.5', revokingOthers ? 'animate-spin' : '']" />
                {{ revokingOthers ? 'Revoking…' : confirmOthers ? `Confirm logout of ${otherSessionCount}` : 'Log out everywhere else' }}
            </button>
        </div>

        <div v-if="actionError" class="rounded-lg border border-red-500/20 bg-red-500/[0.06] px-3 py-2 text-xs text-red-300">
            {{ actionError }}
        </div>

        <div v-if="pending" class="glass-panel p-8 text-center text-sm text-gray-500">
            <Icon name="lucide:loader-2" class="animate-spin mr-2" /> Loading sessions…
        </div>
        <div v-else-if="error" class="glass-panel p-5 text-sm text-red-300">
            Could not load your sessions.
        </div>
        <div v-else class="space-y-3">
            <div
                v-for="session in sessions"
                :key="session.id"
                class="glass-panel p-4 flex flex-col sm:flex-row sm:items-center gap-4"
                :class="session.current ? 'border-blue-500/25' : ''"
            >
                <div class="w-10 h-10 rounded-lg bg-white/[0.04] border border-white/[0.07] flex items-center justify-center flex-shrink-0">
                    <Icon :name="session.device === 'Mobile' ? 'lucide:smartphone' : session.device === 'Tablet' ? 'lucide:tablet' : 'lucide:monitor'" class="text-lg text-gray-400" />
                </div>

                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 flex-wrap">
                        <span class="text-sm font-semibold text-white">{{ session.browser }} on {{ session.operatingSystem }}</span>
                        <span v-if="session.current" class="px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-300 text-fine font-semibold uppercase tracking-wider">This device</span>
                    </div>
                    <div class="text-xs text-gray-500 mt-1 truncate">{{ locationLabel(session) }}</div>
                    <div class="text-fine text-gray-600 mt-1">
                        Last active {{ timeAgo(session.lastSeenAt) }} · Signed in {{ formatDateTime(session.createdAt) }}
                    </div>
                </div>

                <button
                    class="self-start sm:self-center px-3 py-2 rounded-lg text-xs font-medium border border-white/[0.08] text-gray-400 hover:text-red-300 hover:border-red-500/30 hover:bg-red-500/[0.06] transition-colors cursor-pointer disabled:opacity-50"
                    :disabled="revokingId === session.id"
                    @click="revokeSession(session)"
                >
                    <Icon v-if="revokingId === session.id" name="lucide:loader-2" class="animate-spin mr-1" />
                    {{ session.current ? 'Log out' : 'Revoke' }}
                </button>
            </div>
        </div>
    </div>
</template>
