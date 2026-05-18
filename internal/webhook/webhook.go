package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/amirhdev/ebook-lcp-server/internal/observability"
)

const (
	EventPublicationUploaded = "publication.uploaded"
	EventLicenseCreated      = "license.created"
	EventLicenseRevoked      = "license.revoked"
)

type Event struct {
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Data      any       `json:"data"`
}

type Failure struct {
	URL       string    `json:"url"`
	EventType string    `json:"eventType"`
	Attempts  int       `json:"attempts"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"createdAt"`
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

type NopPublisher struct{}

func (NopPublisher) Publish(context.Context, Event) error {
	return nil
}

type HTTPPublisher struct {
	urls        []string
	secret      string
	client      *http.Client
	maxAttempts int
	backoff     time.Duration
	failures    FailureRecorder
	urlsFor     func(context.Context) []string
}

type FailureRecorder interface {
	Record(context.Context, Failure) error
}

type MemoryFailureRecorder struct {
	mu       sync.Mutex
	failures []Failure
}

func (r *MemoryFailureRecorder) Record(_ context.Context, failure Failure) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failures = append(r.failures, failure)
	return nil
}

func (r *MemoryFailureRecorder) List() []Failure {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]Failure, len(r.failures))
	copy(items, r.failures)
	return items
}

type PersistentFailureRecorder struct {
	path string
	mu   sync.Mutex
}

func NewPersistentFailureRecorder(path string) *PersistentFailureRecorder {
	return &PersistentFailureRecorder{path: path}
}

func (r *PersistentFailureRecorder) Record(_ context.Context, failure Failure) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var failures []Failure
	if raw, err := os.ReadFile(r.path); err == nil {
		if err := json.Unmarshal(raw, &failures); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	failures = append(failures, failure)
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(failures, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, raw, 0o600)
}

func NewHTTPPublisher(urls []string, secret string) Publisher {
	return NewHTTPPublisherWithOptions(urls, secret, 3, 250*time.Millisecond, nil)
}

func NewHTTPPublisherWithOptions(urls []string, secret string, maxAttempts int, backoff time.Duration, failures FailureRecorder) Publisher {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if backoff < 0 {
		backoff = 0
	}
	return &HTTPPublisher{
		urls:        append([]string(nil), urls...),
		secret:      secret,
		client:      &http.Client{Timeout: 5 * time.Second},
		maxAttempts: maxAttempts,
		backoff:     backoff,
		failures:    failures,
	}
}

func (p *HTTPPublisher) WithURLResolver(resolver func(context.Context) []string) *HTTPPublisher {
	p.urlsFor = resolver
	return p
}

func (p *HTTPPublisher) Publish(ctx context.Context, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	targets := p.urls
	if p.urlsFor != nil {
		if tenantTargets := p.urlsFor(ctx); len(tenantTargets) > 0 {
			targets = tenantTargets
		}
	}
	for _, target := range targets {
		var deliveryErr error
		for attempt := 1; attempt <= p.maxAttempts; attempt++ {
			deliveryErr = p.deliver(ctx, target, body)
			if deliveryErr == nil {
				break
			}
			if attempt < p.maxAttempts && p.backoff > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(attempt) * p.backoff):
				}
			}
		}
		if deliveryErr != nil {
			observability.IncWebhookFailed()
			if p.failures != nil {
				_ = p.failures.Record(ctx, Failure{
					URL:       target,
					EventType: event.Type,
					Attempts:  p.maxAttempts,
					Error:     deliveryErr.Error(),
					CreatedAt: time.Now().UTC(),
				})
			}
			return deliveryErr
		}
		observability.IncWebhookOK()
	}
	return nil
}

func (p *HTTPPublisher) deliver(ctx context.Context, target string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.secret != "" {
		req.Header.Set("X-LCP-Signature", sign(body, p.secret))
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	if err := resp.Body.Close(); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
