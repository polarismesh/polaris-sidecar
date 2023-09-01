/**
 * Tencent is pleased to support the open source community by making CL5 available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"

	"github.com/polarismesh/polaris-sidecar/bootstrap/config"
	"github.com/polarismesh/polaris-sidecar/envoy/metrics"
	"github.com/polarismesh/polaris-sidecar/envoy/rls"
	"github.com/polarismesh/polaris-sidecar/pkg/client"
	debughttp "github.com/polarismesh/polaris-sidecar/pkg/http"
	"github.com/polarismesh/polaris-sidecar/pkg/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
	mtlsAgent "github.com/polarismesh/polaris-sidecar/security/mtls/agent"
)

// Agent provide the listener to dns server
type Agent struct {
	config       *config.SidecarConfig
	dnsSvrs      *resolver.Server
	mtlsAgent    *mtlsAgent.Agent
	metricServer *metrics.Server
	rlsSvr       *rls.RateLimitServer

	debugSvr *http.Server
}

// Start the main agent routines
func Start(configFile string, bootConfig *config.BootConfig) {
	agent, err := newAgent(configFile, bootConfig)
	if err != nil {
		fmt.Printf("[ERROR] loadConfig fail: %v\n", err)
		os.Exit(-1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		var err error
		err = agent.Start(ctx)
		if nil != err {
			errCh <- err
		}
	}()
	runMainLoop(cancel, errCh)
}

// RunMainLoop sidecar server main loop
func runMainLoop(cancel context.CancelFunc, errCh chan error) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)
	for {
		select {
		case s := <-ch:
			log.Infof("catch signal(%+v), stop sidecar server", s)
			cancel()
			return
		case err := <-errCh:
			log.Errorf("catch sidecar server err: %s", err.Error())
			return
		}
	}
}

func newAgent(configFile string, bootConfig *config.BootConfig) (*Agent, error) {
	var err error
	polarisAgent := &Agent{}
	polarisAgent.config, err = config.ParseYamlConfig(configFile, bootConfig)
	if nil != err {
		log.Errorf("[agent] fail to parse sidecar config, err: %v", err)
		return nil, err
	}
	// 初始化日志打印
	if err := log.Configure(polarisAgent.config.Logger); err != nil {
		return nil, err
	}
	log.Infof("[agent] success to init log config")
	log.Infof("[agent] finished to parse sidecar config, current active config is \n%s", *polarisAgent.config)

	client.InitSDKContext(&client.Config{
		Addresses: polarisAgent.config.PolarisConfig.Adddresses,
		Metrics: &client.Metrics{
			Port:     polarisAgent.config.Metrics.Port,
			Type:     polarisAgent.config.Metrics.Type,
			IP:       polarisAgent.config.Bind,
			Interval: polarisAgent.config.Metrics.Interval,
			Address:  polarisAgent.config.Metrics.Address,
		},
		LocationConfigImpl: polarisAgent.config.PolarisConfig.Location,
	})

	mux := http.NewServeMux()
	polarisAgent.debugSvr = &http.Server{
		Handler: mux,
	}

	if err := polarisAgent.buildDns(configFile); err != nil {
		return nil, err
	}
	if err := polarisAgent.buildSecurity(configFile); err != nil {
		return nil, err
	}
	if err := polarisAgent.buildEnvoyMetrics(configFile); err != nil {
		return nil, err
	}
	if err := polarisAgent.buildEnvoyRls(configFile); err != nil {
		return nil, err
	}
	return polarisAgent, nil
}

func (p *Agent) buildSecurity(configFile string) error {
	if p.config.MTLS != nil && p.config.MTLS.Enable {
		log.Info("create mtls agent")
		agent, err := mtlsAgent.New(mtlsAgent.Option{
			CAServer: p.config.MTLS.CAServer,
		})
		if err != nil {
			return err
		}
		p.mtlsAgent = agent
	}
	return nil
}

func (p *Agent) buildEnvoyMetrics(configFile string) error {
	if p.config.Metrics.Enable {
		log.Infof("create metric server")
		p.metricServer = metrics.NewServer(p.config.Namespace, p.config.Metrics.Port)
	}
	return nil
}

func (p *Agent) buildEnvoyRls(configFile string) error {
	if p.config.RateLimit == nil || !p.config.RateLimit.Enable {
		return nil
	}
	log.Infof("create ratelimit server")
	conf := &rls.Config{
		Network: strings.ToLower(p.config.RateLimit.Network),
		TLSInfo: p.config.RateLimit.TLSInfo,
	}
	if conf.Network == "tcp" {
		conf.Address = fmt.Sprintf("%s:%d", p.config.Bind, p.config.RateLimit.BindPort)
	}
	rlsSvr, err := rls.New(p.config.Namespace, conf)
	if err != nil {
		return err
	}
	p.rlsSvr = rlsSvr
	return nil
}

func (p *Agent) buildDns(configFile string) error {
	svr, err := resolver.NewServers(&resolver.ResolverConfig{
		BindLocalhost: p.config.BindLocalhost(),
		BindIP:        p.config.Bind,
		BindPort:      uint32(p.config.Port),
		Recurse:       p.config.Recurse,
		Resolvers:     p.config.Resolvers,
	})
	if err != nil {
		return err
	}
	p.dnsSvrs = svr
	p.registerDebugeHandler(svr.Debugger())
	return nil
}

func (p *Agent) registerDebugeHandler(handlers []debughttp.DebugHandler) {
	mux := p.debugSvr.Handler.(*http.ServeMux)
	for i := range handlers {
		handler := handlers[i]
		if len(handler.Path) == 0 {
			continue
		}
		mux.HandleFunc(handler.Path, handler.Handler)
	}
}

// Start the agent
func (p *Agent) Start(ctx context.Context) error {
	var recvErrCounts int
	errChan := make(chan error)

	if p.config.Debugger.Enable {
		go func() {
			ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", p.config.Bind, p.config.Debugger.Port))
			if err != nil {
				errChan <- err
				return
			}

			mux := p.debugSvr.Handler.(*http.ServeMux)
			mux.HandleFunc("/sidecar/health/readiness", func(resp http.ResponseWriter, _ *http.Request) {
				resp.WriteHeader(http.StatusOK)
			})
			mux.HandleFunc("/sidecar/health/liveness", func(resp http.ResponseWriter, _ *http.Request) {
				resp.WriteHeader(http.StatusOK)
			})
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

			if err := p.debugSvr.Serve(ln); err != nil {
				errChan <- err
			}
		}()
	}
	if p.dnsSvrs != nil {
		go func() {
			log.Info("start dns server")
			errCh := p.dnsSvrs.Run(ctx)
			for {
				select {
				case <-ctx.Done():
					return
				case err := <-errCh:
					errChan <- err
				}
			}
		}()
	}
	if p.mtlsAgent != nil {
		go func() {
			log.Info("start mtls agent")
			errChan <- p.mtlsAgent.Run(ctx)
		}()
	}
	if p.metricServer != nil {
		go func() {
			log.Info("start metric server")
			err := p.metricServer.Start(ctx)
			if nil != err {
				errChan <- err
			}
		}()
	}
	if p.rlsSvr != nil {
		go func() {
			log.Info("start ratelimit server")
			err := p.rlsSvr.Run(ctx)
			if nil != err {
				errChan <- err
			}
		}()
	}

	for {
		select {
		case err := <-errChan:
			if nil != err {
				return err
			}
			recvErrCounts++
			if recvErrCounts == 2 {
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}
