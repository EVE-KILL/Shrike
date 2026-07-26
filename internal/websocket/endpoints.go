package websocket

// Endpoint is one public WebSocket stream and its Redis pub/sub channel.
//
// Paths are deliberately relative to BasePath. The browser-facing contract is
// /ws/<path>; keeping the prefix out of every definition prevents a future
// route move from silently changing the channel or the frame's channel field.
type Endpoint struct {
	Name         string
	Path         string
	Channel      string
	TopicRouting bool
}

const BasePath = "/ws"

var endpoints = []*Endpoint{
	{Name: "Killmails", Path: "/killmails", Channel: "ws:killmails", TopicRouting: true},
	{Name: "Kill List", Path: "/killlist", Channel: "ws:killlist", TopicRouting: true},
	{Name: "Comments", Path: "/comments", Channel: "ws:comments"},
	{Name: "Status", Path: "/status", Channel: "ws:status"},
	{Name: "Announcements", Path: "/announcements", Channel: "ws:announcements"},
}

var (
	endpointByPath    = indexEndpoints(func(endpoint *Endpoint) string { return BasePath + endpoint.Path })
	endpointByChannel = indexEndpoints(func(endpoint *Endpoint) string { return endpoint.Channel })
)

func indexEndpoints(key func(*Endpoint) string) map[string]*Endpoint {
	out := make(map[string]*Endpoint, len(endpoints))
	for _, endpoint := range endpoints {
		out[key(endpoint)] = endpoint
	}
	return out
}

var availableTopics = map[string][]string{
	"universal": {"all"},
	"kill_type": {"solo", "npc"},
	"value":     {"10b", "5b"},
	"security":  {"highsec", "lowsec", "nullsec", "wspace", "abyssal"},
	"ship_class": {
		"big", "citadel", "capitals", "freighters", "supercarriers", "titans",
		"frigates", "destroyers", "cruisers", "battlecruisers", "battleships",
		"t1", "t2", "t3",
	},
	"location": {"system.{id}", "region.{id}", "constellation.{id}"},
	"entity":   {"victim.{id}", "attacker.{id}"},
}
