package certificate

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net/url"
)

type Bundle struct {
	ROOTCA    []byte
	CertChain []byte
	PrivKey   []byte
}

const (
	SchemeSPIFFE = "spiffe"
	TrustDomain  = "cluster.local"
)

func GenerateCSR(ns string, sa string, priv interface{}) (csr []byte, err error) {
	// certificate must meet the SPIFFE document: https://github.com/spiffe/spiffe/blob/main/standards/X509-SVID.md
	tpl := &x509.CertificateRequest{}
	tpl.Subject = pkix.Name{
		Organization: []string{TrustDomain},
	}
	tpl.URIs = append(tpl.URIs, &url.URL{
		Scheme: SchemeSPIFFE,
		Host:   TrustDomain,
		Path:   fmt.Sprintf("/ns/%s/sa/%s", ns, sa),
	})

	csr, err = x509.CreateCertificateRequest(rand.Reader, tpl, priv)
	if err != nil {
		return nil, err
	}

	csr = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
	return csr, nil
}
