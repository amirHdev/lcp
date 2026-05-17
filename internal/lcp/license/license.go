package license

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

var ErrContentNotFound = fmt.Errorf("lcp core content not found")

type Service struct {
	coreURL     string
	coreUser    string
	corePass    string
	statusURL   string
	statusUser  string
	statusPass  string
	providerURI string
	httpClient  *http.Client
}

func NewService(coreURL, coreUser, corePass, statusURL, statusUser, statusPass, providerURI string) *Service {
	return &Service{
		coreURL:     strings.TrimRight(strings.TrimSpace(coreURL), "/"),
		coreUser:    coreUser,
		corePass:    corePass,
		statusURL:   strings.TrimRight(strings.TrimSpace(statusURL), "/"),
		statusUser:  statusUser,
		statusPass:  statusPass,
		providerURI: strings.TrimSpace(providerURI),
		httpClient:  &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *Service) GenerateLicense(ctx context.Context, license *lcp.License) error {
	if license == nil || license.PublicationID == "" || license.UserID == "" {
		return fmt.Errorf("missing publication or user identifiers")
	}
	if s.coreURL == "" {
		return nil
	}

	partial := map[string]any{
		"provider": s.providerURI,
		"user": map[string]any{
			"id":        license.UserID,
			"encrypted": []string{"email", "name"},
		},
		"encryption": map[string]any{
			"user_key": map[string]any{
				"algorithm": "http://www.w3.org/2001/04/xmlenc#sha256",
				"text_hint": license.Hint,
				"hex_value": lcpPassphraseHash(license.Passphrase),
			},
		},
		"rights": map[string]any{},
	}

	if license.RightPrint != nil {
		partial["rights"].(map[string]any)["print"] = license.RightPrint
	}
	if license.RightCopy != nil {
		partial["rights"].(map[string]any)["copy"] = license.RightCopy
	}
	if license.StartDate != nil {
		partial["rights"].(map[string]any)["start"] = license.StartDate.UTC().Truncate(time.Second)
	}
	if license.EndDate != nil {
		partial["rights"].(map[string]any)["end"] = license.EndDate.UTC().Truncate(time.Second)
	}

	body, err := json.Marshal(partial)
	if err != nil {
		return err
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < 5; attempt++ {
		req, reqErr := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			s.coreURL+"/contents/"+license.PublicationID+"/license",
			bytes.NewReader(body),
		)
		if reqErr != nil {
			return reqErr
		}

		req.Header.Set("Content-Type", "application/vnd.readium.lcp.license.v1.0+json")
		if s.coreUser != "" {
			req.SetBasicAuth(s.coreUser, s.corePass)
		}

		resp, lastErr = s.httpClient.Do(req)
		if lastErr == nil {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					log.Printf("close rows: %v", err)
				}
			}()

			if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
				lastErr = nil
			} else if resp.StatusCode == http.StatusNotFound {
				lastErr = ErrContentNotFound
			} else {
				lastErr = fmt.Errorf("lcp core returned %s", resp.Status)
			}
			break
		}

		if !strings.Contains(lastErr.Error(), "connect: connection refused") &&
			!strings.Contains(lastErr.Error(), "connection reset by peer") &&
			!strings.Contains(lastErr.Error(), "i/o timeout") {
			return lastErr
		}

		if attempt < 4 {
			time.Sleep(time.Duration(attempt+1) * 250 * time.Millisecond)
		}
	}

	if lastErr != nil {
		return lastErr
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	license.LCPL = string(bodyBytes)

	var generated struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(bodyBytes, &generated); err == nil && generated.ID != "" {
		license.ID = generated.ID
	}

	return nil
}

func (s *Service) RevokeLicense(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("missing license id")
	}
	if s.statusURL == "" {
		return nil
	}

	payload := map[string]string{"status": "revoked"}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		s.statusURL+"/licenses/"+id+"/status",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/vnd.readium.lcp.license.v1.0+json")
	if s.statusUser != "" {
		req.SetBasicAuth(s.statusUser, s.statusPass)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("close rows: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusNoContent &&
		resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("status server returned %s", resp.Status)
	}

	return nil
}

// lcpPassphraseHash returns the Readium LCP user key hash.
// LCP requires SHA-256 here for interoperability with LCP-compliant readers.
// This is not an account-password storage hash.
func lcpPassphraseHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (s *Service) GetLicense(ctx context.Context, id string) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("missing license id")
	}
	if s.coreURL == "" {
		return nil, fmt.Errorf("missing LCP core url")
	}

	candidates := []string{
		s.coreURL + "/licenses/" + id,
		s.coreURL + "/licenses/" + id + ".lcpl",
	}

	var lastErr error

	for _, url := range candidates {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		if s.coreUser != "" {
			req.SetBasicAuth(s.coreUser, s.corePass)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		lastErr = fmt.Errorf("lcp core returned %s from %s: %s", resp.Status, url, string(body))
	}

	return nil, lastErr
}
