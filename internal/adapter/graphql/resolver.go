package graphql

import (
	usecaseLicense "github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/license"
	usecasePublication "github.com/amirhdev/ebook-lcp-server/internal/usecase/lcp/publication"
)

// Resolver aggregates the use cases needed by the GraphQL layer.
type Resolver struct {
	PublicationUsecase usecasePublication.PublicationUsecase
	LicenseUsecase     usecaseLicense.LicenseUsecase
	PublicBaseURL      string
}
