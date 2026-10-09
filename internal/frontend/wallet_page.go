package frontend

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eve-kill/shrike/internal/api"
)

type walletPageView struct {
	Data     api.PublicWalletData
	Page     int
	Division int
}

func walletQuery(values url.Values) (int, int) {
	page, _ := strconv.Atoi(values.Get("page"))
	if page < 1 {
		page = 1
	}
	if page > 1000 {
		page = 1000
	}
	division, _ := strconv.Atoi(values.Get("division"))
	if division < 1 || division > 7 {
		division = 0
	}
	return page, division
}

func (s *site) walletView(r *http.Request) (walletPageView, error) {
	page, division := walletQuery(r.URL.Query())
	data, err := s.options.Wallet.PublicWallet(r.Context(), page, division)
	if err != nil {
		return walletPageView{}, err
	}
	return walletPageView{Data: data, Page: page, Division: division}, nil
}

func (s *site) walletPage(r *http.Request) (Page, error) {
	view, err := s.walletView(r)
	if err != nil {
		return Page{}, err
	}
	return Page{Metadata: Metadata{
		Title: "EVE-KILL Wallet", CanonicalPath: walletPath(view.Page, view.Division), Indexable: true,
		Description: "The public EVE-KILL.com corporation wallet balance and journal.",
		OGTitle:     "EVE-KILL Wallet", Stylesheets: []string{"wallet.css"}, ModuleScripts: []string{"wallet.js"},
	}, Content: walletPage(view), SharedCacheTTL: time.Minute}, nil
}

func (s *site) walletFragment(w http.ResponseWriter, r *http.Request) {
	view, err := s.walletView(r)
	if err != nil {
		http.Error(w, "Could not load the corporation wallet", http.StatusServiceUnavailable)
		return
	}
	s.renderFragment(w, r, walletContents(view))
}

func walletPath(page, division int) string {
	values := url.Values{}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if division >= 1 && division <= 7 {
		values.Set("division", strconv.Itoa(division))
	}
	if len(values) == 0 {
		return "/wallet"
	}
	return "/wallet?" + values.Encode()
}

func walletDate(raw string) string {
	if raw == "" {
		return "Not yet"
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return "Not yet"
	}
	return parsed.Local().Format("02 Jan 2006, 15:04")
}

func walletIsk(raw *string) string {
	if raw == nil || *raw == "" {
		return "—"
	}
	value, err := strconv.ParseFloat(*raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return "—"
	}
	formatted := fmt.Sprintf("%.2f", value)
	parts := strings.SplitN(formatted, ".", 2)
	sign := ""
	if strings.HasPrefix(parts[0], "-") {
		sign = "-"
		parts[0] = strings.TrimPrefix(parts[0], "-")
	}
	for i := len(parts[0]) - 3; i > 0; i -= 3 {
		parts[0] = parts[0][:i] + "," + parts[0][i:]
	}
	return sign + parts[0] + "." + parts[1] + " ISK"
}

func walletRefType(value string) string {
	parts := strings.Split(value, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

func walletAmountTone(raw *string) string {
	if raw == nil {
		return ""
	}
	value, _ := strconv.ParseFloat(*raw, 64)
	if value > 0 {
		return "positive"
	}
	if value < 0 {
		return "negative"
	}
	return ""
}

func walletID(raw *int64) string {
	if raw == nil {
		return ""
	}
	return strconv.FormatInt(*raw, 10)
}
