<script setup lang="ts">
useHead({ title: 'Support' })
useSeoMeta({
    description: 'Support EVE-KILL — help keep the community killboard running. Donations go directly to server costs and ongoing development.',
    ogTitle: 'Support EVE-KILL',
    ogDescription: 'Help keep EVE-KILL running — donate to support server costs and development.',
})
</script>

<template>
    <InfoPage
        title="Support the Project"
        subtitle="EVE-KILL runs on donated ISK and a few servers. Here is where it goes."
        icon="lucide:heart"
    >
        <div class="glass-panel p-5 md:p-6">

            <p class="text-base text-center text-gray-400 mb-4 leading-relaxed">
                If you value our service and wish to contribute to its continued growth and improvement,
                please consider supporting our development efforts through one of these options.
            </p>

            <div class="border-l-4 border-blue-500 pl-4 py-2 bg-white/[0.04] rounded mb-8">
                <p class="text-sm italic text-gray-300">
                    We firmly believe in providing our services without barriers — using EVE-KILL will always remain
                    completely free. Donations are entirely voluntary expressions of support, and we maintain our
                    commitment to equal access and functionality for all users regardless of contribution status.
                </p>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
                <!-- Markee Dragon -->
                <div class="flex flex-col items-center p-6 border border-gray-700 rounded-lg bg-white/[0.04] hover:border-blue-500/50 transition-colors">
                    <Icon name="lucide:shopping-cart" class="w-10 h-10 text-blue-400 mb-4" />
                    <h3 class="text-xl font-semibold text-white mb-2">Game Time &amp; PLEX</h3>
                    <p class="text-center text-sm text-gray-400 mb-4 leading-relaxed">
                        Support our development while enhancing your EVE Online experience by purchasing game time
                        or PLEX through our official MarkeeDragon affiliate link. Each transaction directly contributes
                        to our ongoing development efforts without any additional cost to you.
                    </p>
                    <a href="https://store.markeedragon.com/affiliate.php?id=1104&redirect=index.php?cat=4"
                        target="_blank" rel="noopener noreferrer"
                        class="w-full block text-center px-4 py-3 rounded-lg bg-blue-600 text-white hover:bg-blue-500 transition-colors text-sm font-medium">
                        Go to MarkeeDragon — use code <strong>evekill</strong> for 3% off
                    </a>
                </div>

                <!-- Ko-Fi -->
                <div class="flex flex-col items-center p-6 border border-gray-700 rounded-lg bg-white/[0.04] hover:border-blue-500/50 transition-colors">
                    <Icon name="lucide:coffee" class="w-10 h-10 text-blue-400 mb-4" />
                    <h3 class="text-xl font-semibold text-white mb-2">Buy Us a Coffee</h3>
                    <p class="text-center text-sm text-gray-400 mb-4 leading-relaxed">
                        Help sustain our infrastructure and fuel future innovations with a direct contribution
                        through Ko-Fi. Choose between one-time donations or recurring support — every contribution,
                        regardless of size, makes a meaningful difference. Donors also get a role on Discord.
                    </p>
                    <a href="https://ko-fi.com/evekill"
                        target="_blank" rel="noopener noreferrer"
                        class="w-full block text-center px-4 py-3 rounded-lg bg-pink-600 text-white hover:bg-pink-500 transition-colors text-sm font-medium">
                        Tip a Coffee (Ko-fi)
                    </a>
                </div>
            </div>

            <p class="text-center text-gray-400 leading-relaxed">
                We extend our sincere gratitude to everyone who supports EVE-KILL. Your contributions directly
                enable us to enhance our services, maintain reliable infrastructure, and develop new features
                for the entire EVE Online community. Your generosity makes a tangible difference while keeping
                our platform accessible to all.
            </p>
        </div>
    </InfoPage>
</template>
