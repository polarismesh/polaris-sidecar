package sds

import (
	"context"
	"strconv"
	"time"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	secretv3 "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"google.golang.org/grpc"

	"github.com/polarismesh/polaris-sidecar/pkg/log"
	"github.com/polarismesh/polaris-sidecar/security/mtls/certificate"
)

type Server struct {
	srv               serverv3.Server
	snap              cache.SnapshotCache
	cryptombPollDelay time.Duration
}

func New(cryptombPollDelay time.Duration) (s *Server) {
	s = &Server{}
	s.cryptombPollDelay = cryptombPollDelay

	snap := cache.NewSnapshotCache(false, defaultHash, nil)
	srv := serverv3.NewServer(context.TODO(), snap, nil)
	s.srv = srv
	s.snap = snap

	return s
}

func (s *Server) UpdateSecrets(ctx context.Context, bundle certificate.Bundle) {
	version := strconv.Itoa(int(time.Now().UnixNano()))

	log.Infof("update secrets: %s", version)

	resources := make(map[resource.Type][]types.Resource)
	resources[resource.SecretType] = s.makeSecrets(bundle)
	snapshot, _ := cache.NewSnapshot(version, resources)
	s.snap.SetSnapshot(ctx, string(defaultHash), snapshot)
}

func (s *Server) Serve(srv *grpc.Server) {
	secretv3.RegisterSecretDiscoveryServiceServer(srv, s.srv)
}

var defaultHash = ConstHash("default")

// ConstHash uses a const string as the node hash.
type ConstHash string

// ID uses the string
func (h ConstHash) ID(node *core.Node) string {
	return string(h)
}

var _ cache.NodeHash = ConstHash("")
