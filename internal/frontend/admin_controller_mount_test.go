package frontend

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// Fragment replacement must leave both component scopes and window actions intact.
func TestAdminEditorsRenderTheirControllerScopes(t *testing.T) {
	for _, entry := range []struct {
		name      string
		component templ.Component
		markers   []string
	}{
		{"announcements", adminAnnouncements(adminView{}), []string{`data-controller="admin-announcements"`, `cancel->admin-announcements#close:capture`, `data-controller="date-time-picker"`}},
		{"blog", adminBlog(adminBlogData{}), []string{`data-controller="admin-blog"`, `cancel->admin-blog#close:capture`, `data-controller="date-time-picker"`}},
		{"campaigns", adminCampaigns(adminCampaignsData{}), []string{`data-controller="admin-campaigns"`, `pagehide@window->admin-campaigns#hide`}},
		{"domains", adminDomains(adminView{}), []string{`data-controller="admin-domains"`, `input->admin-domains#render`}},
		{"domain detail", adminDomainDetail(adminDomainDetailData{Domain: &adminDomain{settingsDomain: settingsDomain{ID: 7}}}), []string{`data-controller="admin-domain-detail"`, `pageshow@window->admin-domain-detail#show`}},
		{"river", adminRiver(adminRiverData{}), []string{`data-controller="admin-river"`, `cancel->admin-river#close:capture`}},
		{"wallet", adminWallet(adminWalletData{}, nil), []string{`data-controller="admin-wallet"`, `pageshow@window->admin-wallet#show`}},
	} {
		t.Run(entry.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := entry.component.Render(context.Background(), &out); err != nil {
				t.Fatal(err)
			}
			for _, marker := range entry.markers {
				if !strings.Contains(out.String(), marker) {
					t.Errorf("missing lifecycle binding %s", marker)
				}
			}
		})
	}
}
