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

package dnsagent

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/pkg/model"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris-sidecar/log"
	"github.com/polarismesh/polaris-sidecar/resolver"
)

func init() {
	resolver.Register(&resolverDiscovery{})
}

const name = resolver.PluginNameDnsAgent

type resolverDiscovery struct {
	consumer polaris.ConsumerAPI
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
	r.consumer, err = polaris.NewConsumerAPI()
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

func canDoResolve(qType uint16) bool {
	if qType == dns.TypeA {
		return true
	}
	if qType == dns.TypeAAAA {
		return true
	}
	if qType == dns.TypeSRV {
		return true
	}

	return false
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
	if !canDoResolve(question.Qtype) {
		return nil
	}

	msg := &dns.Msg{}
	qname := question.Name

	labels := dns.SplitDomainName(qname)
	for i := range labels {
		if labels[i] == "_addr" {
			ret, err := hex.DecodeString(labels[i-1])
			if err != nil {
				log.Error("decode ip str fail", zap.String("domain", qname), zap.Error(err))
				return nil
			}
			rr := r.markRecord(question, net.IP(ret), nil)
			msg.Answer = append(msg.Answer, rr)
			return msg
		}
	}

	instances, err := r.lookupFromPolaris(question)
	if err != nil {
		return nil
	}

	//do reorder and unique
	for i := range instances {
		ins := instances[i]
		rr := r.markRecord(question, net.ParseIP(ins.GetHost()), ins)
		msg.Answer = append(msg.Answer, rr)
	}

	msg.Authoritative = true
	msg.Rcode = dns.RcodeSuccess

	msg = resolver.TrimDNSResponse(ctx, msg)
	return msg
}

func (r *resolverDiscovery) lookupFromPolaris(question dns.Question) ([]model.Instance, error) {
	svcKey, err := resolver.ParseQname(question.Qtype, question.Name, r.suffix)
	if nil != err {
		log.Errorf("[discovery] invalid qname %s, err: %v", &question, err)
		return nil, nil
	}
	if nil == svcKey {
		return nil, nil
	}
	request := &polaris.GetOneInstanceRequest{}
	request.Namespace = svcKey.Namespace
	request.Service = svcKey.Service
	if len(r.config.RouteLabels) > 0 {
		request.SourceService = &model.ServiceInfo{Metadata: r.config.RouteLabels}
	}
	resp, err := r.consumer.GetOneInstance(request)
	if nil != err {
		log.Errorf("[discovery] fail to lookup service %s, err: %v", *svcKey, err)
		return nil, nil
	}

	return resp.GetInstances(), nil
}

func encodeIPAsFqdn(ip net.IP, svcKey model.ServiceKey) string {
	respDomain := fmt.Sprintf("%s._addr.%s.%s", hex.EncodeToString(ip), svcKey.Service, svcKey.Namespace)
	return dns.Fqdn(respDomain)
}

func (r *resolverDiscovery) markRecord(question dns.Question, address net.IP, ins model.Instance) dns.RR {

	var rr dns.RR

	qname := question.Name

	switch question.Qtype {
	case dns.TypeA:
		rr = &dns.A{
			Hdr: dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(r.dnsTtl)},
			A:   address,
		}
	case dns.TypeSRV:
		if ins == nil {
			return rr
		}

		rr = &dns.SRV{
			Hdr:      dns.RR_Header{Name: qname, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: uint32(r.dnsTtl)},
			Priority: uint16(ins.GetPriority()),
			Weight:   uint16(ins.GetWeight()),
			Port:     uint16(ins.GetPort()),
			Target:   encodeIPAsFqdn(address, ins.GetInstanceKey().ServiceKey),
		}
	case dns.TypeAAAA:
		rr = &dns.AAAA{
			Hdr:  dns.RR_Header{Name: qname, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: uint32(r.dnsTtl)},
			AAAA: address,
		}
	}
	return rr
}
