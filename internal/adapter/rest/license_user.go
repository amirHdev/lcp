package rest

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	usecaseLicense "github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/license"
)

type userDataPayload struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	Email          string `json:"email,omitempty"`
	PassphraseHash string `json:"passphrasehash"`
	Hint           string `json:"hint,omitempty"`
}

func LicenseUserData(licenses usecaseLicense.LicenseUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		licenseID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/licenses/"), "/user")
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

		payload := userDataPayload{
			ID:             lic.UserID,
			PassphraseHash: lcpPassphraseHash(lic.Passphrase),
			Hint:           lic.Hint,
		}

		writeJSON(w, http.StatusOK, payload)
	}
}

// lcpPassphraseHash returns the Readium LCP user key hash.
// LCP requires SHA-256 here for interoperability with LCP-compliant readers.
// This is not an account-password storage hash.
func lcpPassphraseHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
