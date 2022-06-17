package caclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	client   *http.Client
	token    string
	endpoint string
}

func NewCAClient(endpoint string) (*Client, error) {
	return New(endpoint, "", nil)
}

func NewWithRootCA(endpoint string, token string, rootcaFile string) (*Client, error) {
	certPEMBlock, err := os.ReadFile(rootcaFile)
	if err != nil {
		return nil, err
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	pool.AppendCertsFromPEM(certPEMBlock)
	cli := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: pool,
		},
	}}

	return New(endpoint, token, cli)
}

func New(endpoint string, token string, client *http.Client) (*Client, error) {
	u, err := url.Parse(endpoint) // must be a valid endpoint
	if err != nil {
		return nil, err
	}

	if sc := strings.ToUpper(u.Scheme); sc != "HTTP" && sc != "HTTPS" {
		return nil, errors.New("unsupported endpoint scheme")
	}

	if token == "" {
		sat := ServiceAccountToken()
		if sat != "" {
			token = sat
		}
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{
		token:    token,
		endpoint: endpoint,
		client:   client,
	}, nil
}

type CreateCertificateRequest struct {
	CSR string `json:"csr"`
	TTL int64  `json:"ttl"`
}

func (c *CreateCertificateRequest) Payload() *bytes.Buffer {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.Encode(c)
	return buf
}

func (c *Client) CreateCertificate(ctx context.Context, csr []byte, ttl time.Duration) (certChanPem string, rootca string, err error) {
	req, err := http.NewRequest("POST",
		c.endpoint+"/security/v1/sign_certificate",
		(&CreateCertificateRequest{
			CSR: string(csr),
			TTL: ttl.Milliseconds() / 1000,
		}).Payload())
	if err != nil {
		return "", "", err
	}
	// oauth2 token style
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	dec := json.NewDecoder(resp.Body)
	ccr := &CreateCertificateResponse{}

	err = dec.Decode(ccr)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", errors.New(ccr.Message)
	}
	return ccr.CertChain, ccr.RootCert, nil
}

type CreateCertificateResponse struct {
	CertChain string `json:"cert_chain"`
	RootCert  string `json:"root_cert"`
	// if signing failed , error message will be wrapped in `Message`
	Message string `json:"msg"`
}
