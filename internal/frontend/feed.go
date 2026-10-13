package frontend

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/eve-kill/shrike/internal/api"
	"github.com/rs/zerolog/log"
)

const rssDateFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	LastBuild   string    `xml:"lastBuildDate"`
	Self        rssSelf   `xml:"atom:link"`
	Image       rssImage  `xml:"image"`
	Items       []rssItem `xml:"item"`
}

type rssSelf struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssImage struct {
	URL   string `xml:"url"`
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

type rssItem struct {
	Title       string        `xml:"title"`
	Link        string        `xml:"link"`
	GUID        rssGUID       `xml:"guid"`
	PubDate     string        `xml:"pubDate"`
	Description string        `xml:"description"`
	Enclosure   *rssEnclosure `xml:"enclosure,omitempty"`
}

type rssGUID struct {
	Permalink string `xml:"isPermaLink,attr"`
	Value     string `xml:",chardata"`
}

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length int    `xml:"length,attr"`
}

func (s *site) feed(w http.ResponseWriter, r *http.Request) {
	if s.options.Homepage == nil {
		http.Error(w, "Feed unavailable", http.StatusServiceUnavailable)
		return
	}
	page, err := s.options.Homepage.LatestKills(r.Context(), 1, 50)
	if err != nil {
		log.Error().Err(err).Msg("load RSS feed")
		http.Error(w, "Feed unavailable", http.StatusServiceUnavailable)
		return
	}
	feed := buildRSS(page.Kills)
	data, err := xml.Marshal(feed)
	if err != nil {
		log.Error().Err(err).Msg("encode RSS feed")
		http.Error(w, "Feed unavailable", http.StatusInternalServerError)
		return
	}
	data = append([]byte(xml.Header), data...)
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, s-maxage=60, stale-while-revalidate=120")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if r.Method != http.MethodHead {
		_, _ = bytes.NewReader(data).WriteTo(w)
	}
}

func buildRSS(kills []api.KilllistRow) rssFeed {
	buildTime := time.Now().UTC()
	if len(kills) > 0 && !kills[0].KillmailTime.IsZero() {
		buildTime = kills[0].KillmailTime.UTC()
	}
	feed := rssFeed{Version: "2.0", AtomNS: "http://www.w3.org/2005/Atom", Channel: rssChannel{
		Title: "EVE-KILL — Latest Kills", Link: canonicalOrigin,
		Description: "Real-time killmail feed from EVE Online — ship destructions, ISK values, and combat data from New Eden.",
		Language:    "en", LastBuild: buildTime.Format(rssDateFormat),
		Self:  rssSelf{Href: canonicalOrigin + "/feed.xml", Rel: "self", Type: "application/rss+xml"},
		Image: rssImage{URL: canonicalOrigin + "/icon.png", Title: "EVE-KILL", Link: canonicalOrigin},
	}}
	for _, kill := range kills {
		who := kill.VictimCharacterName
		if who == "" {
			who = kill.VictimCorporationName
		}
		if who == "" {
			who = "Unknown"
		}
		ship, system := kill.ShipName, kill.SolarSystemName
		if ship == "" {
			ship = "Unknown"
		}
		if system == "" {
			system = "Unknown"
		}
		isk := rssISK(kill.TotalValue)
		link := canonicalOrigin + "/kill/" + strconv.FormatInt(kill.KillmailID, 10)
		item := rssItem{
			Title: fmt.Sprintf("%s | %s | %s ISK", ship, who, isk), Link: link,
			GUID:        rssGUID{Permalink: "true", Value: link},
			PubDate:     kill.KillmailTime.UTC().Format(rssDateFormat),
			Description: fmt.Sprintf("%s lost a %s worth %s ISK in %s.", who, ship, isk, system),
		}
		if kill.ShipTypeID > 0 {
			item.Enclosure = &rssEnclosure{URL: fmt.Sprintf("%s/images/types/%d/overlayrender?size=128", canonicalOrigin, kill.ShipTypeID), Type: "image/png"}
		}
		feed.Channel.Items = append(feed.Channel.Items, item)
	}
	return feed
}

func rssISK(value float64) string {
	switch {
	case value >= 1e9:
		return fmt.Sprintf("%.2fB", value/1e9)
	case value >= 1e6:
		return fmt.Sprintf("%.2fM", value/1e6)
	case value >= 1e3:
		return fmt.Sprintf("%.2fK", value/1e3)
	default:
		return fmt.Sprintf("%.0f", value)
	}
}
