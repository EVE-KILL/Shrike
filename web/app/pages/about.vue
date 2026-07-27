<script setup lang="ts">
useHead({ title: 'About — Community Killboard Since 2007' })
useSeoMeta({
    description: 'Learn about EVE-KILL — a community-driven killboard for EVE Online founded in 2007. Our mission, history, and how to get in touch.',
    ogTitle: 'About EVE-KILL',
    ogDescription: 'The story behind EVE-KILL — a community-driven killboard for EVE Online since 2007.',
})

useSchemaOrg([
    defineWebPage({
        '@type': 'AboutPage',
        'name': 'About EVE-KILL',
        'description': 'EVE-KILL is a community-driven killboard for EVE Online, founded in 2007.',
    }),
])
</script>

<template>
    <InfoPage
        title="About"
        subtitle="A community-run killboard for EVE Online, going since 2007."
        icon="lucide:info"
    >
        <div class="glass-panel p-5 md:p-6">

            <div class="space-y-8">
                <section>
                    <h2 class="text-2xl font-bold text-white mb-3">Our Mission</h2>
                    <p class="text-gray-300 leading-relaxed">
                        Our mission is to deliver the most accurate and timely killboard experience for the EVE Online
                        community. We're dedicated to providing comprehensive analytics that illuminate the dynamic
                        landscape of New Eden, offering valuable insights into both PVP and PVE activities across the universe.
                    </p>
                </section>

                <section>
                    <h2 class="text-2xl font-bold text-white mb-3">History</h2>
                    <p class="text-gray-300 leading-relaxed">
                        Founded in 2007 by Ralle030583, Beansman, and Karbowiak, EVE-KILL established itself as a
                        cornerstone of the EVE Online community. Though later rebranded as zKillboard, evolving
                        circumstances led to the end of our collaborative development. Now, EVE-KILL returns with
                        renewed purpose, building on our rich legacy while embracing modern technologies and community needs.
                    </p>
                </section>

                <section>
                    <h2 class="text-2xl font-bold text-white mb-3">Contact Us</h2>
                    <p class="text-gray-300 leading-relaxed about-content">
                        We value your feedback and are here to help. For questions, suggestions, or technical support,
                        join our active <a href="https://discord.gg/Bz5gMHd" target="_blank" rel="noopener noreferrer">Discord community</a>
                        where our team regularly engages with users. Alternatively, you can submit issues or feature requests
                        directly through our <a href="https://github.com/EVE-KILL" target="_blank" rel="noopener noreferrer">GitHub repository</a>
                        for more technical matters.
                    </p>
                </section>

                <div class="flex flex-col sm:flex-row gap-4 justify-center pt-4">
                    <a href="https://discord.gg/Bz5gMHd" target="_blank" rel="noopener noreferrer"
                        class="inline-flex items-center gap-2 justify-center px-6 py-3 rounded-lg bg-[#5865F2] text-white hover:bg-[#4752C4] transition-colors text-sm font-medium">
                        <Icon name="lucide:message-circle" class="w-5 h-5" />
                        Join our Discord
                    </a>
                    <a href="https://github.com/EVE-KILL" target="_blank" rel="noopener noreferrer"
                        class="inline-flex items-center gap-2 justify-center px-6 py-3 rounded-lg bg-gray-700 text-white hover:bg-gray-600 transition-colors text-sm font-medium">
                        <Icon name="lucide:github" class="w-5 h-5" />
                        GitHub Repository
                    </a>
                </div>
            </div>
        </div>
    </InfoPage>
</template>

<style scoped>
.about-content :deep(a) {
    color: #00aaff;
    text-decoration: none;
    font-weight: 500;
    transition: all 0.2s ease;
}
.about-content :deep(a:hover) {
    text-decoration: underline;
    opacity: 0.9;
}
</style>
