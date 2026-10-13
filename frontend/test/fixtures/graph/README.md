page.html was captured directly from the Go graphPageContent component on
2026-10-08 by a temporary package test, removed after capture. It preserves all
nine panels, tab/search/type controls, classes and declared actions. DOM regressions
use controlled graph/search data modeled on graph_api.go, including nested path
affiliations, coalition cooperation and census character totals. No Memgraph or
production data was changed. Browser/phone checks remain a separate gate.
