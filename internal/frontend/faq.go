package frontend

// Keep the plain-text answers in sync with faqPage's visible copy. Search
// engines and screen readers should receive the same explanations.
var faqAnswers = []struct{ question, answer string }{
	{"What is a killmail?", "A killmail is the combat record EVE Online creates when a ship or structure is destroyed. It identifies the victim, participating attackers, damage dealt, location, ship, and destroyed or dropped items."},
	{"How are kills processed?", "Killmails enter a queue, are validated and enriched with names, ship information, historical prices, categories, combat points, and related statistics. New kills appear on the site and API as soon as processing finishes. Delays can still occur during EVE downtime, unusually heavy traffic, or upstream outages."},
	{"How are combat points awarded?", "Each killmail has one combat-point pool. Ten percent is reserved for participation and divided equally among the player characters on the kill; the remaining ninety percent is divided according to damage dealt. This gives tackle, logistics, and other low-damage participants some credit without making their contribution equal to the pilots dealing most of the damage. NPC attackers do not receive points. Integer rounding is balanced so the complete point pool is always awarded."},
	{"How do rankings and EVE-KILL Rating work?", "Rankings total combat points within the selected period. The available views cover the last hour, 7, 14, 30, 90, 180, and 365 days, plus all time. Characters, corporations, alliances, ships, systems, and regions are ranked separately. Characters also earn achievement points. A corporation receives the combined achievement points of its current members, and an alliance receives the combined total of its current member characters. Achievement standing can add up to 3/7 of combat points, producing a 70/30 combat-to-achievement weighting at the top of the achievement distribution. EVE-KILL Rating is open-ended: it is not capped at 1,000. Ships, systems, and regions use combat points directly because achievements belong to pilots and their organisations. The hourly view is calculated live; longer periods are generated from the statistics rollups. Explore the PvP statistics and leaderboards to compare pilots, corporations and alliances."},
	{"Where can I find ship fittings?", "Browse EVE Online ship fittings recovered from killmails, alliance doctrines and community-shared builds. Use fitting search to find examples by hull and fitted modules, or create a fit with the fitting tool."},
	{"How are item prices calculated?", "EVE-KILL stores daily market prices from The Forge, home of Jita. A killmail is valued using the price snapshot for the date of the loss, so an older kill keeps its historical valuation instead of changing with today's market. Items without usable market data may use an available fallback value."},
	{"Do you offer an API?", "Yes. The public, read-only API powers the website and provides killmails, statistics, rankings, universe data, and other features. Browse the current endpoints, parameters, examples, and response schemas in the API documentation."},
	{"Can I support EVE-KILL?", "Yes. The optional Support page lists the available ways to help with infrastructure and development costs. The public killboard remains available without a contribution."},
	{"Do you run ads or track visitors?", "EVE-KILL does not use an advertising network or sell visitor data. We use limited first-party analytics to understand site usage and diagnose problems."},
	{"Do I need an account?", "No account is required to browse killmails, categories, rankings, battles, campaigns, or the public API. EVE SSO is used for account features such as comments and creating or managing content. The login flow shows the permissions being requested."},
	{"How can I contact EVE-KILL?", "Join the EVE-KILL Discord for discussion and support. Bugs and technical feature requests can also be reported through the EVE-KILL GitHub organisation."},
}

func faqStructuredData() []any {
	data := []any{map[string]any{"@context": "https://schema.org", "@type": "FAQPage"}}
	for _, item := range faqAnswers {
		data = append(data, map[string]any{
			"@context": "https://schema.org",
			"@type":    "Question",
			"name":     item.question,
			"acceptedAnswer": map[string]any{
				"@type": "Answer",
				"text":  item.answer,
			},
		})
	}
	return data
}
