package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHTTPPublisherSendsSignedEvent(t *testing.T) {
	var signature string
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature = r.Header.Get("X-LCP-Signature")
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	publisher := NewHTTPPublisher([]string{server.URL}, "secret")
	err := publisher.Publish(context.Background(), Event{
		Type:      EventLicenseCreated,
		CreatedAt: time.Unix(1, 0).UTC(),
		Data:      map[string]string{"id": "lic1"},
	})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if signature == "" {
		t.Fatal("expected signature header")
	}
	if len(body) == 0 {
		t.Fatal("expected request body")
	}
}

func TestHTTPPublisherRetriesAndRecordsFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	recorder := &MemoryFailureRecorder{}
	publisher := NewHTTPPublisherWithOptions([]string{server.URL}, "", 3, 0, recorder)
	err := publisher.Publish(context.Background(), Event{Type: EventLicenseCreated, CreatedAt: time.Now().UTC()})
	if err == nil {
		t.Fatal("expected webhook delivery to fail")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	failures := recorder.List()
	if len(failures) != 1 {
		t.Fatalf("expected 1 failure record, got %d", len(failures))
	}
	if failures[0].Attempts != 3 || failures[0].EventType != EventLicenseCreated {
		t.Fatalf("unexpected failure record: %+v", failures[0])
	}
}

func TestHTTPPublisherUsesContextURLs(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	publisher := NewHTTPPublisherWithOptions(nil, "", 1, 0, nil).(*HTTPPublisher).
		WithURLResolver(func(context.Context) []string { return []string{server.URL} })
	if err := publisher.Publish(context.Background(), Event{Type: EventPublicationUploaded}); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if !called {
		t.Fatal("expected tenant webhook target to be called")
	}
}

func TestPersistentFailureRecorderWritesDeadLetterFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webhook-failures.json")
	recorder := NewPersistentFailureRecorder(path)
	if err := recorder.Record(context.Background(), Failure{
		URL:       "https://example.test/webhook",
		EventType: EventPublicationUploaded,
		Attempts:  2,
		Error:     "boom",
		CreatedAt: time.Unix(2, 0).UTC(),
	}); err != nil {
		t.Fatalf("Record failed: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected dead-letter contents")
	}
}
