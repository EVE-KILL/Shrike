<script setup lang="ts">
useHead({ title: "FAQ" });
useSeoMeta({
  description:
    "Frequently asked questions about EVE-KILL — killmails, combat points, rankings, item pricing, accounts, and the API.",
  ogTitle: "FAQ — EVE-KILL",
  ogDescription:
    "Answers to common questions about EVE-KILL, killmails, points, rankings, pricing, and the API.",
});

const items = [
  {
    key: "killmail",
    label: "What is a killmail?",
    icon: "lucide:file-text",
    content:
      "A killmail is the combat record EVE Online creates when a ship or structure is destroyed. It identifies the victim, participating attackers, damage dealt, location, ship, and destroyed or dropped items.",
  },
  {
    key: "killmails",
    label: "How are kills processed?",
    icon: "lucide:mail",
    content:
      "Killmails enter a queue, are validated and enriched with names, ship information, historical prices, categories, combat points, and related statistics. New kills appear on the site and API as soon as processing finishes. Delays can still occur during EVE downtime, unusually heavy traffic, or upstream outages.",
  },
  {
    key: "points",
    label: "How are combat points awarded?",
    icon: "lucide:swords",
    content:
      "<p>Each killmail has one combat-point pool. Ten percent is reserved for participation and divided equally among the player characters on the kill; the remaining ninety percent is divided according to damage dealt.</p><p>This gives tackle, logistics, and other low-damage participants some credit without making their contribution equal to the pilots dealing most of the damage. NPC attackers do not receive points. Integer rounding is balanced so the complete point pool is always awarded.</p>",
  },
  {
    key: "rankings",
    label: "How do rankings and EVE-KILL Rating work?",
    icon: "lucide:trophy",
    content:
      "<p>Rankings total combat points within the selected period. The available views cover the last hour, 7, 14, 30, 90, 180, and 365 days, plus all time. Characters, corporations, alliances, ships, systems, and regions are ranked separately.</p><p>Characters also earn achievement points. A corporation receives the combined achievement points of its current members, and an alliance receives the combined total of its current member characters. Achievement standing can add up to 3/7 of combat points, producing a 70/30 combat-to-achievement weighting at the top of the achievement distribution.</p><p>EVE-KILL Rating is open-ended: it is not capped at 1,000. Ships, systems, and regions use combat points directly because achievements belong to pilots and their organisations. The hourly view is calculated live; longer periods are generated from the statistics rollups.</p>",
  },
  {
    key: "pricing",
    label: "How are item prices calculated?",
    icon: "lucide:dollar-sign",
    content:
      "EVE-KILL stores daily market prices from The Forge, home of Jita. A killmail is valued using the price snapshot for the date of the loss, so an older kill keeps its historical valuation instead of changing with today's market. Items without usable market data may use an available fallback value.",
  },
  {
    key: "api",
    label: "Do you offer an API?",
    icon: "lucide:code",
    content:
      'Yes. The public, read-only API powers the website and provides killmails, statistics, rankings, universe data, and other features. Browse the current endpoints, parameters, examples, and response schemas in the <a href="/docs">API documentation</a>.',
  },
  {
    key: "donations",
    label: "Can I support EVE-KILL?",
    icon: "lucide:heart",
    content:
      'Yes. The optional <a href="/donate">Support page</a> lists the available ways to help with infrastructure and development costs. The public killboard remains available without a contribution.',
  },
  {
    key: "ads",
    label: "Do you run ads or track visitors?",
    icon: "lucide:shield-check",
    content:
      "EVE-KILL does not use an advertising network or sell visitor data. We use limited first-party analytics to understand site usage and diagnose problems.",
  },
  {
    key: "account",
    label: "Do I need an account?",
    icon: "lucide:user",
    content:
      "No account is required to browse killmails, categories, rankings, battles, campaigns, or the public API. EVE SSO is used for account features such as comments and creating or managing content. The login flow shows the permissions being requested.",
  },
  {
    key: "contact",
    label: "How can I contact EVE-KILL?",
    icon: "lucide:message-square",
    content:
      'Join the <a href="https://discord.gg/Bz5gMHd" target="_blank" rel="noopener noreferrer">EVE-KILL Discord</a> for discussion and support. Bugs and technical feature requests can also be reported through the <a href="https://github.com/EVE-KILL" target="_blank" rel="noopener noreferrer">EVE-KILL GitHub organisation</a>.',
  },
];

// Strip HTML tags for schema.org plain-text answers
const stripHtml = (html: string) => html.replace(/<[^>]*>/g, "");

useSchemaOrg([
  defineWebPage({ "@type": "FAQPage" }),
  ...items.map((item) => ({
    "@type": "Question" as const,
    name: item.label,
    acceptedAnswer: {
      "@type": "Answer" as const,
      text: stripHtml(item.content),
    },
  })),
]);

const openItem = ref<string | null>(null);
const toggle = (key: string) => {
  openItem.value = openItem.value === key ? null : key;
};
</script>

<template>
  <InfoPage
    title="Frequently Asked Questions"
    subtitle="How killmails get here, where prices come from, and what the API will and won't do."
    icon="lucide:circle-help"
  >
    <div class="glass-panel p-2">
      <div class="divide-y divide-gray-700/50 rounded-lg">
        <div v-for="item in items" :key="item.key" class="last:border-b-0">
          <button
            class="w-full flex items-center justify-between py-5 px-3 text-left hover:bg-white/[0.02] transition-colors"
            @click="toggle(item.key)"
          >
            <div class="flex items-center gap-3">
              <Icon
                :name="item.icon"
                class="w-5 h-5 text-blue-400 flex-shrink-0"
              />
              <span class="text-lg font-medium text-white">{{
                item.label
              }}</span>
            </div>
            <Icon
              :name="
                openItem === item.key
                  ? 'lucide:chevron-up'
                  : 'lucide:chevron-down'
              "
              class="w-5 h-5 text-gray-500 flex-shrink-0 transition-transform duration-200"
            />
          </button>
          <div
            v-show="openItem === item.key"
            class="pb-5 px-3 pl-11 text-gray-300 leading-relaxed faq-content"
            v-html="item.content"
          ></div>
        </div>
      </div>
    </div>
  </InfoPage>
</template>

<style scoped>
.faq-content :deep(a) {
  color: var(--color-brand-primary);
  text-decoration: none;
  font-weight: 500;
  transition: all 0.2s ease;
}
.faq-content :deep(a:hover) {
  text-decoration: underline;
  opacity: 0.9;
}
.faq-content :deep(p + p) {
  margin-top: 0.75rem;
}
</style>
