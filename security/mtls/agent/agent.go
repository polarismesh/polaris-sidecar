package agent

import (
	"context"
	"net"

	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/security/mtls/certificate/caclient"
	"github.com/polarismesh/polaris-sidecar/security/mtls/certificate/manager"
	"github.com/polarismesh/polaris-sidecar/security/mtls/rotator"
	"github.com/polarismesh/polaris-sidecar/security/mtls/sds"
	"google.golang.org/grpc"
)

type Agent struct {
	network     string
	addr        string
	sds         *sds.Server
	client      manager.CSRClient
	certManager manager.Manager
	rotator     *rotator.Rotator
}

const defaultCAPath = "/etc/polaris-sidecar/certs/rootca.pem"

func New(opt Option) (*Agent, error) {
	err := opt.init()
	if err != nil {
		return nil, err
	}
	a := &Agent{}
	a.network = opt.Network
	a.addr = opt.Address
	a.rotator = rotator.New(opt.RotatePeriod, opt.FailedRetryDelay)
	a.sds = sds.New(opt.CryptombPollDelay)

	cli, err := caclient.NewWithRootCA(opt.CAServer, caclient.ServiceAccountToken(), defaultCAPath)
	if err != nil {
		return nil, err
	}
	a.client = cli

	a.certManager = manager.NewManager(opt.Namespace, opt.ServiceAccount, opt.RSAKeyBits, opt.TTL, a.client)
	return a, nil
}

func (a *Agent) Run(ctx context.Context) error {
	// start sds grpc service
	srv := grpc.NewServer()
	a.sds.Serve(srv)
	l, err := net.Listen(a.network, a.addr)
	if err != nil {
		return err
	}
	go srv.Serve(l)
	defer srv.GracefulStop()
	log.Info("start rotator")
	// start certificate generation rotator
	return a.rotator.Run(ctx, func(ctx context.Context) error {
		bundle, err := a.certManager.GetBundle(ctx)
		if err != nil {
			return err
		}
		a.sds.UpdateSecrets(ctx, *bundle)
		return nil
	})
}
