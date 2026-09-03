<script setup lang="ts">
interface LoginPrefs { charKm: boolean; corpKm: boolean; delay: number }
const props = defineProps<{
    prefs: LoginPrefs
    delayOptions: number[]
    mobile?: boolean
}>()
const emit = defineEmits<{
    (e: 'set', key: keyof LoginPrefs, value: boolean | number): void
    (e: 'login'): void
}>()

const delayIndex = computed({
    get: () => {
        const i = props.delayOptions.indexOf(props.prefs.delay)
        return i === -1 ? 0 : i
    },
    set: (i: number) => emit('set', 'delay', props.delayOptions[i] ?? 0),
})

const delayLabel = (h: number) => h === 0 ? 'No delay' : h === 1 ? '1 hour' : `${h} hours`
</script>

<template>
    <div :class="mobile ? 'p-1' : 'w-72 p-3'">
        <div v-if="!mobile" class="mb-3 flex items-center gap-2.5">
            <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-white/[0.06] bg-white/[0.035] text-blue-400/70">
                <Icon name="lucide:log-in" class="text-sm" />
            </span>
            <div>
                <div class="text-sm font-medium text-gray-300">Login with EVE Online</div>
                <div class="text-fine text-gray-600">Choose what to synchronize</div>
            </div>
        </div>

        <!-- EVE SSO image button -->
        <button
            type="button"
            class="block w-full mb-4 hover:opacity-90 transition-opacity"
            @click="emit('login')"
            aria-label="Log in with EVE Online"
        >
            <img src="/eve-sso-login-white-large.png" alt="Log in with EVE Online" width="270" height="45" class="w-full h-auto" />
        </button>

        <!-- Scope checkboxes -->
        <div class="space-y-2 mb-4">
            <label class="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                <input
                    type="checkbox"
                    :checked="prefs.charKm"
                    @change="emit('set', 'charKm', ($event.target as HTMLInputElement).checked)"
                    class="w-4 h-4 accent-blue-500"
                />
                Character killmails
            </label>
            <label class="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                <input
                    type="checkbox"
                    :checked="prefs.corpKm"
                    @change="emit('set', 'corpKm', ($event.target as HTMLInputElement).checked)"
                    class="w-4 h-4 accent-blue-500"
                />
                Corporation killmails
            </label>
        </div>

        <!-- Delay slider -->
        <div class="border-t border-white/[0.08] pt-3">
            <div class="flex items-center justify-between mb-2">
                <span class="text-xs text-gray-400">Killmail delay</span>
                <span class="text-xs text-blue-400 font-medium tabular-nums">{{ delayLabel(prefs.delay) }}</span>
            </div>
            <input
                type="range"
                :min="0"
                :max="delayOptions.length - 1"
                :step="1"
                :value="delayIndex"
                @input="delayIndex = Number(($event.target as HTMLInputElement).value)"
                class="w-full accent-blue-500"
            />
            <div class="flex justify-between mt-1 px-0.5">
                <span v-for="h in delayOptions" :key="h" class="text-fine text-gray-600 tabular-nums">{{ h }}</span>
            </div>
        </div>
    </div>
</template>
