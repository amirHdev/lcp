package rest

import (
	"context"
	"net/http"
	"strings"

	usecaseLicense "github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/license"
)

type LCPLProvider interface {
	GetLicense(ctx context.Context, id string) ([]byte, error)
}

type LicenseDownloadHandler struct {
	licenses usecaseLicense.LicenseUsecase
	lcp      LCPLProvider
}

func NewLicenseDownloadHandler(
	licenses usecaseLicense.LicenseUsecase,
	lcp LCPLProvider,
) *LicenseDownloadHandler {
	return &LicenseDownloadHandler{
		licenses: licenses,
		lcp:      lcp,
	}
}

func (h *LicenseDownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	licenseID, ok := extractLicenseIDForLCPL(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	lic, err := h.licenses.GetByID(r.Context(), licenseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if lic == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
		return
	}

	lcpl := []byte(lic.LCPL)
	if len(lcpl) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "license document not found"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.readium.lcp.license.v1.0+json")
	w.Header().Set("Content-Disposition", `attachment; filename="`+licenseID+`.lcpl"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(lcpl)
}

func extractLicenseIDForLCPL(path string) (string, bool) {
	path = strings.Trim(path, "/")

	if strings.HasPrefix(path, "licenses/") && strings.HasSuffix(path, ".lcpl") {
		id := strings.TrimSuffix(strings.TrimPrefix(path, "licenses/"), ".lcpl")
		id = strings.Trim(id, "/")
		return id, id != ""
	}

	if strings.HasPrefix(path, "api/v1/licenses/") && strings.HasSuffix(path, "/lcpl") {
		id := strings.TrimSuffix(strings.TrimPrefix(path, "api/v1/licenses/"), "/lcpl")
		id = strings.Trim(id, "/")
		return id, id != ""
	}

	return "", false
}
