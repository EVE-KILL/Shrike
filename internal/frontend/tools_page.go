package frontend

var toolCards = []struct{ Label, Href, Icon, Description string }{
	{"Killmail Categories", "/categories", "tags", "Browse killmail categories by space, engagement, involvement, faction warfare, value, hull class, and technology."},
	{"Advanced Search", "/advancedsearch", "search", "Filter killmails across characters, corps, alliances, systems, ships, and time ranges."},
	{"Battle Generator", "/battlegenerator", "shield", "Paste killmail IDs to build a shareable battle report on the fly."},
	{"Roam Report", "/tools/roam-report", "route", "Paste a fleet list and turn its recorded fights into a shareable combat trail."},
	{"Campaign Creator", "/campaigncreator", "flag", "Bundle killmails by attackers, victims, systems or timeframe into a named campaign."},
	{"Comments", "/comments", "message-square", "Recent community comments across killmails, battles, and entities."},
	{"Local Scan", "/tools/localscan", "scan-search", "Paste an in-game Local list to instantly break it down by corp, alliance, and ship composition."},
	{"D-Scan", "/tools/dscan", "radar", "Parse a directional scan export into a clean ship-by-ship summary."},
	{"Ship Fittings", "/fits", "wrench", "Browse and search community-submitted ship fittings with modules and stats."},
	{"Legacy Kills", "/legacy", "archive", "Pre-API killmail archive from 2003–2007 with search and top-loss lists."},
}
