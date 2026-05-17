package lcp

import (
	"context"
	"testing"

	domain "github.com/amirhdev/ebook-lcp-server/internal/domain/lcp"
)

func TestPublicationRepositorySaveAndFind(t *testing.T) {
	repo := NewPublicationRepository()

	pub := &domain.Publication{
		ID:    "pub1",
		Title: "Test",
	}

	err := repo.Save(context.Background(), pub)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	found, err := repo.FindByID(context.Background(), "pub1")
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}

	if found == nil {
		t.Fatal("expected publication")
	}

	if found.Title != "Test" {
		t.Fatalf("unexpected title: %s", found.Title)
	}
}

func TestPublicationRepositoryFindAll(t *testing.T) {
	repo := NewPublicationRepository()

	_ = repo.Save(context.Background(), &domain.Publication{
		ID:    "1",
		Title: "A",
	})

	pubs, err := repo.FindAll(context.Background())
	if err != nil {
		t.Fatalf("find all failed: %v", err)
	}

	if len(pubs) != 1 {
		t.Fatalf("expected 1 publication, got %d", len(pubs))
	}
}
