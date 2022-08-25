package sds

import (
	"time"

	cryptomb "github.com/envoyproxy/go-control-plane/contrib/envoy/extensions/private_key_providers/cryptomb/v3alpha"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoytls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/intel-go/cpuid"
	"github.com/polarismesh/polaris-sidecar/security/mtls/certificate"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// cryptombSupported indicate that whether the cpu can use crypto_mb library.
// The crypto_mb library can accelerate tls in envoy by using AVX512 instructions.
// So we should check the CPUID here.
var cryptombSupported = cpuid.EnabledAVX512 && cpuid.HasFeature(
	cpuid.BMI2&
		cpuid.AVX512F&
		cpuid.AVX512DQ&
		cpuid.AVX512BW&
		cpuid.AVX512IFMA&
		cpuid.AVX512VBMI2,
)

// makeSecrets make all secrets which should be pushed to envoy.
// For now, just ROOTCA & default.
func (s *Server) makeSecrets(bundle certificate.Bundle) []types.Resource {
	results := []types.Resource{}

	rootCA := s.makeCASecret("ROOTCA", bundle.ROOTCA)
	def := s.makeSecret("default", bundle.PrivKey, bundle.CertChain, s.cryptombPollDelay)
	results = append(results, rootCA, def)
	return results
}

// makeSecret make secret object with the specified name.
// key and cryptombPollDelay are optional parameters.
// If key and cryptombPollDelay are provided, and the `cryptombSupported` is true,
// secret will use the cryptomb PrivateKeyProvider.
// See also https://www.envoyproxy.io/docs/envoy/v1.22.2/api-v3/extensions/private_key_providers/cryptomb/v3alpha/cryptomb.proto.html
func (s *Server) makeSecret(name string, key, cert []byte, cryptombPollDelay time.Duration) *envoytls.Secret {
	tlsCert := &envoytls.Secret_TlsCertificate{
		TlsCertificate: &envoytls.TlsCertificate{
			CertificateChain: &core.DataSource{
				Specifier: &core.DataSource_InlineBytes{
					InlineBytes: cert,
				},
			},
		},
	}

	if key != nil {
		if cryptombSupported && cryptombPollDelay != 0 {
			cpc := &cryptomb.CryptoMbPrivateKeyMethodConfig{
				PollDelay: durationpb.New(cryptombPollDelay),
				PrivateKey: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: key,
					},
				},
			}
			msg, _ := anypb.New(cpc)
			tlsCert.TlsCertificate.PrivateKeyProvider = &envoytls.PrivateKeyProvider{
				ProviderName: "cryptomb",
				ConfigType: &envoytls.PrivateKeyProvider_TypedConfig{
					TypedConfig: msg,
				},
			}
		} else {
			tlsCert.TlsCertificate.PrivateKey = &core.DataSource{
				Specifier: &core.DataSource_InlineBytes{
					InlineBytes: key,
				},
			}
		}
	}
	return &envoytls.Secret{
		Name: name,
		Type: tlsCert,
	}
}

func (s *Server) makeCASecret(name string, ca []byte) *envoytls.Secret {
	return &envoytls.Secret{
		Name: name,
		Type: &envoytls.Secret_ValidationContext{
			ValidationContext: &envoytls.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: ca,
					},
				},
			},
		},
	}
}
