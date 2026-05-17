package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	usecaseLicense "github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/license"
)

type UpdatedField struct {
	License string `json:"license"`
	Status  string `json:"status"`
}

type LicenseStatusLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
	Type string `json:"type,omitempty"`
}

type LicenseStatusDocumentResponse struct {
	ID      string              `json:"id"`
	Status  string              `json:"status"`
	Message string              `json:"message,omitempty"`
	Updated UpdatedField        `json:"updated"`
	Links   []LicenseStatusLink `json:"links"`
}

func LicenseStatusDocument(licenses usecaseLicense.LicenseUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		licenseID := strings.TrimPrefix(r.URL.Path, "/licenses/")
		licenseID = strings.TrimPrefix(licenseID, "api/v1/licenses/")
		licenseID = strings.TrimSuffix(licenseID, "/status")
		licenseID = strings.Trim(licenseID, "/")

		if licenseID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "license id is required"})
			return
		}

		lic, err := licenses.GetByID(r.Context(), licenseID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if lic == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
			return
		}

		scheme := "https"
		if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
			scheme = forwarded
		}

		host := r.Host
		self := scheme + "://" + host + "/licenses/" + lic.ID + "/status"
		licenseLink := "https://yourdomain.com/licenses/" + lic.ID + ".lcpl"

		w.Header().Set("Content-Type", "application/vnd.readium.license.status.v1.0+json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(LicenseStatusDocumentResponse{
			ID:      lic.ID,
			Status:  "ready",
			Message: "License is ready.",
			Updated: UpdatedField{
				License: lic.ID,
				Status:  "ready",
			},
			Links: []LicenseStatusLink{
				{
					Rel:  "self",
					Href: self,
					Type: "application/vnd.readium.license.status.v1.0+json",
				},
				{
					Rel:  "license",
					Href: licenseLink,
					Type: "application/vnd.readium.lcp.license.v1.0+json",
				},
			},
		})
	}
}
