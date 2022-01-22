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

package discovery

import (
	"context"
	"github.com/miekg/dns"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
	"net"
	"strings"
)

const name = "discovery"

type resolverDiscovery struct {
	consumer api.ConsumerAPI
	suffix   string
	dnsTtl   int
	config   *resolverConfig
}

// Name will return the name to resolver
func (r *resolverDiscovery) Name() string {
	return name
}

// Initialize will init the resolver on startup
func (r *resolverDiscovery) Initialize(c *resolver.ConfigEntry) error {
	var err error
	r.config, err = parseOptions(c.Option)
	if nil != err {
		return err
	}
	r.consumer, err = api.NewConsumerAPI()
	if nil != err {
		return err
	}
	if strings.HasSuffix(c.Suffix, resolver.Quota) {
		r.suffix = c.Suffix
	} else {
		r.suffix = c.Suffix + resolver.Quota
	}
	r.dnsTtl = c.DnsTtl
	return err
}

// Start the plugin runnable
func (r *resolverDiscovery) Start(context.Context) {

}

// Destroy will destroy the resolver on shutdown
func (r *resolverDiscovery) Destroy() {
	if nil != r.consumer {
		r.consumer.Destroy()
	}
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
func (r *resolverDiscovery) ServeDNS(ctx context.Context, question dns.Question) *dns.Msg {
	if question.Qtype != dns.TypeA && question.Qtype != dns.TypeAAAA {
		return nil
	}
	qname := question.Name
	svcKey, err := resolver.ParseQname(qname, r.suffix)
	if nil != err {
		log.Errorf("[discovery] invalid qname %s, err: %v", qname, err)
		return nil
	}
	if nil == svcKey {
		return nil
	}
	request := &api.GetOneInstanceRequest{}
	request.Namespace = svcKey.Namespace
	request.Service = svcKey.Service
	if len(r.config.RouteLabels) > 0 {
		request.SourceService = &model.ServiceInfo{Metadata: r.config.RouteLabels}
	}
	resp, err := r.consumer.GetOneInstance(request)
	if nil != err {
		log.Errorf("[discovery] fail to lookup service %s, err: %v", *svcKey, err)
		return nil
	}
	instance := resp.GetInstance()
	address := net.ParseIP(instance.GetHost())

	var rr dns.RR
	if question.Qtype != dns.TypeA {
		rr = &dns.A{
			Hdr: dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(r.dnsTtl)},
			A:   address,
		}
	} else {
		rr = &dns.AAAA{
			Hdr:  dns.RR_Header{Name: qname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: uint32(r.dnsTtl)},
			AAAA: address,
		}
	}
	msg := &dns.Msg{}
	msg.Authoritative = true
	msg.Answer = append(msg.Answer, rr)
	msg.Rcode = dns.RcodeSuccess
	return msg
}

func init() {
	resolver.Register(&resolverDiscovery{})
}
