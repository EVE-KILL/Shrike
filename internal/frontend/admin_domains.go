package frontend

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type adminDomain struct {
	settingsDomain
	OwnerID   int64      `json:"user_id"`
	OwnerName string     `json:"owner_name"`
	Hostname  string     `json:"custom_hostname"`
	Pending   int        `json:"pending_assets"`
	Created   *time.Time `json:"created_at"`
	Updated   *time.Time `json:"updated_at"`
}
type adminDomainsData struct {
	Domains []adminDomain `json:"domains"`
	Failed  bool          `json:"-"`
}
type adminDomainAsset struct {
	ID          int64      `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Size        int64      `json:"file_size"`
	ContentType string     `json:"content_type"`
	Created     *time.Time `json:"created_at"`
	Reviewed    *time.Time `json:"reviewed_at"`
	Reason      string     `json:"reject_reason"`
}
type adminDomainDetailData struct {
	Domain *adminDomain       `json:"domain"`
	Assets []adminDomainAsset `json:"assets"`
}

func (data adminDomainsData) count(filter string) int {
	n := 0
	for _, d := range data.Domains {
		if filter == "all" || filter == "pending" && d.Pending > 0 || filter == "inactive" && !d.Active {
			n++
		}
	}
	return n
}
func (data adminDomainDetailData) pending() []adminDomainAsset {
	result := []adminDomainAsset{}
	for _, a := range data.Assets {
		if a.Status == "pending" {
			result = append(result, a)
		}
	}
	return result
}
func adminAssetTone(status string) string {
	switch status {
	case "approved":
		return "text-green-400 bg-green-500/20"
	case "rejected":
		return "text-red-400 bg-red-500/20"
	default:
		return "text-yellow-400 bg-yellow-500/20"
	}
}
func adminAssetSize(size int64) string {
	if size < 1024 {
		return strconv.FormatInt(size, 10) + " B"
	}
	if size < 1048576 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/1048576)
}
func adminDomainOwner(d adminDomain) string {
	return domainDefault(d.OwnerName, strconv.FormatInt(d.OwnerID, 10))
}
func adminDomainDetailPath(id int64) string { return "/admin/domains/" + strconv.FormatInt(id, 10) }
func adminAssetPreviewPath(domainID, assetID int64) string {
	return "/api" + adminDomainDetailPath(domainID) + "/assets/" + strconv.FormatInt(assetID, 10) + "/preview"
}
func (s *site) readAdminDomain(r *http.Request) (adminDomainDetailData, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id <= 0 {
		return adminDomainDetailData{}, accountPageError{http.StatusBadRequest}
	}
	var data adminDomainDetailData
	err = s.accountRead(r, "/api"+adminDomainDetailPath(id), &data)
	if err == nil && data.Domain == nil {
		err = ErrNotFound
	}
	return data, err
}
func (s *site) adminDomainPage(r *http.Request) (Page, error) {
	user, err := s.settingsIdentity(r)
	if err != nil {
		return Page{}, err
	}
	if user != nil && !user.Admin {
		return Page{RedirectPath: "/"}, nil
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 32)
	if err != nil || id <= 0 {
		return Page{Status: 400, Metadata: adminMetadata("Domain"), Content: errorPage("Invalid domain ID", "This domain ID is not valid.")}, nil
	}
	if user == nil {
		return Page{Metadata: adminMetadata("Domain"), Content: adminPageContent(adminView{})}, nil
	}
	data, err := s.readAdminDomain(r)
	if err != nil {
		var failure accountPageError
		if errors.Is(err, ErrNotFound) || (errors.As(err, &failure) && (failure.Status == 404 || failure.Status == 400)) {
			return Page{Status: 404, Metadata: adminMetadata("Domain"), Content: errorPage("Domain not found", "This domain is not available.")}, nil
		}
		return Page{}, err
	}
	return Page{Metadata: adminMetadata(domainDefault(data.Domain.Name, data.Domain.Subdomain)), Content: adminDomainDetail(data)}, nil
}
func (s *site) adminDomainFragment(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.adminFragmentIdentity(w, r); !ok {
		return
	}
	data, err := s.readAdminDomain(r)
	if err != nil {
		status := 503
		var failure accountPageError
		if errors.As(err, &failure) {
			status = failure.Status
		}
		if errors.Is(err, ErrNotFound) {
			status = 404
		}
		http.Error(w, "Could not load this domain", status)
		return
	}
	s.renderFragment(w, r, adminDomainDetail(data))
}
