package manager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"time"

	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/security/mtls/certificate"
)

type manager struct {
	namespace      string
	serviceAccount string
	bits           int
	ttl            time.Duration
	cli            CSRClient
}

func NewManager(namespace string,
	serviceAccount string,
	bits int,
	ttl time.Duration,
	cli CSRClient,
) Manager {
	return &manager{
		namespace:      namespace,
		serviceAccount: serviceAccount,
		bits:           bits,
		ttl:            ttl,
		cli:            cli,
	}
}

const (
	PemCertificateType   = "CERTIFICATE"
	PemPrivateKeyType    = "PRIVATE KEY"
	PemRSAPrivateKeyType = "RSA PRIVATE KEY"
)

const (
	SchemeSPIFFE = "spiffe"
	TrustDomain  = "cluster.local"
)

func (m *manager) GetBundle(ctx context.Context) (*certificate.Bundle, error) {
	priv, err := rsa.GenerateKey(rand.Reader, m.bits)
	if err != nil {
		log.Errorf("rsa generate key failed: %s", err)
		return nil, err
	}
	// generate CSR using spiffe style
	csr, err := certificate.GenerateCSR(m.namespace, m.serviceAccount, priv)
	if err != nil {
		return nil, err
	}
	log.Infof("generate CSR for %s/%s", m.namespace, m.serviceAccount)

	chain, rootCA, err := m.cli.CreateCertificate(ctx, csr, m.ttl)
	if err != nil {
		log.Errorf("signed certificate failed: %s", err)
		return nil, err
	}
	log.Infof("signed certificate for %s/%s", m.namespace, m.serviceAccount)

	// the last cert in the chain must be the rootCA

	// rsa key should always got a success result
	PKCS8Priv, _ := x509.MarshalPKCS8PrivateKey(priv)

	return &certificate.Bundle{
		ROOTCA:    []byte(rootCA),
		CertChain: []byte(chain),
		PrivKey:   pem.EncodeToMemory(&pem.Block{Type: PemPrivateKeyType, Bytes: PKCS8Priv}),
	}, nil
}
