package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func fixtureEntitySearch(w http.ResponseWriter, r *http.Request) bool {
	query := strings.ToLower(r.URL.Query().Get("q"))
	if r.Method != "GET" || r.URL.Path != "/api/search" || !strings.Contains(query, "fixture") {
		return false
	}
	rows := []map[string]string{}
	for _, entity := range []struct{ ID, Name, Type string }{{"character_4242", "Fixture Owner", "character"}, {"character_4343", "Fixture Reader", "character"}, {"corporation_10", "Fixture Red", "corporation"}, {"corporation_11", "Fixture Blue", "corporation"}, {"alliance_70121", "Fixture Red Alliance", "alliance"}, {"region_90000001", "Fixture Region", "region"}, {"constellation_90000002", "Fixture Constellation", "constellation"}, {"system_90000003", "Fixture System", "system"}} {
		if strings.Contains(strings.ToLower(entity.Name), query) {
			rows = append(rows, map[string]string{"id": entity.ID, "name": entity.Name, "type": entity.Type})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{"hits": rows})
	return true
}
