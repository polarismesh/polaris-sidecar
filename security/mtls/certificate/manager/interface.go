package manager

import (
	"context"
	"time"

	"github.com/polarismesh/polaris-sidecar/security/mtls/certificate"
)

type Manager interface {
	GetBundle(ctx context.Context) (bundle *certificate.Bundle, err error)
}

type CSRClient interface {
	CreateCertificate(ctx context.Context, csr []byte, ttl time.Duration) (certChanPem string, rootca string, err error)
}
