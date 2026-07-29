package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/jackc/pgx/v5"
)

// Database is the read-only Postgres surface used by MCP tools.
type Database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// GraphDatabase is the read-only Memgraph surface used by intelligence tools.
type GraphDatabase interface {
	Read(context.Context, string, map[string]any) ([]map[string]any, error)
}

// Dependencies are shared by the MCP and Huma transports.
type Dependencies struct {
	DB      Database
	Graph   GraphDatabase
	BaseURL string
}

type EntityType string

const (
	EntityCharacter     EntityType = "character"
	EntityCorporation   EntityType = "corporation"
	EntityAlliance      EntityType = "alliance"
	EntityShip          EntityType = "ship"
	EntityItem          EntityType = "item"
	EntitySystem        EntityType = "system"
	EntityRegion        EntityType = "region"
	EntityConstellation EntityType = "constellation"
	EntityFaction       EntityType = "faction"
)

var entityTypes = []EntityType{
	EntityCharacter,
	EntityCorporation,
	EntityAlliance,
	EntityShip,
	EntityItem,
	EntitySystem,
	EntityRegion,
	EntityConstellation,
	EntityFaction,
}

// StringOrInt64 preserves the existing MCP convention that an entity or type
// may be addressed by either its display name or numeric identifier.
type StringOrInt64 struct {
	text   string
	number int64
	isInt  bool
}

func StringRef(value string) StringOrInt64 {
	return StringOrInt64{text: value}
}

func IntRef(value int64) StringOrInt64 {
	return StringOrInt64{number: value, isInt: true}
}

func (v StringOrInt64) IsInt() bool {
	return v.isInt
}

func (v StringOrInt64) Int64() int64 {
	return v.number
}

func (v StringOrInt64) String() string {
	if v.isInt {
		return strconv.FormatInt(v.number, 10)
	}
	return v.text
}

func (v StringOrInt64) MarshalJSON() ([]byte, error) {
	if v.isInt {
		return json.Marshal(v.number)
	}
	return json.Marshal(v.text)
}

func (v *StringOrInt64) UnmarshalJSON(data []byte) error {
	if v == nil {
		return fmt.Errorf("cannot decode into nil StringOrInt64")
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("expected a string or integer")
	}
	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*v = StringRef(value)
		return nil
	}
	var value int64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("expected a string or integer: %w", err)
	}
	*v = IntRef(value)
	return nil
}

// Schema supplies Huma with the same union used by the MCP JSON Schema.
func (StringOrInt64) Schema(_ huma.Registry) *huma.Schema {
	return &huma.Schema{
		OneOf: []*huma.Schema{
			{Type: huma.TypeString},
			{Type: huma.TypeInteger, Format: "int64"},
		},
	}
}

func mcpSchemaOptions() *jsonschema.ForOptions {
	return &jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[StringOrInt64](): {
				OneOf: []*jsonschema.Schema{
					{Type: "string"},
					{Type: "integer"},
				},
			},
			reflect.TypeFor[StatsWindow](): statsWindowJSONSchema(),
		},
	}
}

// StatsWindow preserves the established MCP wire contract where lifetime
// results use the string "lifetime" and bounded results use a date object.
type StatsWindow struct {
	Lifetime bool
	Since    string
	Until    string
}

func LifetimeWindow() StatsWindow {
	return StatsWindow{Lifetime: true}
}

func BoundedWindow(since, until string) StatsWindow {
	return StatsWindow{Since: since, Until: until}
}

func (v StatsWindow) MarshalJSON() ([]byte, error) {
	if v.Lifetime {
		return json.Marshal("lifetime")
	}
	return json.Marshal(struct {
		Since string `json:"since"`
		Until string `json:"until"`
	}{Since: v.Since, Until: v.Until})
}

func (StatsWindow) Schema(_ huma.Registry) *huma.Schema {
	return &huma.Schema{OneOf: []*huma.Schema{
		{Type: huma.TypeString, Enum: []any{"lifetime"}},
		{
			Type: huma.TypeObject,
			Properties: map[string]*huma.Schema{
				"since": {Type: huma.TypeString},
				"until": {Type: huma.TypeString},
			},
			Required: []string{"since", "until"},
		},
	}}
}

func statsWindowJSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{OneOf: []*jsonschema.Schema{
		{Type: "string", Enum: []any{"lifetime"}},
		{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"since": {Type: "string"},
				"until": {Type: "string"},
			},
			Required: []string{"since", "until"},
		},
	}}
}

type Entity struct {
	ID     int64      `json:"id"`
	Name   string     `json:"name"`
	Type   EntityType `json:"type"`
	Ticker *string    `json:"ticker"`
	URL    string     `json:"url"`
}

type ResolvedEntity struct {
	ID     int64
	Name   string
	Type   EntityType
	Ticker *string
}

func (r ResolvedEntity) Public(baseURL string) Entity {
	return Entity{
		ID:     r.ID,
		Name:   r.Name,
		Type:   r.Type,
		Ticker: r.Ticker,
		URL:    entityURL(baseURL, r.Type, r.ID),
	}
}

func entityURL(baseURL string, entityType EntityType, id int64) string {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://eve-kill.com"
	}
	return strings.TrimRight(baseURL, "/") + "/" + string(entityType) + "/" +
		strconv.FormatInt(id, 10)
}

func killmailURL(baseURL string, id int64) string {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://eve-kill.com"
	}
	return strings.TrimRight(baseURL, "/") + "/kill/" + strconv.FormatInt(id, 10)
}
