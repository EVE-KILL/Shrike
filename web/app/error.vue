<script setup lang="ts">
const props = defineProps({
    error: Object,
})

const statusCode = computed(() => props.error?.statusCode || 404)

useHead({ title: computed(() => String(statusCode.value)) })
useSeoMeta({
    robots: 'noindex, nofollow',
})

const title = computed(() => {
    switch (statusCode.value) {
        case 404: return 'Page Not Found'
        case 403: return 'Access Forbidden'
        case 500: return 'Server Error'
        case 503: return 'Service Unavailable'
        default: return `Error ${statusCode.value}`
    }
})

const description = computed(() => {
    switch (statusCode.value) {
        case 404: return 'The page you\'re looking for doesn\'t exist or has been moved.'
        case 403: return 'You don\'t have permission to access this resource.'
        case 500: return 'Something went wrong on our end. Please try again later.'
        case 503: return 'The service is temporarily unavailable. Please check back soon.'
        default: return props.error?.message || 'An unexpected error occurred.'
    }
})

const icon = computed(() => {
    switch (statusCode.value) {
        case 403: return 'lucide:shield-x'
        case 500: return 'lucide:server-crash'
        case 503: return 'lucide:wrench'
        default: return 'lucide:alert-triangle'
    }
})

const handleError = () => clearError({ redirect: '/' })
</script>

<template>
    <NuxtLayout>
        <div class="flex flex-col items-center justify-center min-h-[60vh] px-4 text-center">
            <!-- Error card -->
            <div class="w-full max-w-lg rounded-xl border border-white/[0.08] bg-black/80 backdrop-blur-2xl shadow-2xl shadow-black/60 overflow-hidden">
                <!-- Status code + icon -->
                <div class="px-8 pt-10 pb-6">
                    <div class="flex items-center justify-center mb-6">
                        <Icon :name="icon" class="text-4xl text-gray-500" />
                    </div>
                    <div class="text-6xl font-bold text-white/20 mb-2">{{ statusCode }}</div>
                    <h1 class="text-xl font-semibold text-white mb-3">{{ title }}</h1>
                    <p class="text-sm text-gray-400 leading-relaxed max-w-sm mx-auto">{{ description }}</p>
                </div>

                <!-- Actions -->
                <div class="px-8 pb-8 flex flex-col sm:flex-row items-center justify-center gap-3">
                    <button
                        class="flex items-center gap-2 px-5 py-2.5 rounded-md text-sm font-medium text-white bg-blue-500/80 hover:bg-blue-500 transition-colors"
                        @click="handleError"
                    >
                        <Icon name="lucide:home" class="text-sm" />
                        Back to Home
                    </button>
                    <button
                        class="flex items-center gap-2 px-5 py-2.5 rounded-md text-sm font-medium text-gray-400 border border-white/[0.08] hover:bg-blue-500/[0.06] hover:text-blue-400 transition-colors"
                        @click="$router.go(-1)"
                    >
                        <Icon name="lucide:arrow-left" class="text-sm" />
                        Go Back
                    </button>
                </div>

                <!-- Separator + hint -->
                <div class="px-8 py-3 border-t border-white/[0.06] text-center">
                    <span class="text-xs text-gray-600">
                        If this keeps happening, please let us know on
                        <a href="https://github.com/EVE-KILL" target="_blank" rel="noopener noreferrer" class="text-blue-400/80 hover:text-blue-400">GitHub</a>
                    </span>
                </div>
            </div>
        </div>
    </NuxtLayout>
</template>
