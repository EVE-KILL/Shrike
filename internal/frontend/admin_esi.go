package frontend

import (
	"net/url"
	"strconv"
)

type adminEntity struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type adminESIData struct {
	Entity   *adminEntity
	Logs     map[string]any
	LogError bool
}

func adminESIEntity(query url.Values) *adminEntity {
	id, err := strconv.ParseInt(query.Get("character_id"), 10, 32)
	if err != nil || id <= 0 {
		return nil
	}
	name := query.Get("character_name")
	if name == "" {
		name = strconv.FormatInt(id, 10)
	}
	return &adminEntity{ID: id, Name: name, Type: "character"}
}
