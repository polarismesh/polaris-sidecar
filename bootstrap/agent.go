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
	"github.com/polarismesh/polaris-sidecar/metrics"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/miekg/dns"

	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
	mtlsAgent "github.com/polarismesh/polaris-sidecar/security/mtls/agent"
)

// Agent provide the listener to dns server
type Agent struct {
	config       *SidecarConfig
	resolvers    []resolver.NamingResolver
	tcpServer    *dns.Server
	udpServer    *dns.Server
	mtlsAgent    *mtlsAgent.Agent
	metricServer *metrics.Server
}

// Start the main agent routines
func Start(configFile string, bootConfig *BootConfig) {
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

const (
	etcResolvConfPath = "/etc/resolv.conf"
)

func parseResolvConf(bindLocalhost bool) ([]string, []string) {
	if !isFile(etcResolvConfPath) {
		return nil, nil
	}
	dnsConfig, err := dns.ClientConfigFromFile(etcResolvConfPath)
	if err != nil {
		log.Errorf("[agent] failed to load /etc/resolv.conf: %v", err)
		return nil, nil
	}
	var searchNames []string
	var nameservers []string
	if dnsConfig != nil {
		for _, search := range dnsConfig.Search {
			searchNames = append(searchNames, search+".")
		}

		for _, server := range dnsConfig.Servers {
			if server == "127.0.0.1" && bindLocalhost {
				continue
			}
			nameservers = append(nameservers, server)
		}
	}
	return nameservers, searchNames
}

func newAgent(configFile string, bootConfig *BootConfig) (*Agent, error) {
	var err error
	polarisAgent := &Agent{}
	polarisAgent.config, err = parseYamlConfig(configFile, bootConfig)
	if nil != err {
		log.Errorf("[agent] fail to parse sidecar config, err: %v", err)
		return nil, err
	}
	nameservers, searchNames := parseResolvConf(polarisAgent.config.bindLocalhost())
	log.Infof("[agent] finished to parse /etc/resolv.conf, nameservers %s, search %s", nameservers, searchNames)
	if len(polarisAgent.config.Recurse.NameServers) == 0 {
		polarisAgent.config.Recurse.NameServers = nameservers
	}
	log.Infof("[agent] finished to parse sidecar config, current active config is %s", *polarisAgent.config)
	// 初始化日志打印
	err = log.Configure(polarisAgent.config.Logger)
	log.Infof("[agent] success to init log config")
	if err != nil {
		return nil, err
	}
	for _, resolverCfg := range polarisAgent.config.Resolvers {
		if !resolverCfg.Enable {
			log.Infof("[agent] resolver %s is not enabled", resolverCfg.Name)
			continue
		}
		name := resolverCfg.Name
		handler := resolver.NameResolver(name)
		if nil == handler {
			log.Errorf("[agent] resolver %s is not found", resolverCfg.Name)
			return nil, fmt.Errorf("fail to lookup resolver %s, consider it's not registered", name)
		}
		err = handler.Initialize(resolverCfg)
		if nil != err {
			for _, initHandler := range polarisAgent.resolvers {
				initHandler.Destroy()
			}
			log.Errorf("[agent] fail to init resolver %s, err: %v", resolverCfg.Name, err)
			return nil, err
		}
		log.Infof("[agent] finished to init resolver %s", resolverCfg.Name)
		polarisAgent.resolvers = append(polarisAgent.resolvers, handler)
	}
	recurseAddresses := make([]string, 0, len(polarisAgent.config.Recurse.NameServers))
	for _, nameserver := range polarisAgent.config.Recurse.NameServers {
		recurseAddresses = append(recurseAddresses, fmt.Sprintf("%s:53", nameserver))
	}
	polarisAgent.udpServer = &dns.Server{
		Addr: polarisAgent.config.Bind + ":" + strconv.Itoa(polarisAgent.config.Port), Net: "udp",
	}
	polarisAgent.udpServer.Handler = &dnsHandler{
		protocol:        "udp",
		resolvers:       polarisAgent.resolvers,
		searchNames:     searchNames,
		recursorTimeout: time.Duration(polarisAgent.config.Recurse.TimeoutSec) * time.Second,
		recursors:       recurseAddresses,
		recurseEnable:   polarisAgent.config.Recurse.Enable,
	}
	polarisAgent.tcpServer = &dns.Server{
		Addr: polarisAgent.config.Bind + ":" + strconv.Itoa(polarisAgent.config.Port), Net: "tcp",
	}
	polarisAgent.tcpServer.Handler = &dnsHandler{
		protocol:        "tcp",
		resolvers:       polarisAgent.resolvers,
		searchNames:     searchNames,
		recursorTimeout: time.Duration(polarisAgent.config.Recurse.TimeoutSec) * time.Second,
		recursors:       recurseAddresses,
		recurseEnable:   polarisAgent.config.Recurse.Enable,
	}
	if polarisAgent.config.MTLS != nil && polarisAgent.config.MTLS.Enable {
		log.Info("create mtls agent")
		agent, err := mtlsAgent.New(mtlsAgent.Option{
			CAServer: polarisAgent.config.MTLS.CAServer,
		})
		if err != nil {
			return nil, err
		}
		polarisAgent.mtlsAgent = agent
	}
	if polarisAgent.config.Metrics.Enable {
		log.Infof("create metric server")
		polarisAgent.metricServer = metrics.NewServer(polarisAgent.config.Namespace, polarisAgent.config.Metrics.Port)
	}
	return polarisAgent, nil
}

// Start the agent
func (p *Agent) Start(ctx context.Context) error {
	for _, handler := range p.resolvers {
		handler.Start(ctx)
		log.Infof("[agent] success to start resolver %s", handler.Name())
	}
	errChan := make(chan error)
	go func() {
		errChan <- p.tcpServer.ListenAndServe()
	}()
	go func() {
		errChan <- p.udpServer.ListenAndServe()
	}()
	var recvErrCounts int
	defer func() {
		for _, handler := range p.resolvers {
			handler.Destroy()
		}
	}()

	if p.mtlsAgent != nil {
		mCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			log.Info("start mtls agent")
			errChan <- p.mtlsAgent.Run(mCtx)
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
