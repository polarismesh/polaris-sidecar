package caclient

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"testing"
	"time"

	"github.com/polarismesh/polaris-sidecar/security/mtls/certificate"
)

func TestCreateCertificate(t *testing.T) {
	rootca := os.Getenv("TEST_CREATE_CERT_ROOTCA")
	sat := os.Getenv("TEST_CREATE_CERT_SAT")
	certEndpoint := os.Getenv("TEST_CREATE_CERT_ENDPOINT")

	if rootca == "" || sat == "" || certEndpoint == "" {
		t.Skip()
		return
	}
	cli, err := NewWithRootCA(certEndpoint, sat, rootca)
	if err != nil {
		t.Fatal(err)
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	csr, _ := certificate.GenerateCSR("default", "default", priv)
	chain, root, err := cli.CreateCertificate(context.TODO(), csr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("cert chain:", chain)
	t.Log("root ca", root)
}
