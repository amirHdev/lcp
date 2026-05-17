package rest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/auth"
	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
	"github.com/amirhdev/ebook-lcp-server/internal/pkg/id"
	usecasePublication "github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/publication"
)

type PublicationHandler struct {
	repo   lcp.PublicationRepository
	upload usecasePublication.PublicationUsecase
}

type PublicationRequest struct {
	Title               string   `json:"title"`
	Authors             []string `json:"authors"`
	Language            string   `json:"language"`
	Subjects            []string `json:"subjects"`
	Tags                []string `json:"tags"`
	RightPrint          *int     `json:"right_print"`
	RightCopy           *int     `json:"right_copy"`
	EncryptedURI        string   `json:"encrypted_uri"`
	Checksum            string   `json:"checksum"`
	LicenseDurationDays int      `json:"license_duration_days"`
	File                string   `json:"file"`
	Status              string   `json:"status"`
}

type PublicationPatchRequest struct {
	Title               *string  `json:"title,omitempty"`
	Authors             []string `json:"authors,omitempty"`
	Language            *string  `json:"language,omitempty"`
	Subjects            []string `json:"subjects,omitempty"`
	Tags                []string `json:"tags,omitempty"`
	RightPrint          *int     `json:"right_print,omitempty"`
	RightCopy           *int     `json:"right_copy,omitempty"`
	EncryptedURI        *string  `json:"encrypted_uri,omitempty"`
	Checksum            *string  `json:"checksum,omitempty"`
	LicenseDurationDays *int     `json:"license_duration_days,omitempty"`
	Status              *string  `json:"status,omitempty"`
}

func NewPublicationHandler(repo lcp.PublicationRepository, upload usecasePublication.PublicationUsecase) *PublicationHandler {
	return &PublicationHandler{repo: repo, upload: upload}
}

func (h *PublicationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "publications" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	switch {
	case len(parts) == 3:
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.requirePublisherOrAdmin(w, r, h.create)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	case len(parts) == 4:
		switch r.Method {
		case http.MethodGet:
			h.get(w, r, parts[3])
		case http.MethodPatch:
			h.requirePublisherOrAdmin(w, r, func(w http.ResponseWriter, r *http.Request) { h.patch(w, r, parts[3]) })
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	case len(parts) == 5 && parts[4] == "activate" && r.Method == http.MethodPost:
		h.requirePublisherOrAdmin(w, r, func(w http.ResponseWriter, r *http.Request) { h.setStatus(w, r, parts[3], "active") })
		return
	case len(parts) == 5 && parts[4] == "deactivate" && r.Method == http.MethodPost:
		h.requirePublisherOrAdmin(w, r, func(w http.ResponseWriter, r *http.Request) { h.setStatus(w, r, parts[3], "inactive") })
		return
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (h *PublicationHandler) requirePublisherOrAdmin(w http.ResponseWriter, r *http.Request, next func(http.ResponseWriter, *http.Request)) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || (!claimsHasRole(claims, "admin") && !claimsHasRole(claims, "publisher")) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "publisher or admin role required"})
		return
	}
	next(w, r)
}

func (h *PublicationHandler) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	pubs, err := h.repo.FindAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"publications": pubs})
}

func (h *PublicationHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	pub, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if pub == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "publication not found"})
		return
	}
	writeJSON(w, http.StatusOK, pub)
}

func (h *PublicationHandler) create(w http.ResponseWriter, r *http.Request) {
	var req PublicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	if err := validatePublicationRights(req.RightPrint, req.RightCopy); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	pub := &lcp.Publication{
		Title:               req.Title,
		Authors:             req.Authors,
		Language:            req.Language,
		Subjects:            req.Subjects,
		Tags:                req.Tags,
		Status:              defaultPublicationStatus(req.Status),
		RightPrint:          req.RightPrint,
		RightCopy:           req.RightCopy,
		EncryptedURI:        req.EncryptedURI,
		Checksum:            req.Checksum,
		LicenseDurationDays: req.LicenseDurationDays,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if req.File != "" {
		raw, err := base64.StdEncoding.DecodeString(req.File)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file must be base64 encoded"})
			return
		}
		processed, err := h.upload.UploadAndEncrypt(r.Context(), req.Title, bytes.NewReader(raw))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		pub.ID = processed.ID
		pub.FilePath = processed.FilePath
		pub.EncryptedPath = processed.EncryptedPath
		pub.EncryptedURI = processed.EncryptedURI
		if pub.Status == "" {
			pub.Status = "active"
		}
	} else if req.EncryptedURI == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file or encrypted_uri is required"})
		return
	}

	if pub.ID == "" {
		pub.ID = id.New()
	}
	if pub.EncryptedURI == "" {
		pub.EncryptedURI = pub.EncryptedPath
	}
	if pub.LicenseDurationDays == 0 {
		pub.LicenseDurationDays = 30
	}

	if err := h.repo.Save(r.Context(), pub); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, pub)
}

func (h *PublicationHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	pub, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if pub == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "publication not found"})
		return
	}

	var req PublicationPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if err := validatePublicationRights(req.RightPrint, req.RightCopy); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if req.Title != nil {
		pub.Title = *req.Title
	}
	if req.Language != nil {
		pub.Language = *req.Language
	}
	if req.EncryptedURI != nil {
		pub.EncryptedURI = *req.EncryptedURI
	}
	if req.Checksum != nil {
		pub.Checksum = *req.Checksum
	}
	if req.LicenseDurationDays != nil {
		pub.LicenseDurationDays = *req.LicenseDurationDays
	}
	if req.Status != nil {
		pub.Status = *req.Status
	}
	if req.Authors != nil {
		pub.Authors = req.Authors
	}
	if req.Subjects != nil {
		pub.Subjects = req.Subjects
	}
	if req.Tags != nil {
		pub.Tags = req.Tags
	}
	if req.RightPrint != nil {
		pub.RightPrint = req.RightPrint
	}
	if req.RightCopy != nil {
		pub.RightCopy = req.RightCopy
	}

	pub.UpdatedAt = time.Now()
	if err := h.repo.Save(r.Context(), pub); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, pub)
}

func (h *PublicationHandler) setStatus(w http.ResponseWriter, r *http.Request, id, status string) {
	pub, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if pub == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "publication not found"})
		return
	}
	pub.Status = status
	pub.UpdatedAt = time.Now()
	if err := h.repo.Save(r.Context(), pub); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, pub)
}

func defaultPublicationStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return "active"
	}
	return strings.TrimSpace(strings.ToLower(status))
}

func validatePublicationRights(rightPrint, rightCopy *int) error {
	if rightPrint != nil && *rightPrint < 0 {
		return fmt.Errorf("right_print must be zero or positive")
	}
	if rightCopy != nil && *rightCopy < 0 {
		return fmt.Errorf("right_copy must be zero or positive")
	}
	return nil
}

func claimsHasRole(claims *auth.Claims, role string) bool {
	role = strings.ToLower(role)
	if claims == nil {
		return false
	}
	if strings.ToLower(claims.Role) == role {
		return true
	}
	for _, candidate := range claims.Roles {
		if strings.ToLower(candidate) == role {
			return true
		}
	}
	return false
}
