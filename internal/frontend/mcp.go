package frontend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eve-kill/shrike/internal/mcpserver"
)

const publicMCPBase = "https://eve-kill.com/api"

type mcpParameter struct {
	Name, Type, Description, Default string
	Required                         bool
}

type mcpTool struct {
	Name, Description, OutputSchema string
	Required                        []string
	Parameters                      []mcpParameter
}

type mcpGroup struct {
	Name, Anchor string
	Tools        []mcpTool
}

type mcpView struct {
	Groups []mcpGroup
	Tools  []mcpTool
	Error  string
}

var (
	mcpViewOnce sync.Once
	cachedMCP   mcpView
)

var mcpGroupOrder = []string{
	"Discovery", "Entity", "Killmails & Battles", "Fittings & Dogma", "SDE Lookups",
	"Intel (Memgraph)", "Intel (Derived)", "Routing", "Fun & Narrative", "Me (self-serve)",
}

func mcpCategory(name string) string {
	switch {
	case strings.HasPrefix(name, "me_"):
		return "Me (self-serve)"
	case name == "search":
		return "Discovery"
	case name == "route_danger":
		return "Routing"
	case member(name, "killmail_fitting", "dogma_eval", "fit_compare", "killmail_forensics", "doctrine_detect", "meta_pulse"):
		return "Fittings & Dogma"
	case member(name, "ship_info", "item_info", "system_info", "ship_compare"):
		return "SDE Lookups"
	case member(name, "killmail", "battle_report", "find_battles", "war_report"):
		return "Killmails & Battles"
	case member(name, "flies_with", "hunts_in", "hunted_by", "preys_on", "compare"):
		return "Intel (Memgraph)"
	case member(name, "coalition_graph", "character_history", "pilot_efficiency", "kills_with", "ships_used"):
		return "Intel (Derived)"
	case member(name, "system_pulse", "global_pulse", "expensive_losses", "capsuleer_dossier", "killmail_story"):
		return "Fun & Narrative"
	default:
		return "Entity"
	}
}

func member(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func mcpCatalogView() mcpView {
	mcpViewOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tools, err := mcpserver.Catalog(ctx)
		if err != nil {
			cachedMCP.Error = err.Error()
			return
		}
		groups := make(map[string][]mcpTool)
		for _, tool := range tools {
			var schema struct {
				Properties map[string]struct {
					Type        any    `json:"type"`
					Description string `json:"description"`
					Enum        []any  `json:"enum"`
					Default     any    `json:"default"`
				} `json:"properties"`
				Required []string `json:"required"`
			}
			encoded, err := json.Marshal(tool.InputSchema)
			if err != nil || json.Unmarshal(encoded, &schema) != nil {
				continue
			}
			entry := mcpTool{Name: tool.Name, Description: tool.Description, Required: schema.Required}
			paramNames := make([]string, 0, len(schema.Properties))
			for name := range schema.Properties {
				paramNames = append(paramNames, name)
			}
			sort.Strings(paramNames)
			for _, name := range paramNames {
				property := schema.Properties[name]
				param := mcpParameter{Name: name, Type: mcpSchemaType(property.Type), Description: property.Description, Required: member(name, schema.Required...)}
				if len(property.Enum) > 0 {
					values := make([]string, 0, len(property.Enum))
					for _, value := range property.Enum {
						encoded, _ := json.Marshal(value)
						values = append(values, string(encoded))
					}
					param.Type += " (" + strings.Join(values, " | ") + ")"
				}
				if property.Default != nil {
					encoded, _ := json.Marshal(property.Default)
					param.Default = "default: " + string(encoded)
				}
				entry.Parameters = append(entry.Parameters, param)
			}
			output, err := json.MarshalIndent(tool.OutputSchema, "", "  ")
			if err == nil && string(output) != "null" {
				entry.OutputSchema = string(output)
			}
			cachedMCP.Tools = append(cachedMCP.Tools, entry)
			category := mcpCategory(entry.Name)
			groups[category] = append(groups[category], entry)
		}
		for _, name := range mcpGroupOrder {
			entries := groups[name]
			if len(entries) == 0 {
				continue
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
			cachedMCP.Groups = append(cachedMCP.Groups, mcpGroup{Name: name, Anchor: mcpAnchor(name), Tools: entries})
		}
	})
	return cachedMCP
}

func mcpSchemaType(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, part := range typed {
			parts = append(parts, fmt.Sprint(part))
		}
		return strings.Join(parts, " | ")
	default:
		return "any"
	}
}

func mcpAnchor(name string) string {
	var out strings.Builder
	out.WriteString("cat-")
	dash := false
	for _, char := range strings.ToLower(name) {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			out.WriteRune(char)
			dash = false
		} else if !dash {
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(out.String(), "-")
}

func mcpHTTPConfig() string {
	return fmt.Sprintf("{\n  \"mcpServers\": {\n    \"evekill\": {\n      \"type\": \"http\",\n      \"url\": \"%s/mcp\"\n    }\n  }\n}", publicMCPBase)
}

func joinNames(names []string) string { return strings.Join(names, ", ") }

func mcpDesktopConfig() string {
	return fmt.Sprintf("{\n  \"mcpServers\": {\n    \"evekill\": {\n      \"command\": \"npx\",\n      \"args\": [\n        \"-y\",\n        \"mcp-remote\",\n        \"%s/mcp\"\n      ]\n    }\n  }\n}", publicMCPBase)
}

func mcpCurlExample() string {
	return fmt.Sprintf("curl -s -X POST %s/mcp \\\n  -H 'content-type: application/json' \\\n  -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"search\",\"arguments\":{\"query\":\"Jita\"}}}'", publicMCPBase)
}

var mcpToolOrder = []string{
	"search", "entity_overview", "entity_kills", "entity_top", "entity_timeline", "killmail", "killmail_story",
	"flies_with", "hunts_in", "hunted_by", "preys_on", "compare", "kills_with", "ships_used",
	"battle_report", "find_battles", "system_pulse", "expensive_losses", "capsuleer_dossier", "route_danger",
	"me_dossier", "me_overview", "me_kills", "me_flies_with", "me_hunts_in", "me_hunted_by", "me_preys_on",
	"me_kills_with", "me_ships_used", "me_timeline",
}

func mcpRank(name string) int {
	for index, value := range mcpToolOrder {
		if value == name {
			return index
		}
	}
	return len(mcpToolOrder) + int(name[0])
}

func buildMCPGuide(view mcpView) string {
	var b strings.Builder
	b.WriteString("# EVE-KILL MCP Server\n\n")
	b.WriteString("- [EVE-KILL](https://eve-kill.com/) — human-friendly killboard\n")
	b.WriteString("- [REST API documentation](https://eve-kill.com/docs) — OpenAPI reference\n")
	fmt.Fprintf(&b, "- [MCP endpoint](%s/mcp) — JSON-RPC over streamable HTTP\n\n", publicMCPBase)
	fmt.Fprintf(&b, "Endpoint: %s/mcp  (JSON-RPC over streamable HTTP)\n", publicMCPBase)
	b.WriteString("Protocol: MCP 2025-06-18\nNo auth or API key required; send an identifying User-Agent header\n\n")
	b.WriteString("## What this server does\n\nExposes EVE-KILL killmail and intel data to AI agents as a set of narrow, purpose-built tools.\n")
	b.WriteString("Covers entity lookup, killmail detail, character/corp/alliance stats, relationship graphs, battle reports, system activity, route danger and personalized queries.\n\n")
	b.WriteString("## Connecting a client\n\n### Native HTTP clients\n\n")
	fmt.Fprintf(&b, "    {\"mcpServers\":{\"evekill\":{\"type\":\"http\",\"url\":\"%s/mcp\"}}}\n\n", publicMCPBase)
	fmt.Fprintf(&b, "Claude Code: `claude mcp add --scope user --transport http evekill %s/mcp`\n\n", publicMCPBase)
	b.WriteString("### Claude Desktop via mcp-remote\n\n")
	fmt.Fprintf(&b, "    {\"mcpServers\":{\"evekill\":{\"command\":\"npx\",\"args\":[\"-y\",\"mcp-remote\",\"%s/mcp\"]}}}\n\n", publicMCPBase)
	b.WriteString("## How to call it\n\nPOST /mcp with JSON-RPC 2.0. Call `tools/list` to discover current tools, then `tools/call` with a name and arguments.\n")
	b.WriteString("Example: `POST /mcp {\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"search\",\"arguments\":{\"query\":\"Jita\"}}}`\n\n")
	b.WriteString("## Usage tips for LLMs\n\n1. Resolve named EVE entities with `search` before passing IDs to other tools.\n")
	b.WriteString("2. Use `me_*` tools for self-identified characters, passing the `me` argument every time.\n")
	b.WriteString("3. Use `capsuleer_dossier` for a one-shot profile and `find_battles` with `battle_report` for recent fights.\n")
	b.WriteString("4. `route_danger` supports shortest and safest stargate routes.\n")
	b.WriteString("5. Entity responses include canonical eve-kill.com URLs for citations.\n\n## Tools\n\n")
	if view.Error != "" {
		fmt.Fprintf(&b, "(Live tool list unavailable: %s. Try POST %s/mcp directly.)\n\n", view.Error, publicMCPBase)
	}
	tools := append([]mcpTool(nil), view.Tools...)
	sort.Slice(tools, func(i, j int) bool {
		a, b := mcpRank(tools[i].Name), mcpRank(tools[j].Name)
		if a != b {
			return a < b
		}
		return tools[i].Name < tools[j].Name
	})
	for _, tool := range tools {
		fmt.Fprintf(&b, "### %s\n%s\n", tool.Name, tool.Description)
		if len(tool.Parameters) == 0 {
			b.WriteString("Arguments: none\n")
		} else {
			b.WriteString("Arguments:\n")
			for _, param := range tool.Parameters {
				requirement := "optional"
				if param.Required {
					requirement = "required"
				}
				fmt.Fprintf(&b, "  - %s (%s) [%s", param.Name, requirement, param.Type)
				if param.Default != "" {
					fmt.Fprintf(&b, ", %s", param.Default)
				}
				b.WriteString("]")
				if param.Description != "" {
					fmt.Fprintf(&b, " — %s", param.Description)
				}
				b.WriteByte('\n')
			}
		}
		b.WriteString("Returns: see tools/list outputSchema or /api/docs for the full schema\n\n")
	}
	b.WriteString("## Notes and limits\n\n- Public read-only killboard data; no authentication or per-user state.\n")
	b.WriteString("- Memgraph relationship edges cover a rolling 90-day window.\n")
	b.WriteString("- ISK values use EVE-KILL price overlays when present, otherwise market prices.\n")
	b.WriteString("- Human-friendly site: https://eve-kill.com · REST API: https://eve-kill.com/api\n")
	return b.String()
}

func (s *site) mcpGuide(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300, s-maxage=300")
	_, _ = w.Write([]byte(buildMCPGuide(mcpCatalogView())))
}

func (s *site) mcpOutputFragment(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	for _, tool := range mcpCatalogView().Tools {
		if tool.Name != name || tool.OutputSchema == "" {
			continue
		}
		var output bytes.Buffer
		if err := mcpOutput(tool.OutputSchema).Render(r.Context(), &output); err != nil {
			http.Error(w, "Could not render schema", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300, s-maxage=300")
		_, _ = w.Write(output.Bytes())
		return
	}
	http.NotFound(w, r)
}
