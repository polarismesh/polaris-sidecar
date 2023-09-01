/**
 * Tencent is pleased to support the open source community by making Polaris available.
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

package meshproxy

import (
	"context"
	"strings"
	"time"

	"github.com/polarismesh/polaris-go"

	"github.com/miekg/dns"

	"github.com/polarismesh/polaris-sidecar/pkg/client"
	debughttp "github.com/polarismesh/polaris-sidecar/pkg/http"
	"github.com/polarismesh/polaris-sidecar/pkg/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
)

const name = resolver.PluginNameMeshProxy

type resolverMesh struct {
	localDNSServer *LocalDNSServer
	config         *resolverConfig
	registry       registry
	suffix         string
	consumer       polaris.ConsumerAPI
}

// Name will return the name to resolver
func (r *resolverMesh) Name() string {
	return name
}

// Initialize will init the resolver on startup
func (r *resolverMesh) Initialize(c *resolver.ConfigEntry) error {
	var err error
	r.config, err = parseOptions(c.Option)
	if nil != err {
		return err
	}
	r.config.Namespace = c.Namespace
	r.consumer, err = client.GetConsumerAPI()
	if nil != err {
		return err
	}
	r.registry, err = newRegistry(r.config, r.consumer, r.config.FilterByBusiness)
	if err != nil {
		return err
	}
	r.suffix = c.Suffix
	r.localDNSServer, err = newLocalDNSServer(uint32(c.DnsTtl), r.config.RecursionAvailable)
	if nil != err {
		return err
	}
	return err
}

// Destroy will destroy the resolver on shutdown
func (r *resolverMesh) Destroy() {

}

// ServeDNS is like dns.Handler except ServeDNS may return an rcode
// and/or error.
// If ServeDNS writes to the response body, it should return a status
// code. Resolvers assumes *no* reply has yet been written if the status
// code is one of the following:
//
// * SERVFAIL (dns.RcodeServerFailure)
//
// * REFUSED (dns.RecodeRefused)
//
// * NOTIMP (dns.RcodeNotImplemented)
func (r *resolverMesh) ServeDNS(ctx context.Context, question dns.Question, qname string) *dns.Msg {
	_, matched := resolver.MatchSuffix(qname, r.suffix)
	if !matched {
		log.Infof("[Mesh] suffix not matched for name %s, suffix %s", qname, r.suffix)
		return nil
	}
	ret := r.localDNSServer.ServeDNS(ctx, &question, qname)
	if ret != nil {
		return ret
	}
	// 可能这个时候 qname 只有服务名称，这里手动补充 Namespace 信息
	if strings.HasSuffix(qname, resolver.Quota) {
		qname = qname[0 : len(qname)-1]
	}
	qname = qname + "." + r.config.Namespace + "."
	ret = r.localDNSServer.ServeDNS(ctx, &question, qname)
	if ret == nil {
		log.Infof("[Mesh] host not found for name %s", qname)
	}
	return ret
}

func (r *resolverMesh) Start(ctx context.Context) {
	interval := time.Duration(r.config.ReloadIntervalSec) * time.Second

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		var currentServices map[string]struct{}
		var nextServices map[string]struct{}
		var changed bool

		nextServices, changed = r.doReload(currentServices)
		if changed {
			currentServices = nextServices
		}
		for {
			select {
			case <-ticker.C:
				nextServices, changed = r.doReload(currentServices)
				if changed {
					currentServices = nextServices
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (r *resolverMesh) Debugger() []debughttp.DebugHandler {
	return []debughttp.DebugHandler{}
}

func (r *resolverMesh) doReload(currentServices map[string]struct{}) (map[string]struct{}, bool) {
	services, err := r.registry.GetCurrentNsService()
	if err != nil {
		log.Errorf("[mesh] error to get services, err: %v", err)
		return nil, false
	}
	if ifServiceListChanged(currentServices, services) {
		r.localDNSServer.UpdateLookupTable(services, r.config.DNSAnswerIp)
		return services, true
	}
	return nil, false
}

func ifServiceListChanged(currentServices, newNsServices map[string]struct{}) bool {
	if len(currentServices) != len(newNsServices) {
		return true
	}
	if len(currentServices) == 0 {
		return false
	}
	for svc := range currentServices {
		if _, ok := newNsServices[svc]; !ok {
			return true
		}
	}
	return false
}

func init() {
	resolver.Register(&resolverMesh{})
}
