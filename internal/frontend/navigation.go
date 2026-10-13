package frontend

import (
	"embed"
	"encoding/json"
	"strings"
)

// These menus are the default navigation and community links from the Nuxt site.
// They are embedded so page rendering does not depend on the legacy web tree.
//
//go:embed data/*.json
var navigationData embed.FS

type navLink struct {
	Label    string     `json:"label"`
	Href     string     `json:"href"`
	Icon     string     `json:"icon"`
	Children []navGroup `json:"children"`
	External bool       `json:"external"`
}

type navGroup struct {
	Label string    `json:"label"`
	Items []navLink `json:"items"`
}

type communityLink struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	URL   string `json:"url"`
	Icon  string `json:"icon"`
	Title string `json:"title"`
}

func loadMenu[T any](path string) T {
	data, err := navigationData.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		panic(err)
	}
	return result
}

var navigation = loadMenu[[]navLink]("data/navigation.json")
var community = loadMenu[[]communityLink]("data/community.json")
var information = []navLink{
	{Label: "Blog", Href: "/blog", Icon: "file-text"},
	{Label: "FAQ", Href: "/faq", Icon: "circle-help"},
	{Label: "API Docs", Href: "/docs", Icon: "file-text"},
	{Label: "MCP", Href: "/mcp", Icon: "plug"},
	{Label: "Status", Href: "/status", Icon: "activity"},
	{Label: "Wallet", Href: "/wallet", Icon: "wallet-cards"},
	{Label: "About", Href: "/about", Icon: "info"},
	{Label: "Donate", Href: "/donate", Icon: "heart"},
}

func iconURL(name string) string {
	name = strings.TrimPrefix(name, "lucide:")
	if !commonIconNames[name] {
		if _, ok := manifest.files["lucide/"+name+".svg"]; ok {
			return assetURL("lucide/"+name+".svg") + "#icon"
		}
	}
	return assetURL("icons.svg") + "#" + name
}

var backgrounds = []string{"bg1", "bg2", "bg3", "bg4", "bg5", "bg6", "bg7", "bg8"}
