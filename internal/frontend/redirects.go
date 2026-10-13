package frontend

import (
	"net/http"
	"net/url"
	"strconv"
)

// These URLs used to be handled by Nuxt or its ingress. Keep the canonical
// destinations available when Go is the only page renderer.
func (s *site) registerLegacyRedirects(mux *http.ServeMux) {
	notFound := s.pageHandler(func(*http.Request) (Page, error) { return Page{}, ErrNotFound })
	redirectNumeric := func(pattern, target string) {
		mux.HandleFunc("GET "+pattern, func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			if !validRedirectID(id) {
				notFound.ServeHTTP(w, r)
				return
			}
			// #nosec G710 -- target is a fixed local prefix and id passed numeric validation.
			http.Redirect(w, r, target+id, http.StatusMovedPermanently)
		})
	}
	redirectNumeric("/wars/{id}", "/war/")
	redirectNumeric("/killmail/{id}", "/kill/")
	redirectNumeric("/ship/{id}", "/item/")
	mux.HandleFunc("GET /ship/{id}/{tab}", func(w http.ResponseWriter, r *http.Request) {
		id, tab := r.PathValue("id"), r.PathValue("tab")
		if !validRedirectID(id) || tab == "" {
			notFound.ServeHTTP(w, r)
			return
		}
		// #nosec G710 -- fixed local path, validated numeric id, escaped tab segment.
		http.Redirect(w, r, "/item/"+id+"/"+url.PathEscape(tab), http.StatusMovedPermanently)
	})
	for _, pattern := range []string{"/shipgroup/{id}", "/shipgroup/{id}/{tab}"} {
		mux.HandleFunc("GET "+pattern, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/market/ships", http.StatusMovedPermanently)
		})
	}
}

func validRedirectID(id string) bool {
	value, err := strconv.ParseUint(id, 10, 64)
	return err == nil && value > 0
}
