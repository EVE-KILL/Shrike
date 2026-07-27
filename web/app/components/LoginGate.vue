<script setup lang="ts">
const props = defineProps<{
    message: string
    redirect?: string
}>()

const route = useRoute()
const { login } = useAuth()

const doLogin = () => login({ redirect: props.redirect || route.fullPath })
</script>

<template>
    <div class="glass-panel p-8 text-center">
        <Icon name="lucide:log-in" class="text-3xl text-gray-500 mb-3" />
        <p class="text-sm text-gray-400 mb-4">{{ message }}</p>
        <button type="button" class="px-4 py-2 rounded-lg text-sm font-medium bg-blue-500 text-white hover:bg-blue-600 transition-colors cursor-pointer" @click="doLogin">
            Log in with EVE Online
        </button>
    </div>
</template>
