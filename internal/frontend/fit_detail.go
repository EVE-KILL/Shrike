package frontend

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
)

type savedFitRow struct {
	ID          string   `json:"fit_id"`
	OwnerID     int64    `json:"owner_character_id"`
	ShipID      int64    `json:"ship_type_id"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Visibility  int      `json:"visibility"`
	Rating      *float64 `json:"rating_avg"`
	Ratings     int      `json:"rating_count"`
}
type savedFitItem struct {
	Group    int    `json:"slot_group"`
	Ordinal  int    `json:"ordinal"`
	TypeID   int64  `json:"type_id"`
	State    int    `json:"state"`
	ChargeID *int64 `json:"charge_type_id"`
	Quantity int    `json:"quantity"`
}
type savedFitPayload struct {
	Fit          savedFitRow    `json:"fit"`
	Items        []savedFitItem `json:"items"`
	ViewerRating *int           `json:"viewer_rating"`
}
type fitEditorConfig struct {
	Detail bool
	Owner  bool
	Data   savedFitPayload
}

func (c fitEditorConfig) viewOnly() bool { return c.Detail && !c.Owner }
func (c fitEditorConfig) description() string {
	if c.Data.Fit.Description != nil {
		return *c.Data.Fit.Description
	}
	return ""
}
func (c fitEditorConfig) ratingOn(star int) bool {
	if c.Data.ViewerRating != nil {
		return star <= *c.Data.ViewerRating
	}
	return catalogRatingOn(c.Data.Fit.Rating, star)
}

var savedFitID = regexp.MustCompile(`^[a-zA-Z0-9]{14}$`)

func pluralStars(stars int) string {
	if stars == 1 {
		return "star"
	}
	return "stars"
}

// The existing fitting API owns access control. Never cache a saved-fit
// document publicly: its ratings and permitted contents depend on the session.
func (s *site) savedFitPage(r *http.Request) (Page, error) {
	id := r.PathValue("id")
	config := fitEditorConfig{Detail: true}
	status := 0
	if !savedFitID.MatchString(id) {
		status = http.StatusNotFound
	} else if err := s.fittingAPIRead(r, "/api/fit/"+id, url.Values{}, &config.Data); err != nil {
		status = http.StatusInternalServerError
		var apiErr fittingAPIError
		if errors.As(err, &apiErr) && (apiErr.Status == 403 || apiErr.Status == 404) {
			status = apiErr.Status
		}
	} else if config.Data.Fit.ID != id || config.Data.Fit.ShipID <= 0 {
		status = http.StatusInternalServerError
	} else {
		user, err := s.browserSession(r)
		if err != nil {
			status = http.StatusInternalServerError
		} else {
			config.Owner = user != nil && int64(user.CharacterID) == config.Data.Fit.OwnerID
		}
	}
	if status != 0 {
		title := "Something went wrong"
		if status == 403 {
			title = "This fit is private"
		}
		if status == 404 {
			title = "Fit not found"
		}
		return Page{Status: status, Metadata: Metadata{Title: title, Stylesheets: []string{"fit-editor.css", "fit-detail.css"}}, Content: savedFitError(id, status, title)}, nil
	}
	return Page{Metadata: Metadata{
		Title: config.Data.Fit.Name + " — Fit", Description: config.description(), CanonicalPath: "/fit/" + id,
		Indexable: config.Data.Fit.Visibility == 3, NeedsWASM: true, CommentEditor: true, CommentMedia: true,
		Stylesheets: []string{"fit-editor.css", "fit-wheel.css", "fit-detail.css", "comment-thread.css"}, ModuleScripts: []string{"fit-editor.js", "fit-detail.js"},
	}, Content: fitEditorContent(config)}, nil
}
