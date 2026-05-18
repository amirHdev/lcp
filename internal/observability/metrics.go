package observability

import "sync/atomic"

type Snapshot struct {
	WebhookOK       int64
	WebhookFailed   int64
	S3StoreOK       int64
	S3StoreFailed   int64
	S3OpenOK        int64
	S3OpenFailed    int64
	S3SignedURLOK   int64
	S3SignedURLFail int64
	LicensesOK      int64
	LicensesFailed  int64
	AuthFailed      int64
}

var (
	webhookOK       atomic.Int64
	webhookFailed   atomic.Int64
	s3StoreOK       atomic.Int64
	s3StoreFailed   atomic.Int64
	s3OpenOK        atomic.Int64
	s3OpenFailed    atomic.Int64
	s3SignedURLOK   atomic.Int64
	s3SignedURLFail atomic.Int64
	licensesOK      atomic.Int64
	licensesFailed  atomic.Int64
	authFailed      atomic.Int64
)

func IncWebhookOK()       { webhookOK.Add(1) }
func IncWebhookFailed()   { webhookFailed.Add(1) }
func IncS3StoreOK()       { s3StoreOK.Add(1) }
func IncS3StoreFailed()   { s3StoreFailed.Add(1) }
func IncS3OpenOK()        { s3OpenOK.Add(1) }
func IncS3OpenFailed()    { s3OpenFailed.Add(1) }
func IncS3SignedURLOK()   { s3SignedURLOK.Add(1) }
func IncS3SignedURLFail() { s3SignedURLFail.Add(1) }
func IncLicensesOK()      { licensesOK.Add(1) }
func IncLicensesFailed()  { licensesFailed.Add(1) }
func IncAuthFailed()      { authFailed.Add(1) }

func Current() Snapshot {
	return Snapshot{
		WebhookOK:       webhookOK.Load(),
		WebhookFailed:   webhookFailed.Load(),
		S3StoreOK:       s3StoreOK.Load(),
		S3StoreFailed:   s3StoreFailed.Load(),
		S3OpenOK:        s3OpenOK.Load(),
		S3OpenFailed:    s3OpenFailed.Load(),
		S3SignedURLOK:   s3SignedURLOK.Load(),
		S3SignedURLFail: s3SignedURLFail.Load(),
		LicensesOK:      licensesOK.Load(),
		LicensesFailed:  licensesFailed.Load(),
		AuthFailed:      authFailed.Load(),
	}
}
