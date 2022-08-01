package agent

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Option struct {
	// Network and Address are used to created a listener which sds will serve on.
	// See also: https://pkg.go.dev/net#Listen
	// Default Network: "unix"
	Network string

	// Default Address: "/var/run/polaris/mtls/sds.sock"
	Address string

	// CryptombPollDelay is a parameter of cryptomb
	// See also: https://www.envoyproxy.io/docs/envoy/v1.22.2/api-v3/extensions/private_key_providers/cryptomb/v3alpha/cryptomb.proto.html
	// Default is 0.2 ms.
	CryptombPollDelay time.Duration

	// RotatePeriod is the period of the rotation.
	// Default is 30 minute.
	RotatePeriod time.Duration

	// FailedRetryDelay It defines the retry interval when the rotation action failed.
	// Default is 1 second.
	FailedRetryDelay time.Duration

	// Namespace is the current namespace.
	Namespace string

	// ServiceAccount is the current ServiceAccount name.
	// See also: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	ServiceAccount string

	// CAServer is the address of the CA server.
	CAServer string

	// RSAKeyBits is the bit size of the RSA key.
	RSAKeyBits int

	// TTL of the certificate
	// Default is 1 hour.
	TTL time.Duration
}

func EnvDefaultDuration(name string, val time.Duration, def time.Duration) time.Duration {
	if val != 0 {
		return val
	}
	if d, err := time.ParseDuration(os.Getenv(name)); err == nil {
		return d
	}
	return def
}

func EnvDefaultInt(name string, val int, def int) int {
	if val != 0 {
		return val
	}
	if d, err := strconv.Atoi(os.Getenv(name)); err == nil {
		return d
	}
	return def
}

const DefaultSDSAddress = "/var/run/polaris/mtls/sds.sock"

// init options with enviroment variables
func (opt *Option) init() error {
	if opt.Network == "" {
		opt.Network = "unix"
	}
	if opt.Address == "" {
		opt.Address = DefaultSDSAddress
	}

	opt.CryptombPollDelay = EnvDefaultDuration("POLARIS_SIDECAR_MTLS_CRYPTO_MB_POLL_DELAY",
		opt.CryptombPollDelay,
		time.Microsecond*200)

	opt.RotatePeriod = EnvDefaultDuration("POLARIS_SIDECAR_MTLS_ROTATE_PERIOD",
		opt.RotatePeriod,
		time.Hour)

	opt.FailedRetryDelay = EnvDefaultDuration("POLARIS_SIDECAR_MTLS_ROTATE_FAILED_RETRY_DELAY",
		opt.FailedRetryDelay,
		time.Second)

	if opt.Namespace == "" || opt.ServiceAccount == "" {
		if envNS := os.Getenv("KUBERNETES_NAMESPACE"); envNS != "" {
			opt.Namespace = envNS
		}
		if envSA := os.Getenv("KUBERNETES_SERVICE_ACCOUNT"); envSA != "" {
			opt.ServiceAccount = envSA
		}
		if opt.Namespace == "" || opt.ServiceAccount == "" {
			sa, err := loadServiceAccount()
			if err != nil {
				return fmt.Errorf("cannot detect which namespace and serviceaccount being used :%w", err)
			}
			opt.Namespace = sa.Namespace
			opt.ServiceAccount = sa.AccountName
		}
	}
	if opt.CAServer == "" {
		ca := os.Getenv("POLARIS_SIDECAR_MTLS_CA_SERVER")
		if ca == "" {
			return errors.New("no ca server endpoint provided")
		}
		opt.CAServer = ca
	}

	lca := strings.ToLower(opt.CAServer)

	if !strings.HasPrefix(lca, "http://") && !strings.HasPrefix(lca, "https://") {
		// add scheme to endpoint
		opt.CAServer = "http://" + opt.CAServer
	}

	opt.RSAKeyBits = EnvDefaultInt("POLARIS_SIDECAR_MTLS_KEY_BITS",
		opt.RSAKeyBits, 2048)

	opt.TTL = EnvDefaultDuration("POLARIS_SIDECAR_MTLS_CERT_TTL",
		opt.TTL, time.Hour)

	return nil
}
