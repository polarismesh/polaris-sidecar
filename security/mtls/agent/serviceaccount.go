package agent

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type ServiceAccount struct {
	Namespace   string `json:"kubernetes.io/serviceaccount/namespace"`
	AccountName string `json:"kubernetes.io/serviceaccount/service-account.name"`
}

const DefaultJWTPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

// loadServiceAccount load service account information from token
func loadServiceAccount() (*ServiceAccount, error) {
	tokenFile := DefaultJWTPath
	if _, err := os.Stat(tokenFile); err != nil {
		return nil, errors.New("no service account token file")
	}
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read service account token: %w", err)
	}
	parts := strings.Split(string(data), ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("unknown token format")
	}
	jsonStr, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("unknown token base64 format")
	}
	sa := &ServiceAccount{}
	err = json.Unmarshal(jsonStr, sa)
	if err != nil {
		return nil, fmt.Errorf("unknown token payload json format")
	}
	return sa, nil
}
