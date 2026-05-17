package graphql

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

// NewHandler wires a lightweight GraphQL-compatible endpoint without external dependencies.
// The handler supports the repository's defined operations and expects uploads as base64 strings.
func NewHandler(resolver *Resolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Send a POST {query, variables} payload to interact with the LCP GraphQL API.",
			})
			return
		}

		payload, err := DecodePayload(r)
		if err != nil {
			writeGraphQLError(w, err)
			return
		}

		query := strings.ToLower(payload.Query)
		switch {
		case strings.Contains(query, "uploadpublication"):
			handleUploadPublication(w, r, resolver, payload)
		case strings.Contains(query, "createlicense"):
			handleCreateLicense(w, r, resolver, payload)
		case strings.Contains(query, "revokelicense"):
			handleRevokeLicense(w, r, resolver, payload)
		case strings.Contains(query, "licenses"):
			handleListLicenses(w, r, resolver, payload)
		case strings.Contains(query, "publications"):
			handleListPublications(w, r, resolver)
		default:
			writeGraphQLError(w, ErrUnsupportedOperation)
		}
	})
}

func handleUploadPublication(w http.ResponseWriter, r *http.Request, resolver *Resolver, payload *GraphQLPayload) {
	title := stringValue(payload.Variables["title"])
	rawFile := payload.Variables["file"]
	if title == "" || rawFile == nil {
		writeGraphQLError(w, ErrMissingFields)
		return
	}

	data, err := readUploadBytes(rawFile)
	if err != nil {
		writeGraphQLError(w, err)
		return
	}

	pub, err := resolver.PublicationUsecase.UploadAndEncrypt(r.Context(), title, bytes.NewReader(data))
	if err != nil {
		writeGraphQLError(w, err)
		return
	}

	writeGraphQLData(w, map[string]interface{}{
		"uploadPublication": encodePublication(pub, resolver.PublicBaseURL),
	})
}

func handleCreateLicense(w http.ResponseWriter, r *http.Request, resolver *Resolver, payload *GraphQLPayload) {
	startDate, err := parseTimePtr(stringPtr(payload.Variables["startDate"]))
	if err != nil {
		writeGraphQLError(w, err)
		return
	}
	endDate, err := parseTimePtr(stringPtr(payload.Variables["endDate"]))
	if err != nil {
		writeGraphQLError(w, err)
		return
	}

	license, err := resolver.LicenseUsecase.Create(r.Context(), &lcp.LicenseInput{
		PublicationID: stringValue(payload.Variables["publicationID"]),
		UserID:        stringValue(payload.Variables["userID"]),
		Passphrase:    stringValue(payload.Variables["passphrase"]),
		Hint:          stringValue(payload.Variables["hint"]),
		RightPrint:    intPtr(payload.Variables["rightPrint"]),
		RightCopy:     intPtr(payload.Variables["rightCopy"]),
		StartDate:     startDate,
		EndDate:       endDate,
	})
	if err != nil {
		writeGraphQLError(w, err)
		return
	}

	writeGraphQLData(w, map[string]interface{}{
		"createLicense": encodeLicense(license),
	})
}

func handleRevokeLicense(w http.ResponseWriter, r *http.Request, resolver *Resolver, payload *GraphQLPayload) {
	id := stringValue(payload.Variables["id"])
	if id == "" {
		writeGraphQLError(w, ErrMissingFields)
		return
	}

	if err := resolver.LicenseUsecase.Revoke(r.Context(), id); err != nil {
		writeGraphQLError(w, err)
		return
	}

	writeGraphQLData(w, map[string]interface{}{
		"revokeLicense": true,
	})
}

func handleListLicenses(w http.ResponseWriter, r *http.Request, resolver *Resolver, payload *GraphQLPayload) {
	var publicationID *string
	if val := stringPtr(payload.Variables["publicationID"]); val != nil {
		publicationID = val
	}

	licenses, err := resolver.LicenseUsecase.GetByPublication(r.Context(), publicationID)
	if err != nil {
		writeGraphQLError(w, err)
		return
	}

	encoded := make([]map[string]interface{}, 0, len(licenses))
	for _, lic := range licenses {
		encoded = append(encoded, encodeLicense(lic))
	}

	writeGraphQLData(w, map[string]interface{}{
		"licenses": encoded,
	})
}

func handleListPublications(w http.ResponseWriter, r *http.Request, resolver *Resolver) {
	pubs, err := resolver.PublicationUsecase.GetAll(r.Context())
	if err != nil {
		writeGraphQLError(w, err)
		return
	}

	encoded := make([]map[string]interface{}, 0, len(pubs))
	for _, pub := range pubs {
		encoded = append(encoded, encodePublication(pub, resolver.PublicBaseURL))
	}

	writeGraphQLData(w, map[string]interface{}{
		"publications": encoded,
	})
}

func writeGraphQLData(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": data})
}

func writeGraphQLError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": []map[string]interface{}{{"message": err.Error()}},
	})
}

func encodePublication(pub *lcp.Publication, baseURL string) map[string]interface{} {
	return map[string]interface{}{
		"id":            pub.ID,
		"title":         pub.Title,
		"rightPrint":    pub.RightPrint,
		"rightCopy":     pub.RightCopy,
		"filePath":      pub.FilePath,
		"encryptedPath": pub.EncryptedPath,
		"createdAt":     pub.CreatedAt.Format(time.RFC3339),
		"downloadURL":   strings.TrimRight(baseURL, "/") + "/publications/" + pub.ID + "/content",
	}
}

func encodeLicense(license *lcp.License) map[string]interface{} {
	var startDate, endDate *string
	if license.StartDate != nil {
		formatted := license.StartDate.Format(time.RFC3339)
		startDate = &formatted
	}
	if license.EndDate != nil {
		formatted := license.EndDate.Format(time.RFC3339)
		endDate = &formatted
	}

	return map[string]interface{}{
		"id":             license.ID,
		"publicationID":  license.PublicationID,
		"userID":         license.UserID,
		"passphrase":     license.Passphrase,
		"hint":           license.Hint,
		"publicationURL": license.PublicationURL,
		"rightPrint":     license.RightPrint,
		"rightCopy":      license.RightCopy,
		"startDate":      startDate,
		"endDate":        endDate,
		"createdAt":      license.CreatedAt.Format(time.RFC3339),
	}
}

func readUploadBytes(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		if decoded, err := base64.StdEncoding.DecodeString(v); err == nil {
			return decoded, nil
		}
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return nil, ErrUnsupportedFile
	}
}

func stringValue(value interface{}) string {
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}

func stringPtr(value interface{}) *string {
	if v, ok := value.(string); ok {
		return &v
	}
	return nil
}

func intPtr(value interface{}) *int {
	switch v := value.(type) {
	case int:
		return &v
	case int64:
		i := int(v)
		return &i
	case float64:
		i := int(v)
		return &i
	default:
		return nil
	}
}

func parseTimePtr(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// GraphQLPayload describes the request body accepted by the handler.
type GraphQLPayload struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// DecodePayload parses the HTTP request body into a GraphQLPayload.
func DecodePayload(r *http.Request) (*GraphQLPayload, error) {
	var payload GraphQLPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// GraphQL handler errors.
var (
	ErrUnsupportedOperation = errors.New("operation not supported by this handler")
	ErrMissingFields        = errors.New("required fields are missing in variables")
	ErrUnsupportedFile      = errors.New("file must be provided as a base64 string")
)
