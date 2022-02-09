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
	"reflect"
	"sort"
	"time"

	"github.com/miekg/dns"

	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
)

const name = resolver.PluginNameMeshProxy

type resolverMesh struct {
	localDNSServer *LocalDNSServer
	config         *resolverConfig
	registry       registry
	suffix         string
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
	r.registry, err = newRegistry(r.config)
	if err != nil {
		return err
	}
	r.suffix = c.Suffix
	r.localDNSServer, err = newLocalDNSServer(uint32(c.DnsTtl))
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
func (r *resolverMesh) ServeDNS(ctx context.Context, question dns.Question) *dns.Msg {
	qname, matched := resolver.MatchSuffix(question.Name, r.suffix)
	if !matched {
		return nil
	}
	question.Name = qname
	return r.localDNSServer.ServeDNS(ctx, &question)
}

func (r *resolverMesh) Start(ctx context.Context) {
	interval := time.Duration(r.config.ReloadIntervalSec) * time.Second

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		var currentServices []string
		for {
			select {
			case <-ticker.C:
				services, err := r.registry.GetCurrentNsService()
				if err != nil {
					log.Errorf("[mesh] error to get services, err: %v", err)
					continue
				}
				sort.Strings(services)
				if ifServiceListChanged(currentServices, services) {
					r.localDNSServer.UpdateLookupTable(services, r.config.DNSAnswerIp)
					currentServices = services
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func ifServiceListChanged(currentServices, newNsServices []string) bool {
	if reflect.DeepEqual(currentServices, newNsServices) {
		return false
	}
	return true
}

func init() {
	resolver.Register(&resolverMesh{})
}
