package frontend

import "net/http"

type graphTab struct{ ID, Label, Subtitle, Empty, Loading string }

var graphTabs = []graphTab{
	{"path_finder", "Path Finder", "Six degrees of EVE", "Search for two characters to find the shortest FLEW_WITH chain between them", "Searching for connection..."},
	{"entity_intel", "Allies & Enemies", "Lookup any group", "Search for an alliance or corporation to see their allies and enemies", "Analyzing relationships..."},
	{"hunting_grounds", "Hunting Grounds", "Where to find them", "Search for an alliance or corporation to see where they're active", "Scanning systems..."},
	{"hot_zones", "Hot Zones", "Active conflict areas", "No hot zones detected in the last 24 hours", "Scanning for hot zones..."},
	{"census", "Military Census", "FC / Logi / Cap count", "Search for an alliance to see its military composition", "Running census..."},
	{"spy_check", "Spy Check", "Cross-alliance activity", "Search for an alliance or corporation to check for cross-group activity", "Scanning for cross-alliance activity..."},
	{"migration", "Migration", "Corp member movement", "Search for a corporation to see member movements", "Tracking migrations..."},
	{"coalitions", "Coalitions", "Auto-detected blocs", "No coalitions detected", "Detecting coalitions..."},
	{"rivalries", "Rivalries", "Who fights who", "No rivalry data available", "Loading rivalries..."},
}

func (s *site) graphPage(r *http.Request) (Page, error) {
	active := "path_finder"
	for _, tab := range graphTabs {
		if tab.ID == r.URL.Query().Get("tab") {
			active = tab.ID
		}
	}
	// Match Nuxt's client-loaded graph tools: the document never waits for
	// Memgraph. Searches and each auto-loaded tab use the existing graph API.
	return Page{Metadata: Metadata{
		Title: "Graph Intelligence", OGTitle: "Graph Intelligence — EVE-KILL", CanonicalPath: "/graph", Indexable: true,
		Description: "Explore EVE Online character connections, coalition structures, and rivalries using graph analysis.",
		Stylesheets: []string{"utilities.css", "utility-page.css", "graph.css"}, ModuleScripts: []string{"graph.js"},
	}, Content: graphPageContent(active)}, nil
}
